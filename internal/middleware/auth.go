package handlers

//(Файл находится в пакете handlers, чтобы иметь доступ к типу Handler и конфигурации)

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"AvitoPVZ/internal/utils"
)

// Определяем ключ для хранения информации о пользователе в контексте
type key int

const (
	UserKey key = iota
)

// AuthMiddleware проверяет наличие и валидность JWT-токена
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
		// Для отладки: выведем в логи claims
		fmt.Printf("Claims extracted: %+v\n", claims)
		ctx := context.WithValue(r.Context(), UserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
