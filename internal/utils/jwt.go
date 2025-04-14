package utils

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

// GenerateToken создает новый JWT токен с указанными userID, role и секретом.
func GenerateToken(userID, role, jwtSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Срок действия токена - 72 часа
	})
	return token.SignedString([]byte(jwtSecret))
}

// ParseToken проверяет и разбирает строку JWT токена, возвращая claims.
func ParseToken(tokenStr, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Проверка метода подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return []byte(jwtSecret), nil
	})

	// Обработка ошибок валидации (включая истекший срок, невалидную подпись)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	// Если claims не MapClaims или токен невалиден (хотя Parse должен был вернуть ошибку)
	return nil, jwt.NewValidationError("invalid token claims", jwt.ValidationErrorClaimsInvalid)
}
