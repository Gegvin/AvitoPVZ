package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"AvitoPVZ/internal/config"
	"AvitoPVZ/internal/metrics"
	"AvitoPVZ/internal/models"
	"AvitoPVZ/internal/repository"
	"AvitoPVZ/internal/utils"
	"AvitoPVZ/pkg/api"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "AvitoPVZ/proto"
)

// Определяем уникальный тип ключей для context, чтобы избежать конфликтов
type contextKey string

// UserKey используется для хранения информации о пользователе в context.

const UserKey contextKey = "user"

type Handler struct {
	db              *sql.DB
	cfg             *config.Config
	logger          *slog.Logger
	userRepo        repository.UserRepository
	pvzRepo         repository.PVZRepository
	receptionRepo   repository.ReceptionRepository
	productRepo     repository.ProductRepository
	productTypeRepo repository.ProductTypeRepository
}

// NewHandler инициализирует все репозитории
func NewHandler(db *sql.DB, cfg *config.Config, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		db:              db,
		cfg:             cfg,
		logger:          logger.With(slog.String("component", "http_handler")),
		userRepo:        repository.NewUserRepository(db),
		pvzRepo:         repository.NewPVZRepository(db),
		receptionRepo:   repository.NewReceptionRepository(db),
		productRepo:     repository.NewProductRepository(db),
		productTypeRepo: repository.NewProductTypeRepository(db),
	}
}

// отправляет ответ в формате JSON.
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		slog.Error("JSON marshal error in respondJSON", slog.Any("error", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Message: "Internal Server Error"}) // Используем api.Error
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// respondError - хелпер для отправки стандартизированных ошибок
func respondError(w http.ResponseWriter, logger *slog.Logger, status int, message string, err error) {
	logLevel := slog.LevelError
	if status < 500 && status >= 400 {
		logLevel = slog.LevelWarn
	}

	errToLog := err
	if errToLog == nil && status >= 500 { // Log message as error for 5xx if no underlying error
		errToLog = errors.New(message)
	}
	logger.Log(context.Background(), logLevel, message, slog.Any("error", errToLog), slog.Int("status", status))
	respondJSON(w, status, api.Error{Message: message}) // Используем api.Error
}

// AuthMiddleware – HTTP middleware для проверки заголовка Authorization.

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqLogger := h.logger.With(slog.String("middleware", "AuthMiddleware"))
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			reqLogger.Warn("Missing auth header")
			respondJSON(w, http.StatusUnauthorized, api.Error{Message: "missing auth header"}) // DTO
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			reqLogger.Warn("Invalid auth header format")
			respondJSON(w, http.StatusUnauthorized, api.Error{Message: "invalid auth header"}) // DTO
			return
		}
		tokenStr := parts[1]
		claims, err := utils.ParseToken(tokenStr, h.cfg.JWTSecret)
		if err != nil {
			reqLogger.Warn("Invalid token", slog.Any("error", err))
			respondJSON(w, http.StatusUnauthorized, api.Error{Message: "invalid token"}) // DTO
			return
		}

		userID, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)
		reqLogger.Debug("Token validated", slog.String("user_id", userID), slog.String("role", role))

		ctx := context.WithValue(r.Context(), UserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DummyLogin – выдаёт тестовый токен для указанной роли.

func (h *Handler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "DummyLogin"))
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	logger.Debug("Request received", slog.String("role", req.Role))

	if req.Role != "employee" && req.Role != "moderator" {
		respondError(w, logger, http.StatusBadRequest, "Invalid role specified", nil)
		return
	}
	tempUserID := uuid.New().String()
	token, err := utils.GenerateToken(tempUserID, req.Role, h.cfg.JWTSecret)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Token generation error", err)
		return
	}
	logger.Info("Dummy token generated", slog.String("role", req.Role), slog.String("user_id", tempUserID))
	respondJSON(w, http.StatusOK, api.TokenResponse{Token: &token})
}

// Register – ИСПОЛЬЗУЕТ DTO
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "Register"))
	var req api.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	// Преобразуем типы DTO к string для использования внутри
	emailStr := string(req.Email)
	roleStr := string(req.Role)
	logger.Debug("Request body decoded", slog.String("email", emailStr), slog.String("role", roleStr))

	// Валидация роли
	if roleStr != "employee" && roleStr != "moderator" {
		respondError(w, logger, http.StatusBadRequest, "Invalid role specified", nil)
		return
	}

	hashedPass, err := utils.HashPassword(req.Password)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Password hashing error", err)
		return
	}
	// Создаем модель
	userID := uuid.New() // Генерируем UUID сразу
	user := &models.User{
		ID:       userID.String(), // Сохраняем как строку в модели
		Email:    emailStr,
		Password: hashedPass,
		Role:     roleStr,
	}
	err = h.userRepo.CreateUser(user)
	if err != nil {
		// Проверяем на ошибку дубликата
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") && strings.Contains(err.Error(), "users_email_key") {
			respondError(w, logger, http.StatusConflict, "Email already exists", err)
		} else {
			respondError(w, logger, http.StatusInternalServerError, "User creation failed", err)
		}
		return
	}
	logger.Info("User registered successfully", slog.String("user_id", user.ID), slog.String("email", user.Email))

	// Создаем DTO для ответа
	emailDto := types.Email(user.Email)
	roleDto := api.UserResponseRole(user.Role)
	idForDto := userID

	responseUser := api.UserResponse{
		Id:    &idForDto,
		Email: &emailDto,
		Role:  &roleDto,
	}

	respondJSON(w, http.StatusCreated, responseUser)
}

// Login – ИСПОЛЬЗУЕТ DTO
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "Login"))
	var req api.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	// Преобразуем к string для использования внутри
	emailStr := string(req.Email) //
	logger.Debug("Login attempt", slog.String("email", emailStr))

	// Используем модель
	user, err := h.userRepo.GetUserByEmail(emailStr)
	if err != nil {
		errMsg := "User not found or invalid credentials"
		if errors.Is(err, sql.ErrNoRows) {

			logger.Warn(errMsg, slog.String("email", emailStr))
			respondError(w, logger, http.StatusUnauthorized, errMsg, nil) // Не передаем sql.ErrNoRows клиенту
		} else {

			respondError(w, logger, http.StatusInternalServerError, "Database error during login", err)
		}
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		logger.Warn("Invalid password attempt", slog.String("email", user.Email), slog.String("user_id", user.ID))
		respondError(w, logger, http.StatusUnauthorized, "User not found or invalid credentials", nil)
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Role, h.cfg.JWTSecret)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Token generation error", err)
		return
	}
	logger.Info("User logged in successfully", slog.String("user_id", user.ID), slog.String("email", user.Email))

	// Создаем DTO для ответа
	respondJSON(w, http.StatusOK, api.TokenResponse{Token: &token})
}

// CreatePVZ – создание нового ПВЗ.

func (h *Handler) CreatePVZ(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "CreatePVZ"))
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))
	if role != "moderator" {
		respondError(w, logger, http.StatusForbidden, "Access denied: insufficient privileges", nil)
		return
	}

	// Ожидаем JSON с полем "city", содержащим название города
	var req struct {
		City string `json:"city"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body format", err)
		return
	}
	logger.Debug("Request body decoded", slog.String("city_name", req.City))

	if req.City == "" {
		respondError(w, logger, http.StatusBadRequest, "City name cannot be empty", nil)
		return
	}

	// Проверка и получение ID города из репозитория
	cityID, err := h.pvzRepo.GetCityIDByName(req.City)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("Attempt to create PVZ in disallowed city", slog.String("city_name", req.City))
			respondError(w, logger, http.StatusBadRequest, "City not allowed or does not exist", nil) // Не передаем ошибку БД клиенту
		} else {
			// Логируем как Error для других ошибок БД
			respondError(w, logger, http.StatusInternalServerError, "Failed to validate city", err)
		}
		return
	}
	logger.Debug("City validated", slog.String("city_name", req.City), slog.Int("city_id", cityID))

	// Создаем модель PVZ с CityID
	pvzModel := models.PVZ{
		ID:               uuid.New().String(),
		RegistrationDate: time.Now(),
		CityID:           cityID,

		CityName: req.City,
	}

	// Создаем ПВЗ в репозитории
	err = h.pvzRepo.CreatePVZ(&pvzModel)
	if err != nil {

		respondError(w, logger, http.StatusInternalServerError, "Failed to create PVZ", err)
		return
	}

	metrics.IncPVZCreated()
	logger.Info("PVZ created successfully", slog.String("pvz_id", pvzModel.ID), slog.String("city_name", pvzModel.CityName), slog.Int("city_id", pvzModel.CityID))

	// Отправляем ответ
	respondJSON(w, http.StatusCreated, pvzModel)
}

func (h *Handler) GetPVZList(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "GetPVZList"))

	// Проверка роли (employee или moderator)
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	if role != "employee" && role != "moderator" {
		respondError(w, logger, http.StatusForbidden, "Access denied", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))

	//  Парсинг параметров пагинации и фильтрации
	params := r.URL.Query() // Получаем все query параметры
	logger.Info("Fetching PVZ list", slog.String("query", r.URL.RawQuery))

	// Вызов репозитория с параметрами

	pvzs, err := h.pvzRepo.GetPVZList(params)
	if err != nil {
		// Обрабатываем возможные ошибки парсинга из репозитория как Bad Request
		if strings.Contains(err.Error(), "invalid parameter") { // Пример проверки текста ошибки
			respondError(w, logger, http.StatusBadRequest, err.Error(), err)
		} else {
			respondError(w, logger, http.StatusInternalServerError, "Failed to get PVZ list", err)
		}
		return
	}

	logger.Info("PVZ list retrieved successfully", slog.Int("count", len(pvzs)))
	// Отправляем список моделей
	respondJSON(w, http.StatusOK, pvzs)
}

// CreateReception – создание новой приёмки товаров (только для сотрудников ПВЗ).
func (h *Handler) CreateReception(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "CreateReception"))
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))
	if role != "employee" {
		respondError(w, logger, http.StatusForbidden, "Access denied: employee required", nil)
		return
	}
	var req struct {
		PVZID string `json:"pvzId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	logger.Debug("Request body decoded", slog.String("pvz_id", req.PVZID))
	exists, err := h.receptionRepo.OpenReceptionExists(req.PVZID)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Error checking reception status", err)
		return
	}
	if exists {
		respondError(w, logger, http.StatusBadRequest, "Open reception already exists for this PVZ", nil)
		return
	}
	reception := &models.Reception{ID: uuid.New().String(), DateTime: time.Now(), PVZID: req.PVZID, Status: "in_progress"}
	if err := h.receptionRepo.CreateReception(reception); err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Failed to create reception", err)
		return
	}
	metrics.IncReceptionCreated()
	logger.Info("Reception created successfully", slog.String("reception_id", reception.ID), slog.String("pvz_id", reception.PVZID))
	respondJSON(w, http.StatusCreated, reception)
}

// добавление товара в текущую приёмку

func (h *Handler) AddProduct(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "AddProduct"))
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))
	if role != "employee" {
		respondError(w, logger, http.StatusForbidden, "Access denied: employee required", nil)
		return
	}

	var req struct {
		Type  string `json:"type"`
		PVZID string `json:"pvzId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, logger, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	logger.Debug("Request body decoded", slog.String("pvz_id", req.PVZID), slog.String("product_type_name", req.Type))

	if req.Type == "" {
		respondError(w, logger, http.StatusBadRequest, "Product type name cannot be empty", nil)
		return
	}
	typeID, err := h.productTypeRepo.GetProductTypeIDByName(req.Type)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("Invalid product type received", slog.String("type_name", req.Type))

			respondError(w, logger, http.StatusBadRequest, "Invalid or unsupported product type", nil)
		} else {

			respondError(w, logger, http.StatusInternalServerError, "Failed to validate product type", err)
		}
		return
	}
	logger.Debug("Product type validated", slog.String("type_name", req.Type), slog.Int("type_id", typeID))

	// Получаем открытую приемку
	reception, err := h.receptionRepo.GetOpenReception(req.PVZID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, logger, http.StatusBadRequest, "No open reception found for this PVZ", nil)
		} else {
			respondError(w, logger, http.StatusInternalServerError, "Error checking reception status", err)
		}
		return
	}
	logger = logger.With(slog.String("reception_id", reception.ID))

	// Создаем модель продукта с TypeID
	product := &models.Product{
		ID:          uuid.New().String(),
		DateTime:    time.Now(),
		ReceptionID: reception.ID,
		TypeID:      typeID,
	}

	// Добавляем продукт в репозиторий
	err = h.productRepo.AddProduct(product)
	if err != nil {
		respondError(w, logger, http.StatusInternalServerError, "Failed to add product", err)
		return
	}
	metrics.IncProductAdded()

	logger.Info("Product added successfully", slog.String("product_id", product.ID), slog.String("product_type_name", req.Type), slog.Int("type_id", product.TypeID))

	product.TypeName = req.Type
	respondJSON(w, http.StatusCreated, product)
}

// DeleteLastProduct – удаление последнего добавленного товара (LIFO) из текущей приёмки.
func (h *Handler) DeleteLastProduct(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "DeleteLastProduct"))
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))
	if role != "employee" {
		respondError(w, logger, http.StatusForbidden, "Access denied: employee required", nil)
		return
	}
	vars := mux.Vars(r)
	pvzId, pvzOk := vars["pvzId"]
	if !pvzOk {
		respondError(w, logger, http.StatusBadRequest, "Bad request: Missing pvzId in path", nil)
		return
	}
	logger = logger.With(slog.String("pvz_id", pvzId))
	logger.Info("Attempting to delete last product")
	err := h.productRepo.DeleteLastProduct(pvzId)
	if err != nil {
		// Используем кастомную ошибку из репозитория
		if err.Error() == "no product to delete" { // Сравниваем текст ошибки (или лучше определить типизированную ошибку)
			respondError(w, logger, http.StatusBadRequest, err.Error(), nil) // Передаем сообщение как есть
		} else {
			respondError(w, logger, http.StatusInternalServerError, "Failed to delete product", err)
		}
		return
	}
	logger.Info("Last product deleted successfully")
	respondJSON(w, http.StatusOK, map[string]string{"message": "Product deleted"})
}

// CloseLastReception – закрытие последней открытой приёмки в рамках ПВЗ.
func (h *Handler) CloseLastReception(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(slog.String("handler", "CloseLastReception"))
	claims, role, userID := getUserClaimsFromContext(r, logger)
	if claims == nil {
		respondError(w, logger, http.StatusInternalServerError, "Internal error: failed to get user claims", nil)
		return
	}
	logger = logger.With(slog.String("user_id", userID), slog.String("role", role))
	if role != "employee" {
		respondError(w, logger, http.StatusForbidden, "Access denied: employee required", nil)
		return
	}
	vars := mux.Vars(r)
	pvzId, pvzOk := vars["pvzId"]
	if !pvzOk {
		respondError(w, logger, http.StatusBadRequest, "Bad request: Missing pvzId in path", nil)
		return
	}
	logger = logger.With(slog.String("pvz_id", pvzId))
	logger.Info("Attempting to close last reception")
	reception, err := h.receptionRepo.GetOpenReception(pvzId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, logger, http.StatusBadRequest, "No open reception found for this PVZ", nil)
		} else {
			respondError(w, logger, http.StatusInternalServerError, "Error checking reception status", err)
		}
		return
	}
	logger = logger.With(slog.String("reception_id", reception.ID))
	if reception.Status != "in_progress" {
		respondError(w, logger, http.StatusBadRequest, "Reception is not in progress", nil)
		return
	}
	err = h.receptionRepo.CloseReception(reception.ID)
	if err != nil {
		// Проверяем кастомную ошибку репозитория
		if err.Error() == "no reception updated" {

			respondError(w, logger, http.StatusBadRequest, "Reception not found or already closed", nil)
		} else {
			respondError(w, logger, http.StatusInternalServerError, "Failed to close reception", err)
		}
		return
	}
	logger.Info("Reception closed successfully")
	reception.Status = "close" // Обновляем статус в объекте перед отправкой
	respondJSON(w, http.StatusOK, reception)
}

// Реализация gRPC-сервиса

type PVZService struct {
	pvzRepo repository.PVZRepository // Используем тот же репозиторий
	logger  *slog.Logger
	pb.UnimplementedPVZServiceServer
}

func NewPVZService(repo repository.PVZRepository, logger *slog.Logger) *PVZService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PVZService{pvzRepo: repo, logger: logger.With(slog.String("component", "grpc_service"), slog.String("service", "PVZService"))}
}

func (s *PVZService) GetPVZList(ctx context.Context, req *pb.GetPVZListRequest) (*pb.GetPVZListResponse, error) {
	s.logger.Info("Handling GetPVZList gRPC request")

	pvzs, err := s.pvzRepo.GetAllPVZ()
	if err != nil {
		s.logger.Error("Failed to get all PVZ from repository for gRPC", slog.Any("error", err))
		return nil, err
	}

	var protoPVZs []*pb.PVZ
	for _, p := range pvzs {
		protoPVZs = append(protoPVZs, &pb.PVZ{
			Id:               p.ID,
			RegistrationDate: timestamppb.New(p.RegistrationDate),
			City:             p.CityName,
		})
	}
	s.logger.Info("Successfully retrieved PVZ list via gRPC", slog.Int("count", len(protoPVZs)))
	return &pb.GetPVZListResponse{Pvzs: protoPVZs}, nil
}

func getUserClaimsFromContext(r *http.Request, logger *slog.Logger) (claims jwt.MapClaims, role string, userID string) {
	claimsValue := r.Context().Value(UserKey)
	if claimsValue == nil {
		logger.Error("Programming error: UserKey not found in context, AuthMiddleware might be missing")
		return nil, "", ""
	}
	claims, ok := claimsValue.(jwt.MapClaims)
	if !ok {
		logger.Error("Programming error: Value for UserKey in context is not jwt.MapClaims", slog.Any("value_type", fmt.Sprintf("%T", claimsValue)))
		return nil, "", ""
	}
	roleVal, roleOk := claims["role"]
	userIDVal, userOk := claims["user_id"]
	if !roleOk || !userOk {
		logger.Error("Claims map is missing 'role' or 'user_id'", slog.Any("claims", claims))
		return claims, "", ""
	}
	roleStr, roleStrOk := roleVal.(string)
	userIDStr, userIDStrOk := userIDVal.(string)
	if !roleStrOk || !userIDStrOk {
		logger.Error("Type assertion failed for 'role' or 'user_id' in claims", slog.Any("claims", claims))
		return claims, "", ""
	}
	return claims, roleStr, userIDStr
}
