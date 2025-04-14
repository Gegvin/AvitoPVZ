package db

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"AvitoPVZ/internal/config"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestConnect_Success(t *testing.T) {

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectPing()

	cfg := &config.Config{
		DBHost:     "mockhost",
		DBPort:     1234,
		DBUser:     "mockuser",
		DBPassword: "mockpassword",
		DBName:     "mockdb",
	}

	mockOpener := func(driverName, dataSourceName string) (*sql.DB, error) {

		assert.Equal(t, "postgres", driverName)
		expectedDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
		assert.Equal(t, expectedDSN, dataSourceName)

		return db, nil
	}

	connectedDb, err := Connect(cfg, mockOpener)

	assert.NoError(t, err)
	assert.NotNil(t, connectedDb)

	assert.Same(t, db, connectedDb)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err, "sqlmock expectations were not met")
}

func TestConnect_OpenError(t *testing.T) {

	cfg := &config.Config{
		DBHost: "mockhost", DBPort: 1234, DBUser: "mockuser", DBPassword: "mockpassword", DBName: "mockdb",
	}
	expectedError := errors.New("failed to open connection")

	mockOpener := func(driverName, dataSourceName string) (*sql.DB, error) {

		assert.Equal(t, "postgres", driverName)
		return nil, expectedError
	}

	connectedDb, err := Connect(cfg, mockOpener)

	assert.Error(t, err)
	assert.Nil(t, connectedDb)
	assert.Equal(t, expectedError, err)
}

func TestConnect_PingError(t *testing.T) {

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	expectedError := errors.New("ping failed")
	mock.ExpectPing().WillReturnError(expectedError)

	cfg := &config.Config{
		DBHost: "mockhost", DBPort: 1234, DBUser: "mockuser", DBPassword: "mockpassword", DBName: "mockdb",
	}

	mockOpener := func(driverName, dataSourceName string) (*sql.DB, error) {
		assert.Equal(t, "postgres", driverName)
		return db, nil
	}

	connectedDb, err := Connect(cfg, mockOpener)

	assert.Error(t, err)
	assert.Nil(t, connectedDb)
	assert.EqualError(t, err, expectedError.Error())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err, "sqlmock expectations were not met")
}
