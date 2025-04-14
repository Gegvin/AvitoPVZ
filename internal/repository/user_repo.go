package repository

import (
	"database/sql"

	"AvitoPVZ/internal/models"
)

// UserRepository определяет интерфейс для работы
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
}

type userRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

// CreateUser добавляет нового пользователя в базу данных.
func (r *userRepo) CreateUser(user *models.User) error {
	query := `INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, user.ID, user.Email, user.Password, user.Role)
	return err
}

// GetUserByEmail находит пользователя по его email.

func (r *userRepo) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, password, role FROM users WHERE email=$1`
	row := r.db.QueryRow(query, email)
	user := &models.User{}
	err := row.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		return nil, err // Ошибка включает sql.ErrNoRows
	}
	return user, nil
}
