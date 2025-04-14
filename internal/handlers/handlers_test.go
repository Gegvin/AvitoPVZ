package handlers

import (
	"AvitoPVZ/internal/config"
	"AvitoPVZ/internal/models"
	"AvitoPVZ/internal/utils"
	"AvitoPVZ/pkg/api"
	pb "AvitoPVZ/proto"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oapi-codegen/runtime/types"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}
func (m *MockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if u := args.Get(0); u != nil {
		return u.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockPVZRepository struct{ mock.Mock }

func (m *MockPVZRepository) GetCityIDByName(cityName string) (int, error) {
	args := m.Called(cityName)
	return args.Int(0), args.Error(1)
}
func (m *MockPVZRepository) GetAllowedCities() ([]models.AllowedCity, error) {
	args := m.Called()
	if c := args.Get(0); c != nil {
		return c.([]models.AllowedCity), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockPVZRepository) CreatePVZ(pvz *models.PVZ) error {
	args := m.Called(pvz)
	return args.Error(0)
}
func (m *MockPVZRepository) GetPVZList(params url.Values) ([]models.PVZ, error) {
	args := m.Called(params)
	if p := args.Get(0); p != nil {
		pvzList := p.([]models.PVZ)
		clone := make([]models.PVZ, len(pvzList))
		copy(clone, pvzList)
		return clone, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockPVZRepository) GetAllPVZ() ([]models.PVZ, error) {
	args := m.Called()
	if p := args.Get(0); p != nil {
		pvzList := p.([]models.PVZ)
		clone := make([]models.PVZ, len(pvzList))
		copy(clone, pvzList)
		return clone, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockReceptionRepository struct{ mock.Mock }

func (m *MockReceptionRepository) OpenReceptionExists(pvzId string) (bool, error) {
	args := m.Called(pvzId)
	return args.Bool(0), args.Error(1)
}
func (m *MockReceptionRepository) CreateReception(reception *models.Reception) error {
	args := m.Called(reception)
	return args.Error(0)
}
func (m *MockReceptionRepository) GetOpenReception(pvzId string) (*models.Reception, error) {
	args := m.Called(pvzId)
	if r := args.Get(0); r != nil {
		return r.(*models.Reception), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockReceptionRepository) CloseReception(receptionId string) error {
	args := m.Called(receptionId)
	return args.Error(0)
}

type MockProductRepository struct{ mock.Mock }

func (m *MockProductRepository) AddProduct(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}
func (m *MockProductRepository) DeleteLastProduct(pvzId string) error {
	args := m.Called(pvzId)
	return args.Error(0)
}

type MockProductTypeRepository struct{ mock.Mock }

func (m *MockProductTypeRepository) GetProductTypeIDByName(typeName string) (int, error) {
	args := m.Called(typeName)
	return args.Int(0), args.Error(1)
}

func setupTestHandler() (*Handler, *MockUserRepository, *MockPVZRepository, *MockReceptionRepository, *MockProductRepository, *MockProductTypeRepository) {
	mockUserRepo := new(MockUserRepository)
	mockPVZRepo := new(MockPVZRepository)
	mockReceptionRepo := new(MockReceptionRepository)
	mockProductRepo := new(MockProductRepository)
	mockProductTypeRepo := new(MockProductTypeRepository)
	cfg := &config.Config{JWTSecret: "test-secret"}
	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := NewHandler(nil, cfg, testLogger)
	h.userRepo = mockUserRepo
	h.pvzRepo = mockPVZRepo
	h.receptionRepo = mockReceptionRepo
	h.productRepo = mockProductRepo
	h.productTypeRepo = mockProductTypeRepo
	return h, mockUserRepo, mockPVZRepo, mockReceptionRepo, mockProductRepo, mockProductTypeRepo
}

func addClaimsToContext(r *http.Request, userID, role string) *http.Request {
	claims := jwt.MapClaims{"user_id": userID, "role": role, "exp": time.Now().Add(time.Hour).Unix()}
	ctx := context.WithValue(r.Context(), UserKey, claims)
	return r.WithContext(ctx)
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) { return 0, errors.New("simulated read error") }

type MalformedJson struct{}

func (m MalformedJson) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("intentional marshal error")
}

func TestHandler_Register_Success(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	mockUserRepo.On("CreateUser", mock.MatchedBy(func(user *models.User) bool { return user.Email == "newuser@example.com" && user.Role == "employee" })).Return(nil)
	registerDTO := api.RegisterRequest{Email: types.Email("newuser@example.com"), Password: "password123", Role: api.RegisterRequestRole("employee")}
	body, _ := json.Marshal(registerDTO)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	var respUser api.UserResponse
	err := json.Unmarshal(rr.Body.Bytes(), &respUser)
	assert.NoError(t, err)
	assert.NotNil(t, respUser.Email)
	assert.Equal(t, "newuser@example.com", string(*respUser.Email))
	assert.NotNil(t, respUser.Role)
	assert.Equal(t, "employee", string(*respUser.Role))
	assert.NotNil(t, respUser.Id)
	assert.NotEmpty(t, *respUser.Id)
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Register_InvalidRole(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	registerDTO := api.RegisterRequest{Email: types.Email("badrole@example.com"), Password: "password123", Role: api.RegisterRequestRole("admin")}
	body, _ := json.Marshal(registerDTO)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid role specified")
}
func TestHandler_Register_CreateUserFails(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	dbError := errors.New("database error")
	mockUserRepo.On("CreateUser", mock.AnythingOfType("*models.User")).Return(dbError)
	registerDTO := api.RegisterRequest{Email: types.Email("fail@example.com"), Password: "password123", Role: api.RegisterRequestRole("employee")}
	body, _ := json.Marshal(registerDTO)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "User creation failed")
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Register_DuplicateEmail(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	duplicateError := errors.New("pq: duplicate key value violates unique constraint \"users_email_key\"")
	mockUserRepo.On("CreateUser", mock.MatchedBy(func(user *models.User) bool { return user.Email == "duplicate@example.com" })).Return(duplicateError)
	registerDTO := api.RegisterRequest{Email: types.Email("duplicate@example.com"), Password: "password123", Role: api.RegisterRequestRole("moderator")}
	body, _ := json.Marshal(registerDTO)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusConflict, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "Email already exists", errResp.Message)
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Register_JSONError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/register", bytes.NewReader([]byte("{invalid json")))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_Register_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/register", errorReader{})
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_Login_Success(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	password := "password123"
	userEmail := "login@example.com"
	hashedPassword, _ := utils.HashPassword(password)
	foundUser := &models.User{ID: uuid.New().String(), Email: userEmail, Password: hashedPassword, Role: "employee"}
	mockUserRepo.On("GetUserByEmail", userEmail).Return(foundUser, nil)
	loginDTO := api.LoginRequest{Email: types.Email(userEmail), Password: password}
	body, _ := json.Marshal(loginDTO)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	var tokenResp api.TokenResponse
	err := json.Unmarshal(rr.Body.Bytes(), &tokenResp)
	assert.NoError(t, err)
	assert.NotNil(t, tokenResp.Token)
	assert.NotEmpty(t, *tokenResp.Token)
	claims, err := utils.ParseToken(*tokenResp.Token, h.cfg.JWTSecret)
	assert.NoError(t, err)
	assert.Equal(t, foundUser.ID, claims["user_id"])
	assert.Equal(t, foundUser.Role, claims["role"])
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Login_UserNotFound(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	email := "notfound@example.com"
	mockUserRepo.On("GetUserByEmail", email).Return(nil, sql.ErrNoRows)
	loginDTO := api.LoginRequest{Email: types.Email(email), Password: "password123"}
	body, _ := json.Marshal(loginDTO)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "User not found or invalid credentials")
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Login_InvalidPassword(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	correctPassword := "password123"
	wrongPassword := "wrongpass"
	userEmail := "login@example.com"
	hashedPassword, _ := utils.HashPassword(correctPassword)
	foundUser := &models.User{ID: uuid.New().String(), Email: userEmail, Password: hashedPassword, Role: "employee"}
	mockUserRepo.On("GetUserByEmail", userEmail).Return(foundUser, nil)
	loginDTO := api.LoginRequest{Email: types.Email(userEmail), Password: wrongPassword}
	body, _ := json.Marshal(loginDTO)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "User not found or invalid credentials")
	mockUserRepo.AssertExpectations(t)
}
func TestHandler_Login_JSONError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/login", bytes.NewReader([]byte("{invalid json")))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_Login_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/login", errorReader{})
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_Login_GetUserDBError(t *testing.T) {
	h, mockUserRepo, _, _, _, _ := setupTestHandler()
	email := "dberror@example.com"
	dbErr := errors.New("unexpected database error")
	mockUserRepo.On("GetUserByEmail", email).Return(nil, dbErr)
	loginDTO := api.LoginRequest{Email: types.Email(email), Password: "password123"}
	body, _ := json.Marshal(loginDTO)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Database error during login")
	mockUserRepo.AssertExpectations(t)
}

func TestHandler_CreatePVZ_Success(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	cityName := "Москва"
	expectedCityID := 1
	mockPVZRepo.On("GetCityIDByName", cityName).Return(expectedCityID, nil)
	mockPVZRepo.On("CreatePVZ", mock.MatchedBy(func(pvz *models.PVZ) bool {
		return pvz.CityID == expectedCityID && pvz.ID != "" && !pvz.RegistrationDate.IsZero()
	})).Return(nil)
	pvzData := map[string]string{"city": cityName}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	var respPVZ models.PVZ
	err := json.Unmarshal(rr.Body.Bytes(), &respPVZ)
	require.NoError(t, err)
	assert.Equal(t, cityName, respPVZ.CityName)
	assert.NotEmpty(t, respPVZ.ID)
	assert.False(t, respPVZ.RegistrationDate.IsZero())
	mockPVZRepo.AssertExpectations(t)
}
func TestHandler_CreatePVZ_Forbidden_NotModerator(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	pvzData := map[string]string{"city": "Казань"}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "insufficient privileges")
	mockPVZRepo.AssertNotCalled(t, "GetCityIDByName", mock.Anything)
	mockPVZRepo.AssertNotCalled(t, "CreatePVZ", mock.Anything)
}
func TestHandler_CreatePVZ_InvalidCity(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	invalidCityName := "Самара"
	mockPVZRepo.On("GetCityIDByName", invalidCityName).Return(0, sql.ErrNoRows)
	pvzData := map[string]string{"city": invalidCityName}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "City not allowed or does not exist")
	mockPVZRepo.AssertExpectations(t)
	mockPVZRepo.AssertNotCalled(t, "CreatePVZ", mock.Anything)
}
func TestHandler_CreatePVZ_CityValidationDBError(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	cityName := "Москва"
	dbError := errors.New("failed to connect to city db")
	mockPVZRepo.On("GetCityIDByName", cityName).Return(0, dbError)
	pvzData := map[string]string{"city": cityName}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to validate city")
	mockPVZRepo.AssertExpectations(t)
	mockPVZRepo.AssertNotCalled(t, "CreatePVZ", mock.Anything)
}
func TestHandler_CreatePVZ_EmptyCityName(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	pvzData := map[string]string{"city": ""}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "City name cannot be empty")
	mockPVZRepo.AssertNotCalled(t, "GetCityIDByName", mock.Anything)
	mockPVZRepo.AssertNotCalled(t, "CreatePVZ", mock.Anything)
}

func TestHandler_GetPVZList_Success_Employee(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	testTime := time.Now().Truncate(time.Second)
	expectedModels := []models.PVZ{
		{ID: uuid.New().String(), CityName: "Москва", RegistrationDate: testTime},
	}
	mockPVZRepo.On("GetPVZList", mock.AnythingOfType("url.Values")).Return(expectedModels, nil).Once()

	req := httptest.NewRequest("GET", "/pvz?limit=5&receptionDate=2025-04-14", nil)
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")

	h.GetPVZList(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var actualModels []models.PVZ
	err := json.Unmarshal(rr.Body.Bytes(), &actualModels)
	require.NoError(t, err)

	require.Len(t, actualModels, len(expectedModels))
	assert.Equal(t, expectedModels[0].ID, actualModels[0].ID)
	assert.Equal(t, expectedModels[0].CityName, actualModels[0].CityName)

	assert.WithinDuration(t, expectedModels[0].RegistrationDate, actualModels[0].RegistrationDate, time.Second)

	mockPVZRepo.AssertExpectations(t)
}

func TestHandler_GetPVZList_Success_Moderator(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	expectedModels := []models.PVZ{}
	mockPVZRepo.On("GetPVZList", mock.AnythingOfType("url.Values")).Return(expectedModels, nil).Once()

	req := httptest.NewRequest("GET", "/pvz", nil)
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")

	h.GetPVZList(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var actualModels []models.PVZ
	err := json.Unmarshal(rr.Body.Bytes(), &actualModels)
	require.NoError(t, err)
	assert.Equal(t, expectedModels, actualModels)
	mockPVZRepo.AssertExpectations(t)
}

func TestHandler_GetPVZList_Forbidden(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/pvz", nil)
	rr := httptest.NewRecorder()
	claims := jwt.MapClaims{"user_id": uuid.New().String(), "role": "guest", "exp": time.Now().Add(time.Hour).Unix()}
	ctx := context.WithValue(req.Context(), UserKey, claims)
	req = req.WithContext(ctx)

	h.GetPVZList(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Access denied")
	mockPVZRepo.AssertNotCalled(t, "GetPVZList", mock.Anything)
}

func TestHandler_GetPVZList_RepoError(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	repoError := errors.New("database connection failed")
	mockPVZRepo.On("GetPVZList", mock.AnythingOfType("url.Values")).Return(nil, repoError).Once()

	req := httptest.NewRequest("GET", "/pvz", nil)
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")

	h.GetPVZList(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to get PVZ list")
	mockPVZRepo.AssertExpectations(t)
}

func TestHandler_GetPVZList_InvalidParamsErrorFromRepo(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	parsingError := errors.New("invalid parameter: 'limit' must be a positive integer")
	mockPVZRepo.On("GetPVZList", mock.AnythingOfType("url.Values")).Return(nil, parsingError).Once()

	req := httptest.NewRequest("GET", "/pvz?limit=abc", nil)
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")

	h.GetPVZList(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, parsingError.Error(), errResp.Message)
	mockPVZRepo.AssertExpectations(t)
}

func TestHandler_CreateReception_Success(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockReceptionRepo.On("OpenReceptionExists", pvzID).Return(false, nil)
	mockReceptionRepo.On("CreateReception", mock.AnythingOfType("*models.Reception")).Return(nil)
	receptionData := map[string]string{"pvzId": pvzID}
	body, _ := json.Marshal(receptionData)
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	var respRec models.Reception
	err := json.Unmarshal(rr.Body.Bytes(), &respRec)
	assert.NoError(t, err)
	assert.Equal(t, pvzID, respRec.PVZID)
	assert.Equal(t, "in_progress", respRec.Status)
	assert.NotEmpty(t, respRec.ID)
	mockReceptionRepo.AssertExpectations(t)
}
func TestHandler_CreateReception_Forbidden(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	receptionData := map[string]string{"pvzId": pvzID}
	body, _ := json.Marshal(receptionData)
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "employee required")
}
func TestHandler_CreateReception_AlreadyExists(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockReceptionRepo.On("OpenReceptionExists", pvzID).Return(true, nil)
	receptionData := map[string]string{"pvzId": pvzID}
	body, _ := json.Marshal(receptionData)
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Open reception already exists")
	mockReceptionRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "CreateReception", mock.Anything)
}

func TestHandler_AddProduct_Success(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productTypeName := "одежда"
	expectedTypeID := 2
	openReception := &models.Reception{ID: uuid.New().String(), PVZID: pvzID, Status: "in_progress"}
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(openReception, nil)
	mockProductTypeRepo.On("GetProductTypeIDByName", productTypeName).Return(expectedTypeID, nil)
	mockProductRepo.On("AddProduct", mock.MatchedBy(func(p *models.Product) bool { return p.ReceptionID == openReception.ID && p.TypeID == expectedTypeID })).Return(nil)
	productData := map[string]string{"type": productTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	var respProd models.Product
	err := json.Unmarshal(rr.Body.Bytes(), &respProd)
	assert.NoError(t, err)
	assert.NotEmpty(t, respProd.ID)
	assert.Equal(t, openReception.ID, respProd.ReceptionID)
	assert.Equal(t, productTypeName, respProd.TypeName)
	mockReceptionRepo.AssertExpectations(t)
	mockProductTypeRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}
func TestHandler_AddProduct_Forbidden(t *testing.T) {
	h, _, _, _, _, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productData := map[string]string{"type": "одежда", "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "employee required")
	mockProductTypeRepo.AssertNotCalled(t, "GetProductTypeIDByName", mock.Anything)
}
func TestHandler_AddProduct_NoOpenReception(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productTypeName := "обувь"
	expectedTypeID := 3
	mockProductTypeRepo.On("GetProductTypeIDByName", productTypeName).Return(expectedTypeID, nil)
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(nil, sql.ErrNoRows)
	productData := map[string]string{"type": productTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "No open reception found")
	mockProductTypeRepo.AssertExpectations(t)
	mockReceptionRepo.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "AddProduct", mock.Anything)
}
func TestHandler_AddProduct_InvalidOrUnsupportedType(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	invalidProductTypeName := "мебель"
	mockProductTypeRepo.On("GetProductTypeIDByName", invalidProductTypeName).Return(0, sql.ErrNoRows)
	productData := map[string]string{"type": invalidProductTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid or unsupported product type")
	mockProductTypeRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "GetOpenReception", mock.Anything)
	mockProductRepo.AssertNotCalled(t, "AddProduct", mock.Anything)
}
func TestHandler_AddProduct_TypeLookupDBError(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productTypeName := "одежда"
	dbError := errors.New("connection error during type lookup")
	mockProductTypeRepo.On("GetProductTypeIDByName", productTypeName).Return(0, dbError)
	productData := map[string]string{"type": productTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to validate product type")
	mockProductTypeRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "GetOpenReception", mock.Anything)
	mockProductRepo.AssertNotCalled(t, "AddProduct", mock.Anything)
}

func TestHandler_CloseLastReception_Success(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	receptionID := uuid.New().String()
	openReceptionModel := &models.Reception{ID: receptionID, PVZID: pvzID, Status: "in_progress", DateTime: time.Now()}
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(openReceptionModel, nil)
	mockReceptionRepo.On("CloseReception", receptionID).Return(nil)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	var respRec models.Reception
	err := json.Unmarshal(rr.Body.Bytes(), &respRec)
	assert.NoError(t, err)
	assert.Equal(t, receptionID, respRec.ID)
	assert.Equal(t, "close", respRec.Status)
	mockReceptionRepo.AssertExpectations(t)
}
func TestHandler_CloseLastReception_Forbidden(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	rr := httptest.NewRecorder()
	reqModerator := addClaimsToContext(req, uuid.New().String(), "moderator")
	router.ServeHTTP(rr, reqModerator)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "employee required")
}
func TestHandler_CloseLastReception_NotFound(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(nil, sql.ErrNoRows)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "No open reception found")
	mockReceptionRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "CloseReception", mock.Anything)
}
func TestHandler_AuthMiddleware_Success(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(UserKey).(jwt.MapClaims)
		assert.True(t, ok)
		assert.NotNil(t, claims)
		assert.Equal(t, "test-user-id", claims["user_id"])
		assert.Equal(t, "employee", claims["role"])
		w.WriteHeader(http.StatusOK)
	})
	middleware := h.AuthMiddleware(nextHandler)
	token, _ := utils.GenerateToken("test-user-id", "employee", h.cfg.JWTSecret)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
func TestHandler_AuthMiddleware_MissingHeader(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("...") })
	middleware := h.AuthMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "missing auth header")
}
func TestHandler_AuthMiddleware_InvalidHeader_WrongFormat(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("...") })
	middleware := h.AuthMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidTokenFormat")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "invalid auth header")
}
func TestHandler_AuthMiddleware_InvalidHeader_EmptyToken(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("...") })
	middleware := h.AuthMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "invalid token")
}
func TestHandler_AuthMiddleware_InvalidToken(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("...") })
	middleware := h.AuthMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+"this.is.an.invalid.token")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "invalid token")
}
func TestHandler_DeleteLastProduct_Success(t *testing.T) {
	h, _, _, _, mockProductRepo, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockProductRepo.On("DeleteLastProduct", pvzID).Return(nil)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/delete_last_product", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	var succResp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &succResp)
	assert.NoError(t, err)
	assert.Equal(t, "Product deleted", succResp["message"])
	mockProductRepo.AssertExpectations(t)
}
func TestHandler_DeleteLastProduct_Forbidden(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/delete_last_product", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")
	rr := httptest.NewRecorder()
	reqModerator := addClaimsToContext(req, uuid.New().String(), "moderator")
	router.ServeHTTP(rr, reqModerator)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "employee required")
}
func TestHandler_DeleteLastProduct_Error(t *testing.T) {
	h, _, _, _, mockProductRepo, _ := setupTestHandler()
	pvzID := uuid.New().String()
	repoError := errors.New("no product to delete")
	mockProductRepo.On("DeleteLastProduct", pvzID).Return(repoError)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/delete_last_product", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, repoError.Error(), errResp.Message)
	mockProductRepo.AssertExpectations(t)
}
func TestHandler_DeleteLastProduct_DBError(t *testing.T) {
	h, _, _, _, mockProductRepo, _ := setupTestHandler()
	pvzID := uuid.New().String()
	dbError := errors.New("database connection error")
	mockProductRepo.On("DeleteLastProduct", pvzID).Return(dbError)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/delete_last_product", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/delete_last_product", h.DeleteLastProduct).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to delete product")
	mockProductRepo.AssertExpectations(t)
}
func TestHandler_DummyLogin_Success(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	roles := []string{"employee", "moderator"}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			payload := map[string]string{"role": role}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/dummyLogin", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			h.DummyLogin(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			var tokenResp api.TokenResponse
			err := json.Unmarshal(rr.Body.Bytes(), &tokenResp)
			assert.NoError(t, err)
			assert.NotNil(t, tokenResp.Token)
			assert.NotEmpty(t, *tokenResp.Token)
			claims, err := utils.ParseToken(*tokenResp.Token, h.cfg.JWTSecret)
			assert.NoError(t, err)
			assert.Equal(t, role, claims["role"])
			assert.NotEmpty(t, claims["user_id"])
		})
	}
}
func TestHandler_DummyLogin_InvalidRole(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	payload := map[string]string{"role": "admin"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/dummyLogin", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.DummyLogin(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid role specified")
}
func TestHandler_DummyLogin_InvalidRequest(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	invalidBody := []byte("{not a valid json")
	req := httptest.NewRequest("POST", "/dummyLogin", bytes.NewReader(invalidBody))
	rr := httptest.NewRecorder()
	h.DummyLogin(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_DummyLogin_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/dummyLogin", errorReader{})
	rr := httptest.NewRecorder()
	h.DummyLogin(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}

func TestPVZService_GetPVZList_Success(t *testing.T) {
	_, _, mockPVZRepo, _, _, _ := setupTestHandler()
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	serviceWithMock := NewPVZService(mockPVZRepo, testLogger)
	expectedDBPVZs := []models.PVZ{{ID: uuid.New().String(), CityID: 2, CityName: "Санкт-Петербург", RegistrationDate: time.Now()}, {ID: uuid.New().String(), CityID: 1, CityName: "Москва", RegistrationDate: time.Now().Add(-2 * time.Hour)}}
	mockPVZRepo.On("GetAllPVZ").Return(expectedDBPVZs, nil)
	req := &pb.GetPVZListRequest{}
	resp, err := serviceWithMock.GetPVZList(context.Background(), req)
	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Pvzs, len(expectedDBPVZs))
	for i, expected := range expectedDBPVZs {
		actual := resp.Pvzs[i]
		assert.Equal(t, expected.ID, actual.Id)
		assert.Equal(t, expected.CityName, actual.City)
		assert.Equal(t, expected.RegistrationDate.Unix(), actual.RegistrationDate.AsTime().Unix())
	}
	mockPVZRepo.AssertExpectations(t)
}
func TestPVZService_GetPVZList_Error(t *testing.T) {
	_, _, mockPVZRepo, _, _, _ := setupTestHandler()
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	serviceWithMock := NewPVZService(mockPVZRepo, testLogger)
	repoError := errors.New("failed to fetch from db")
	mockPVZRepo.On("GetAllPVZ").Return(nil, repoError)
	req := &pb.GetPVZListRequest{}
	resp, err := serviceWithMock.GetPVZList(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, repoError, err)
	mockPVZRepo.AssertExpectations(t)
}

func TestNewHandler(t *testing.T) {
	cfg := &config.Config{}
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(nil, cfg, testLogger)
	assert.NotNil(t, h)
	assert.Equal(t, cfg, h.cfg)
	assert.NotNil(t, h.logger)
	assert.NotNil(t, h.userRepo)
	assert.NotNil(t, h.pvzRepo)
	assert.NotNil(t, h.receptionRepo)
	assert.NotNil(t, h.productRepo)
	assert.NotNil(t, h.productTypeRepo)
}
func Test_respondJSON_MarshalError(t *testing.T) {
	rr := httptest.NewRecorder()
	respondJSON(rr, http.StatusOK, MalformedJson{})
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Internal Server Error")
}
func TestHandler_CreatePVZ_JSONError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader([]byte("{invalid json")))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body format")
}
func TestHandler_CreateReception_JSONError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader([]byte("{invalid json")))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_AddProduct_JSONError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/products", bytes.NewReader([]byte("{invalid json")))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_CreatePVZ_CreateError(t *testing.T) {
	h, _, mockPVZRepo, _, _, _ := setupTestHandler()
	cityName := "Москва"
	cityID := 1
	repoErr := errors.New("db insert constraint violation")
	mockPVZRepo.On("GetCityIDByName", cityName).Return(cityID, nil)
	mockPVZRepo.On("CreatePVZ", mock.AnythingOfType("*models.PVZ")).Return(repoErr)
	pvzData := map[string]string{"city": cityName}
	body, _ := json.Marshal(pvzData)
	req := httptest.NewRequest("POST", "/pvz", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to create PVZ")
	mockPVZRepo.AssertExpectations(t)
}
func TestHandler_CreateReception_ExistsError(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockReceptionRepo.On("OpenReceptionExists", pvzID).Return(false, errors.New("db check error"))
	receptionData := map[string]string{"pvzId": pvzID}
	body, _ := json.Marshal(receptionData)
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Error checking reception status")
	mockReceptionRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "CreateReception", mock.Anything)
}
func TestHandler_CreateReception_CreateError(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	mockReceptionRepo.On("OpenReceptionExists", pvzID).Return(false, nil)
	mockReceptionRepo.On("CreateReception", mock.AnythingOfType("*models.Reception")).Return(errors.New("db insert error"))
	receptionData := map[string]string{"pvzId": pvzID}
	body, _ := json.Marshal(receptionData)
	req := httptest.NewRequest("POST", "/receptions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to create reception")
	mockReceptionRepo.AssertExpectations(t)
}
func TestHandler_AddProduct_GetOpenError(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productTypeName := "одежда"
	expectedTypeID := 2
	dbErr := errors.New("db get error")
	mockProductTypeRepo.On("GetProductTypeIDByName", productTypeName).Return(expectedTypeID, nil)
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(nil, dbErr)
	productData := map[string]string{"type": productTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Error checking reception status")
	mockProductTypeRepo.AssertExpectations(t)
	mockReceptionRepo.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "AddProduct", mock.Anything)
}
func TestHandler_AddProduct_AddError(t *testing.T) {
	h, _, _, mockReceptionRepo, mockProductRepo, mockProductTypeRepo := setupTestHandler()
	pvzID := uuid.New().String()
	productTypeName := "одежда"
	expectedTypeID := 2
	openReception := &models.Reception{ID: uuid.New().String(), PVZID: pvzID, Status: "in_progress"}
	mockProductTypeRepo.On("GetProductTypeIDByName", productTypeName).Return(expectedTypeID, nil)
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(openReception, nil)
	mockProductRepo.On("AddProduct", mock.MatchedBy(func(p *models.Product) bool { return p.ReceptionID == openReception.ID && p.TypeID == expectedTypeID })).Return(errors.New("db insert error"))
	productData := map[string]string{"type": productTypeName, "pvzId": pvzID}
	body, _ := json.Marshal(productData)
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to add product")
	mockProductTypeRepo.AssertExpectations(t)
	mockReceptionRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}
func TestHandler_CloseLastReception_GetOpenError(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	dbErr := errors.New("db get error")
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(nil, dbErr)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Error checking reception status")
	mockReceptionRepo.AssertExpectations(t)
	mockReceptionRepo.AssertNotCalled(t, "CloseReception", mock.Anything)
}
func TestHandler_CloseLastReception_CloseError(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	receptionID := uuid.New().String()
	openReception := &models.Reception{ID: receptionID, PVZID: pvzID, Status: "in_progress", DateTime: time.Now()}
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(openReception, nil)
	mockReceptionRepo.On("CloseReception", receptionID).Return(errors.New("db update error"))
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Failed to close reception")
	mockReceptionRepo.AssertExpectations(t)
}
func TestHandler_CloseLastReception_NoRowsUpdatedError(t *testing.T) {
	h, _, _, mockReceptionRepo, _, _ := setupTestHandler()
	pvzID := uuid.New().String()
	receptionID := uuid.New().String()
	openReception := &models.Reception{ID: receptionID, PVZID: pvzID, Status: "in_progress", DateTime: time.Now()}
	noUpdateErr := errors.New("no reception updated")
	mockReceptionRepo.On("GetOpenReception", pvzID).Return(openReception, nil)
	mockReceptionRepo.On("CloseReception", receptionID).Return(noUpdateErr)
	req := httptest.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)
	req = mux.SetURLVars(req, map[string]string{"pvzId": pvzID})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	router := mux.NewRouter()
	router.HandleFunc("/pvz/{pvzId}/close_last_reception", h.CloseLastReception).Methods("POST")
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "Reception not found or already closed")
	mockReceptionRepo.AssertExpectations(t)
}
func TestHandler_CreatePVZ_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/pvz", errorReader{})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "moderator")
	h.CreatePVZ(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body format")
}
func TestHandler_CreateReception_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/receptions", errorReader{})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.CreateReception(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestHandler_AddProduct_BodyReadError(t *testing.T) {
	h, _, _, _, _, _ := setupTestHandler()
	req := httptest.NewRequest("POST", "/products", errorReader{})
	rr := httptest.NewRecorder()
	req = addClaimsToContext(req, uuid.New().String(), "employee")
	h.AddProduct(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errResp api.Error
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Message, "Invalid request body")
}
func TestGetUserClaimsFromContext_MissingValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims, role, userID := getUserClaimsFromContext(req, testLogger)
	assert.Nil(t, claims)
	assert.Empty(t, role)
	assert.Empty(t, userID)
}
func TestGetUserClaimsFromContext_WrongValueType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserKey, "not jwt.MapClaims")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims, role, userID := getUserClaimsFromContext(req, testLogger)
	assert.Nil(t, claims)
	assert.Empty(t, role)
	assert.Empty(t, userID)
}
func TestGetUserClaimsFromContext_MissingRoleClaim(t *testing.T) {
	claimsMap := jwt.MapClaims{"user_id": uuid.New().String(), "exp": time.Now().Add(time.Hour).Unix()}
	ctx := context.WithValue(context.Background(), UserKey, claimsMap)
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims, role, userID := getUserClaimsFromContext(req, testLogger)
	assert.Equal(t, claimsMap, claims)
	assert.Empty(t, role)
	assert.Empty(t, userID)
}
func TestGetUserClaimsFromContext_WrongUserIDClaimType(t *testing.T) {
	claimsMap := jwt.MapClaims{"user_id": 12345, "role": "employee", "exp": time.Now().Add(time.Hour).Unix()}
	ctx := context.WithValue(context.Background(), UserKey, claimsMap)
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	claims, role, userID := getUserClaimsFromContext(req, testLogger)
	assert.Equal(t, claimsMap, claims)
	assert.Empty(t, role)
	assert.Empty(t, userID)
}
func TestNewHandler_NilLogger(t *testing.T) {
	cfg := &config.Config{}
	h := NewHandler(nil, cfg, nil)
	require.NotNil(t, h)
	assert.NotNil(t, h.logger)
	assert.Equal(t, cfg, h.cfg)
	assert.NotNil(t, h.userRepo)
	assert.NotNil(t, h.pvzRepo)
	assert.NotNil(t, h.receptionRepo)
	assert.NotNil(t, h.productRepo)
	assert.NotNil(t, h.productTypeRepo)
}
func TestNewPVZService_NilLogger(t *testing.T) {
	mockPVZRepo := new(MockPVZRepository)
	s := NewPVZService(mockPVZRepo, nil)
	require.NotNil(t, s)
	assert.NotNil(t, s.logger)
	assert.Equal(t, mockPVZRepo, s.pvzRepo)
}
