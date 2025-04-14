package main

import (
	"AvitoPVZ/internal/repository"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"AvitoPVZ/internal/config"
	"AvitoPVZ/internal/db"
	"AvitoPVZ/internal/handlers"
	"AvitoPVZ/internal/metrics"
	pb "AvitoPVZ/proto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// loggingResponseWriter перехватывает status code для логирования.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK} // По умолчанию 200 OK
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write вызывается неявно, если WriteHeader не был вызван.
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	return lrw.ResponseWriter.Write(b)
}

// setupLogger настраивает глобальный логгер slog на основе переменных окружения.
func setupLogger() *slog.Logger {
	logLevelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	logLevel := slog.LevelInfo // Уровень по умолчанию

	switch logLevelStr {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN", "WARNING":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		if logLevelStr != "" {
			fmt.Fprintf(os.Stderr, "Invalid LOG_LEVEL '%s'. Defaulting to INFO\n", logLevelStr)
		}
	}

	logFormat := strings.ToLower(os.Getenv("LOG_FORMAT"))
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel <= slog.LevelDebug,
	}

	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
		fmt.Fprintln(os.Stderr, "Using JSON log format")
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
		fmt.Fprintln(os.Stderr, "Using Text log format")
	}

	logger := slog.New(handler)
	slog.SetDefault(logger) //  логгер по умолчанию
	return logger
}

func main() {
	logger := setupLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("cannot load config", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	database, err := db.Connect(cfg, sql.Open)
	if err != nil {
		logger.Error("cannot connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("Database connection established")

	metrics.InitMetrics()
	logger.Info("Prometheus metrics initialized")

	h := handlers.NewHandler(database, cfg, logger)

	//  Основной роутер API (порт 8080)
	r := mux.NewRouter()

	// Middleware для логирования запросов
	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := NewLoggingResponseWriter(w)
			next.ServeHTTP(lrw, r)
			duration := time.Since(start)
			logger.Info("HTTP API Request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("status_code", lrw.statusCode),
				slog.Duration("duration", duration),
			)
		})
	}
	r.Use(loggingMiddleware)

	// Публичные эндпоинты
	r.HandleFunc("/dummyLogin", h.DummyLogin).Methods("POST")
	r.HandleFunc("/register", h.Register).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")

	// Защищённые эндпоинты
	api := r.PathPrefix("/").Subrouter()
	api.Use(h.AuthMiddleware)

	api.HandleFunc("/pvz", h.CreatePVZ).Methods("POST")
	api.HandleFunc("/pvz", h.GetPVZList).Methods("GET")
	api.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	api.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")
	api.HandleFunc("/receptions", h.CreateReception).Methods("POST")
	api.HandleFunc("/products", h.AddProduct).Methods("POST")

	//  Отдельный сервер для метрик Prometheus (порт 9000)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsSrv := &http.Server{
		Addr:         ":9000",
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Metrics server starting", slog.String("address", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server ListenAndServe error", slog.Any("error", err))
			// Не фатальная ошибка, приложение продолжит работу
		}
	}()

	//  Запуск основного HTTP-сервера API (порт 8080)
	apiSrv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		// TODO: Рассмотреть возможность установки slog.ErrorLog()
		// apiSrv.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
	}

	go func() {
		logger.Info("HTTP API server starting", slog.String("address", apiSrv.Addr))
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP API server ListenAndServe error", slog.Any("error", err))
			os.Exit(1) // Ошибка основного сервера
		}
	}()

	//  Запуск gRPC-сервера (порт 3000)
	grpcAddr := ":3000"
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("gRPC failed to listen", slog.String("address", grpcAddr), slog.Any("error", err))
		os.Exit(1)
	}
	// TODO: Добавить gRPC interceptors для логирования/метрик/обработки ошибок
	grpcServer := grpc.NewServer()

	pvzRepository := repository.NewPVZRepository(database)
	pvzService := handlers.NewPVZService(pvzRepository, logger)
	pb.RegisterPVZServiceServer(grpcServer, pvzService)

	go func() {
		logger.Info("gRPC server starting", slog.String("address", grpcAddr))
		if err := grpcServer.Serve(grpcLis); err != nil {
			logger.Error("gRPC server failed", slog.Any("error", err))
			// Не вызываем os.Exit(1), позволяем другим серверам завершиться
		}
	}()

	// шатдаун
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	logger.Info("Shutdown signal received", slog.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()

	// Завершаем HTTP API сервер
	logger.Info("Shutting down HTTP API server...")
	if err := apiSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP API server Shutdown error", slog.Any("error", err))
	} else {
		logger.Info("HTTP API server gracefully stopped")
	}

	// Завершаем сервер метрик
	logger.Info("Shutting down Metrics server...")
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Metrics server Shutdown error", slog.Any("error", err))
	} else {
		logger.Info("Metrics server gracefully stopped")
	}

	// Завершаем gRPC сервер
	logger.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop() // Этот метод блокирует до завершения
	logger.Info("gRPC server gracefully stopped")

	logger.Info("Application shut down complete.")
}

var _ io.Writer = io.Discard
