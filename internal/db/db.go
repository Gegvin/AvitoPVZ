package db

import (
	"database/sql"
	"fmt"

	"AvitoPVZ/internal/config"
	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// sqlOpener определяет тип для функции, открывающей соединение с БД.
// Используется для упрощения тестирования с моками.
type sqlOpener func(driverName, dataSourceName string) (*sql.DB, error)

// Connect устанавливает соединение с базой данных PostgreSQL.
// Принимает конфигурацию и функцию opener (для тестов).
func Connect(cfg *config.Config, opener sqlOpener) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	// Используем переданную функцию opener (sql.Open в реальном коде)
	db, err := opener("postgres", connStr)
	if err != nil {
		return nil, err // Ошибка при вызове opener (напр., sql.Open)
	}

	// Проверяем соединение с помощью Ping
	if db == nil {
		// Этого не должно происходить, если opener корректен, но для безопасности
		return nil, fmt.Errorf("database connection object is nil after open")
	}
	if err = db.Ping(); err != nil {
		db.Close()      // Закрываем соединение, если Ping не удался
		return nil, err // Возвращаем ошибку Ping
	}

	return db, nil
}
