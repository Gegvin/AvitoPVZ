package utils

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const testJWTSecret = "test-secret-key-for-jwt"

func TestGenerateAndParseToken(t *testing.T) {
	userID := uuid.New().String()
	role := "employee"

	tokenString, err := GenerateToken(userID, role, testJWTSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	claims, err := ParseToken(tokenString, testJWTSecret)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	assert.Equal(t, userID, claims["user_id"])
	assert.Equal(t, role, claims["role"])

	expClaim, ok := claims["exp"].(float64)
	assert.True(t, ok)
	expTime := time.Unix(int64(expClaim), 0)
	assert.True(t, expTime.After(time.Now()))
}

func TestParseTokenInvalidSignature(t *testing.T) {
	userID := uuid.New().String()
	role := "moderator"

	tokenString, _ := GenerateToken(userID, role, testJWTSecret)

	_, err := ParseToken(tokenString, "different-secret")
	assert.Error(t, err)
	validationErr, ok := err.(*jwt.ValidationError)
	assert.True(t, ok)
	assert.Equal(t, jwt.ValidationErrorSignatureInvalid, validationErr.Errors)
}

func TestParseTokenExpired(t *testing.T) {
	userID := uuid.New().String()
	role := "employee"
	expiredSecret := "expired-secret"

	expiredClaims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, _ := token.SignedString([]byte(expiredSecret))

	_, err := ParseToken(tokenString, expiredSecret)
	assert.Error(t, err)
	validationErr, ok := err.(*jwt.ValidationError)
	assert.True(t, ok)
	assert.Equal(t, jwt.ValidationErrorExpired, validationErr.Errors)
}

func TestParseTokenMalformed(t *testing.T) {
	malformedToken := "this.is.not.a.jwt"
	_, err := ParseToken(malformedToken, testJWTSecret)
	assert.Error(t, err)
}
