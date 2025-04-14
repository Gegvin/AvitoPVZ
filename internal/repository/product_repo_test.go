package repository

import (
	"errors"
	"testing"
	"time"

	"AvitoPVZ/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProductRepo_AddProduct(t *testing.T) {

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewProductRepository(db)
	product := &models.Product{
		ID:          uuid.New().String(),
		DateTime:    time.Now(),
		ReceptionID: uuid.New().String(),
		TypeID:      1,
	}

	expectedSQL := "INSERT INTO products (id, date_time, reception_id, type_id) VALUES ($1, $2, $3, $4)"

	mock.ExpectExec(expectedSQL).
		WithArgs(product.ID, product.DateTime, product.ReceptionID, product.TypeID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddProduct(product)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

const deleteLastProductRegex = `^\s*DELETE\s+FROM\s+products\s+WHERE\s+id\s*=\s*\(\s*SELECT\s+p\.id\s+FROM\s+products\s+p\s+JOIN\s+receptions\s+r\s+ON\s+p\.reception_id\s*=\s*r\.id\s+WHERE\s+r\.pvz_id\s*=\s*\$1\s+AND\s+r\.status\s*=\s*'in_progress'\s+ORDER\s+BY\s+p\.date_time\s+DESC\s+LIMIT\s+1\s*\)\s*$`

func TestProductRepo_DeleteLastProduct_Success(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewProductRepository(db)
	pvzID := uuid.New().String()

	mock.ExpectExec(deleteLastProductRegex).
		WithArgs(pvzID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteLastProduct(pvzID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepo_DeleteLastProduct_NotDeleted(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewProductRepository(db)
	pvzID := uuid.New().String()
	expectedErrorMsg := "no product to delete"

	mock.ExpectExec(deleteLastProductRegex).
		WithArgs(pvzID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.DeleteLastProduct(pvzID)
	assert.Error(t, err)
	assert.EqualError(t, err, expectedErrorMsg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepo_DeleteLastProduct_ExecError(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewProductRepository(db)
	pvzID := uuid.New().String()
	expectedDBError := errors.New("constraint violation")

	mock.ExpectExec(deleteLastProductRegex).
		WithArgs(pvzID).
		WillReturnError(expectedDBError)

	err = repo.DeleteLastProduct(pvzID)
	assert.Error(t, err)
	assert.Equal(t, expectedDBError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepo_DeleteLastProduct_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewProductRepository(db)
	pvzID := uuid.New().String()
	rowsAffectedError := errors.New("driver does not support RowsAffected")

	mockResult := sqlmock.NewErrorResult(rowsAffectedError)

	mock.ExpectExec(deleteLastProductRegex).
		WithArgs(pvzID).
		WillReturnResult(mockResult)

	err = repo.DeleteLastProduct(pvzID)

	assert.Error(t, err)
	assert.Equal(t, rowsAffectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
