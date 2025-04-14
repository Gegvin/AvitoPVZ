package repository

import (
	"database/sql"
	"testing"

	"AvitoPVZ/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserRepo_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	user := &models.User{
		ID:       uuid.New().String(),
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role:     "employee",
	}

	expectedSQL := "INSERT INTO users (id, email, password, role) VALUES ($1, $2, $3, $4)"
	mock.ExpectExec(expectedSQL).
		WithArgs(user.ID, user.Email, user.Password, user.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateUser(user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetUserByEmail(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	expectedUser := &models.User{
		ID:       uuid.New().String(),
		Email:    "found@example.com",
		Password: "hashedpassword",
		Role:     "moderator",
	}

	rows := sqlmock.NewRows([]string{"id", "email", "password", "role"}).
		AddRow(expectedUser.ID, expectedUser.Email, expectedUser.Password, expectedUser.Role)

	expectedSQL := "SELECT id, email, password, role FROM users WHERE email=$1"
	mock.ExpectQuery(expectedSQL).
		WithArgs(expectedUser.Email).
		WillReturnRows(rows)

	user, err := repo.GetUserByEmail(expectedUser.Email)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetUserByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	emailToSearch := "notfound@example.com"

	expectedSQL := "SELECT id, email, password, role FROM users WHERE email=$1"
	mock.ExpectQuery(expectedSQL).
		WithArgs(emailToSearch).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByEmail(emailToSearch)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}
