package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"AvitoPVZ/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReceptionRepo_OpenReceptionExists_True(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	pvzID := uuid.New().String()
	foundID := uuid.New().String()

	rows := sqlmock.NewRows([]string{"id"}).AddRow(foundID)
	expectedSQL := "SELECT id FROM receptions WHERE pvz_id=$1 AND status='in_progress'"
	mock.ExpectQuery(expectedSQL).WithArgs(pvzID).WillReturnRows(rows)

	exists, err := repo.OpenReceptionExists(pvzID)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_OpenReceptionExists_False(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	pvzID := uuid.New().String()

	expectedSQL := "SELECT id FROM receptions WHERE pvz_id=$1 AND status='in_progress'"
	mock.ExpectQuery(expectedSQL).WithArgs(pvzID).WillReturnError(sql.ErrNoRows)

	exists, err := repo.OpenReceptionExists(pvzID)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_OpenReceptionExists_Error(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	pvzID := uuid.New().String()
	queryError := errors.New("db connection lost")

	expectedSQL := "SELECT id FROM receptions WHERE pvz_id=$1 AND status='in_progress'"
	mock.ExpectQuery(expectedSQL).WithArgs(pvzID).WillReturnError(queryError)

	exists, err := repo.OpenReceptionExists(pvzID)
	assert.Error(t, err)
	assert.Equal(t, queryError, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_CreateReception(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	reception := &models.Reception{
		ID:       uuid.New().String(),
		DateTime: time.Now(),
		PVZID:    uuid.New().String(),
		Status:   "in_progress",
	}

	expectedSQL := "INSERT INTO receptions (id, date_time, pvz_id, status) VALUES ($1, $2, $3, $4)"
	mock.ExpectExec(expectedSQL).
		WithArgs(reception.ID, reception.DateTime, reception.PVZID, reception.Status).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateReception(reception)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_GetOpenReception_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	pvzID := uuid.New().String()
	expectedReception := &models.Reception{
		ID:       uuid.New().String(),
		DateTime: time.Now(),
		PVZID:    pvzID,
		Status:   "in_progress",
	}

	rows := sqlmock.NewRows([]string{"id", "date_time", "pvz_id", "status"}).
		AddRow(expectedReception.ID, expectedReception.DateTime, expectedReception.PVZID, expectedReception.Status)

	expectedSQL := "SELECT id, date_time, pvz_id, status FROM receptions WHERE pvz_id=$1 AND status='in_progress' ORDER BY date_time DESC LIMIT 1"
	mock.ExpectQuery(expectedSQL).WithArgs(pvzID).WillReturnRows(rows)

	actualReception, err := repo.GetOpenReception(pvzID)
	assert.NoError(t, err)
	assert.Equal(t, expectedReception, actualReception)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_GetOpenReception_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	pvzID := uuid.New().String()

	expectedSQL := "SELECT id, date_time, pvz_id, status FROM receptions WHERE pvz_id=$1 AND status='in_progress' ORDER BY date_time DESC LIMIT 1"
	mock.ExpectQuery(expectedSQL).WithArgs(pvzID).WillReturnError(sql.ErrNoRows)

	actualReception, err := repo.GetOpenReception(pvzID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, actualReception)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_CloseReception_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	receptionID := uuid.New().String()

	expectedSQL := "UPDATE receptions SET status='close' WHERE id=$1"
	mock.ExpectExec(expectedSQL).
		WithArgs(receptionID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.CloseReception(receptionID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_CloseReception_NotUpdated(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	receptionID := uuid.New().String()

	expectedSQL := "UPDATE receptions SET status='close' WHERE id=$1"
	mock.ExpectExec(expectedSQL).
		WithArgs(receptionID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.CloseReception(receptionID)
	assert.Error(t, err)
	assert.EqualError(t, err, "no reception updated")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceptionRepo_CloseReception_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewReceptionRepository(db)
	receptionID := uuid.New().String()
	expectedError := errors.New("rows affected error")

	mockResult := sqlmock.NewErrorResult(expectedError)

	expectedSQL := "UPDATE receptions SET status='close' WHERE id=$1"
	mock.ExpectExec(expectedSQL).
		WithArgs(receptionID).
		WillReturnResult(mockResult)

	err = repo.CloseReception(receptionID)
	assert.Error(t, err)

	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
