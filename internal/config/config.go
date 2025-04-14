package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Загружаем переменные окружения из файла .env, если он существует.
func init() {
	godotenv.Load()
}

// Config хранит конфигурацию приложения.
// Значения загружаются из переменных окружения или используются значения по умолчанию.
type Config struct {
	DBHost     string // Адрес сервера БД
	DBPort     int    // Порт сервера БД
	DBUser     string // Пользователь БД
	DBPassword string // Пароль БД
	DBName     string // Имя базы данных
	JWTSecret  string // Секрет для подписи JWT токенов
}

// LoadConfig загружает конфигурацию из переменных окружения.
func LoadConfig() (*Config, error) {
	portStr := getEnv("DB_PORT", "5432")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		// Используем порт по умолчанию при ошибке парсинга
		port = 5432
	}
	config := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     port,
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "secret"),
		DBName:     getEnv("DB_NAME", "pvz_db"),
		JWTSecret:  getEnv("JWT_SECRET", "your_jwt_secret_key"),
	}
	return config, nil // Ошибка strconv.Atoi игнорируется, используется default
}

// getEnv получает значение переменной окружения или возвращает fallback.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
