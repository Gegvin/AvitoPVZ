package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductTypeRepo_GetProductTypeIDByName_Found(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductTypeRepository(db)
	typeName := "электроника"
	expectedID := 1

	rows := sqlmock.NewRows([]string{"id"}).AddRow(expectedID)
	expectedSQL := "SELECT id FROM product_types WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(typeName).WillReturnRows(rows)

	actualID, err := repo.GetProductTypeIDByName(typeName)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductTypeRepo_GetProductTypeIDByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductTypeRepository(db)
	typeName := "мебель"

	expectedSQL := "SELECT id FROM product_types WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(typeName).WillReturnError(sql.ErrNoRows)

	actualID, err := repo.GetProductTypeIDByName(typeName)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Equal(t, 0, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductTypeRepo_GetProductTypeIDByName_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductTypeRepository(db)
	typeName := "одежда"
	expectedError := errors.New("connection refused")

	expectedSQL := "SELECT id FROM product_types WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(typeName).WillReturnError(expectedError)

	actualID, err := repo.GetProductTypeIDByName(typeName)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, 0, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
