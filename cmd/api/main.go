package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
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

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// Подключение к базе данных
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer database.Close()

	// Инициализация метрик
	metrics.InitMetrics()

	// Инициализация обработчиков, передаются зависимые объекты (db, config)
	h := handlers.NewHandler(database, cfg)

	// Создаём основной роутер
	r := mux.NewRouter()

	// Публичные эндпоинты
	r.HandleFunc("/dummyLogin", h.DummyLogin).Methods("POST")
	r.HandleFunc("/register", h.Register).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")

	// Защищённые эндпоинты (требуется аутентификация)
	api := r.PathPrefix("/").Subrouter()
	api.Use(h.AuthMiddleware)

	// Эндпоинты PVZ
	api.HandleFunc("/pvz", h.CreatePVZ).Methods("POST")
	api.HandleFunc("/pvz", h.GetPVZList).Methods("GET")
	api.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	api.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")

	// Эндпоинт для создания приёмки
	api.HandleFunc("/receptions", h.CreateReception).Methods("POST")

	// Эндпоинт для добавления товара
	api.HandleFunc("/products", h.AddProduct).Methods("POST")

	// Запуск HTTP-сервера API на порту 8080
	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Println("HTTP API server starting on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	// Запуск сервера метрик Prometheus на порту 9000
	go func() {
		log.Println("Prometheus metrics server starting on port 9000")
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9000", nil); err != nil {
			log.Fatalf("Metrics server ListenAndServe: %v", err)
		}
	}()

	// Запуск gRPC-сервера на порту 3000
	grpcLis, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	// Регистрация gRPC-сервиса PVZ, передаём соединение с базой
	pvzService := handlers.NewPVZService(database)
	pb.RegisterPVZServiceServer(grpcServer, pvzService)
	go func() {
		log.Println("gRPC server starting on port 3000")
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Грейсфул-шифтдаун
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}
	grpcServer.GracefulStop()
}
