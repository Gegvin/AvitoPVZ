package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Загружаем переменные окружения из файла .env, если он существует
func init() {
	godotenv.Load()
}

// Config хранит конфигурационные переменные
// для подключения к базе данных и JWT
//
// DBHost - адрес сервера базы данных
// DBPort - порт
// DBUser - пользователь
// DBPassword - пароль
// DBName - имя базы данных
// JWTSecret - секрет для генерации JWT

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
}

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
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
	return config, nil
}

// getEnv возвращает значение переменной окружения, если она установлена, иначе fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
