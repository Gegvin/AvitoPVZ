package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
	"time"

	"AvitoPVZ/internal/config"
	"AvitoPVZ/internal/metrics"
	"AvitoPVZ/internal/models"
	"AvitoPVZ/internal/repository"
	"AvitoPVZ/internal/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "AvitoPVZ/proto"
)

// Определяем уникальный тип ключей для context, чтобы избежать конфликтов с ключами других пакетов.
type contextKey string

// UserKey используется для хранения информации о пользователе в context.
// Использование собственного типа гарантирует уникальность ключа.
const UserKey contextKey = "user"

// Handler хранит зависимости для HTTP-обработчиков.
type Handler struct {
	db            *sql.DB
	cfg           *config.Config
	userRepo      repository.UserRepository
	pvzRepo       repository.PVZRepository
	receptionRepo repository.ReceptionRepository
	productRepo   repository.ProductRepository
}

// NewHandler создаёт новый объект Handler.
func NewHandler(db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:            db,
		cfg:           cfg,
		userRepo:      repository.NewUserRepository(db),
		pvzRepo:       repository.NewPVZRepository(db),
		receptionRepo: repository.NewReceptionRepository(db),
		productRepo:   repository.NewProductRepository(db),
	}
}

// respondJSON отправляет ответ в формате JSON.
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "JSON marshal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// AuthMiddleware – HTTP middleware для проверки заголовка Authorization.
// Он извлекает токен в формате "Bearer <token>", проверяет его с помощью utils.ParseToken,
// и добавляет полученные данные (claims) в контекст запроса под ключом UserKey.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing auth header", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid auth header", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		claims, err := utils.ParseToken(tokenStr, h.cfg.JWTSecret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		// Добавляем данные о пользователе (claims) в контекст запроса.
		ctx := context.WithValue(r.Context(), UserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DummyLogin – выдаёт тестовый токен для указанной роли (employee или moderator).
func (h *Handler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	if req.Role != "employee" && req.Role != "moderator" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid role"})
		return
	}
	token, err := utils.GenerateToken(uuid.New().String(), req.Role, h.cfg.JWTSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Token generation error"})
		return
	}
	respondJSON(w, http.StatusOK, token)
}

// Register – регистрация нового пользователя.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	if req.Role != "employee" && req.Role != "moderator" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid role"})
		return
	}
	hashedPass, err := utils.HashPassword(req.Password)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Password hashing error"})
		return
	}
	user := &models.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Password: hashedPass,
		Role:     req.Role,
	}
	err = h.userRepo.CreateUser(user)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "User creation failed"})
		return
	}
	respondJSON(w, http.StatusCreated, user)
}

// Login – авторизация пользователя и выдача JWT-токена.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	user, err := h.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"message": "User not found"})
		return
	}
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"message": "Invalid credentials"})
		return
	}
	token, err := utils.GenerateToken(user.ID, user.Role, h.cfg.JWTSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Token generation error"})
		return
	}
	respondJSON(w, http.StatusOK, token)
}

// CreatePVZ – создание нового ПВЗ (только для модераторов).
func (h *Handler) CreatePVZ(w http.ResponseWriter, r *http.Request) {
	// Приводим данные из context к jwt.MapClaims
	claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: no claims found"})
		return
	}

	// Извлекаем роль и проверяем, что она "moderator"
	role, roleOk := claims["role"].(string)
	if !roleOk || role != "moderator" {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: insufficient privileges"})
		return
	}

	var req models.PVZ
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	if req.City != "Москва" && req.City != "Санкт-Петербург" && req.City != "Казань" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "City not allowed"})
		return
	}
	req.ID = uuid.New().String()
	req.RegistrationDate = time.Now()
	err := h.pvzRepo.CreatePVZ(&req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create PVZ"})
		return
	}
	metrics.IncPVZCreated()
	respondJSON(w, http.StatusCreated, req)
}

// GetPVZList – получение списка ПВЗ с фильтрацией и пагинацией.
func (h *Handler) GetPVZList(w http.ResponseWriter, r *http.Request) {
	pvzs, err := h.pvzRepo.GetPVZList(r.URL.Query())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to get PVZ list"})
		return
	}
	respondJSON(w, http.StatusOK, pvzs)
}

// CreateReception – создание новой приёмки товаров (только для сотрудников ПВЗ).
func (h *Handler) CreateReception(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: no claims found"})
		return
	}
	role, roleOk := claims["role"].(string)
	if !roleOk || role != "employee" {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: employee required"})
		return
	}
	// Далее стандартная логика создания приёмки
	var req struct {
		PVZID string `json:"pvzId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	// Проверка, что открытой приёмки нет
	exists, err := h.receptionRepo.OpenReceptionExists(req.PVZID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error checking reception"})
		return
	}
	if exists {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Open reception already exists"})
		return
	}
	reception := &models.Reception{
		ID:       uuid.New().String(),
		DateTime: time.Now(),
		PVZID:    req.PVZID,
		Status:   "in_progress",
	}
	if err := h.receptionRepo.CreateReception(reception); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to create reception"})
		return
	}
	metrics.IncReceptionCreated()
	respondJSON(w, http.StatusCreated, reception)
}

// AddProduct – добавление товара в текущую приёмку (только для сотрудников ПВЗ).
func (h *Handler) AddProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: no claims found"})
		return
	}
	role, roleOk := claims["role"].(string)
	if !roleOk || role != "employee" {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: employee required"})
		return
	}

	var req struct {
		Type  string `json:"type"`
		PVZID string `json:"pvzId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request"})
		return
	}
	if req.Type != "электроника" && req.Type != "одежда" && req.Type != "обувь" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid product type"})
		return
	}
	reception, err := h.receptionRepo.GetOpenReception(req.PVZID)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "No open reception found"})
		return
	}
	product := &models.Product{
		ID:          uuid.New().String(),
		DateTime:    time.Now(),
		Type:        req.Type,
		ReceptionID: reception.ID,
	}
	err = h.productRepo.AddProduct(product)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to add product"})
		return
	}
	metrics.IncProductAdded()
	respondJSON(w, http.StatusCreated, product)
}

// DeleteLastProduct – удаление последнего добавленного товара (LIFO) из текущей приёмки (только для сотрудников ПВЗ).
func (h *Handler) DeleteLastProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: no claims found"})
		return
	}
	role, roleOk := claims["role"].(string)
	if !roleOk || role != "employee" {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: employee required"})
		return
	}

	pvzId := mux.Vars(r)["pvzId"]
	err := h.productRepo.DeleteLastProduct(pvzId)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Product deleted"})
}

// CloseLastReception – закрытие последней открытой приёмки в рамках ПВЗ (только для сотрудников ПВЗ).
func (h *Handler) CloseLastReception(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: no claims found"})
		return
	}
	role, roleOk := claims["role"].(string)
	if !roleOk || role != "employee" {
		respondJSON(w, http.StatusForbidden, map[string]string{"message": "Access denied: employee required"})
		return
	}

	pvzId := mux.Vars(r)["pvzId"]
	reception, err := h.receptionRepo.GetOpenReception(pvzId)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "No open reception found"})
		return
	}
	if reception.Status != "in_progress" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Reception already closed"})
		return
	}
	reception.Status = "close"
	err = h.receptionRepo.CloseReception(reception.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to close reception"})
		return
	}
	respondJSON(w, http.StatusOK, reception)
}

// Реализация gRPC-сервиса PVZService.
type PVZService struct {
	db *sql.DB
	pb.UnimplementedPVZServiceServer
}

// NewPVZService создаёт новый сервис для gRPC.
func NewPVZService(db *sql.DB) *PVZService {
	return &PVZService{db: db}
}

// GetPVZList возвращает список ПВЗ (упрощённо – берём все записи).
func (s *PVZService) GetPVZList(ctx context.Context, req *pb.GetPVZListRequest) (*pb.GetPVZListResponse, error) {
	repo := repository.NewPVZRepository(s.db)
	pvzs, err := repo.GetAllPVZ()
	if err != nil {
		return nil, err
	}
	var protoPVZs []*pb.PVZ
	for _, p := range pvzs {
		protoPVZs = append(protoPVZs, &pb.PVZ{
			Id:               p.ID,
			RegistrationDate: timestamppb.New(p.RegistrationDate),
			City:             p.City,
		})
	}
	return &pb.GetPVZListResponse{Pvzs: protoPVZs}, nil
}
