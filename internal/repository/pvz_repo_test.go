package repository

import (
	"database/sql"
	"errors"
	"net/url"

	"strconv"
	"testing"
	"time"

	"AvitoPVZ/internal/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPVZRepo_GetCityIDByName_Found(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	cityName := "Москва"
	expectedID := 1
	rows := sqlmock.NewRows([]string{"id"}).AddRow(expectedID)
	expectedSQL := "SELECT id FROM allowed_cities WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(cityName).WillReturnRows(rows)
	actualID, err := repo.GetCityIDByName(cityName)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetCityIDByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	cityName := "Неизвестный Город"
	expectedSQL := "SELECT id FROM allowed_cities WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(cityName).WillReturnError(sql.ErrNoRows)
	actualID, err := repo.GetCityIDByName(cityName)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Equal(t, 0, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetCityIDByName_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	cityName := "Москва"
	expectedError := errors.New("database connection error")
	expectedSQL := "SELECT id FROM allowed_cities WHERE name = $1"
	mock.ExpectQuery(expectedSQL).WithArgs(cityName).WillReturnError(expectedError)
	actualID, err := repo.GetCityIDByName(cityName)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, 0, actualID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetAllowedCities_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	expectedCities := []models.AllowedCity{{ID: 3, Name: "Казань"}, {ID: 1, Name: "Москва"}, {ID: 2, Name: "Санкт-Петербург"}}
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(expectedCities[0].ID, expectedCities[0].Name).AddRow(expectedCities[1].ID, expectedCities[1].Name).AddRow(expectedCities[2].ID, expectedCities[2].Name)
	expectedSQL := "SELECT id, name FROM allowed_cities ORDER BY name"
	mock.ExpectQuery(expectedSQL).WillReturnRows(rows)
	actualCities, err := repo.GetAllowedCities()
	assert.NoError(t, err)
	assert.Equal(t, expectedCities, actualCities)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetAllowedCities_Empty(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	rows := sqlmock.NewRows([]string{"id", "name"})
	expectedSQL := "SELECT id, name FROM allowed_cities ORDER BY name"
	mock.ExpectQuery(expectedSQL).WillReturnRows(rows)
	actualCities, err := repo.GetAllowedCities()
	assert.NoError(t, err)
	assert.Empty(t, actualCities)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetAllowedCities_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	expectedError := errors.New("failed to query cities")
	expectedSQL := "SELECT id, name FROM allowed_cities ORDER BY name"
	mock.ExpectQuery(expectedSQL).WillReturnError(expectedError)
	actualCities, err := repo.GetAllowedCities()
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, actualCities)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestPVZRepo_GetAllowedCities_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Москва").AddRow("invalid-id", "Санкт-Петербург")
	expectedSQL := "SELECT id, name FROM allowed_cities ORDER BY name"
	mock.ExpectQuery(expectedSQL).WillReturnRows(rows)
	actualCities, err := repo.GetAllowedCities()
	assert.Error(t, err)
	assert.Nil(t, actualCities)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_CreatePVZ(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	pvz := &models.PVZ{ID: uuid.New().String(), RegistrationDate: time.Now().Truncate(time.Millisecond), CityID: 1, CityName: "Москва"}
	expectedSQL := "INSERT INTO pvz (id, registration_date, city_id) VALUES ($1, $2, $3)"
	mock.ExpectExec(expectedSQL).WithArgs(pvz.ID, pvz.RegistrationDate, pvz.CityID).WillReturnResult(sqlmock.NewResult(1, 1))
	err = repo.CreatePVZ(pvz)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_Defaults(t *testing.T) {

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	expectedPVZs := []models.PVZ{
		{ID: uuid.New().String(), CityID: 1, CityName: "Москва", RegistrationDate: time.Now().Truncate(time.Millisecond)},
	}

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"}).
		AddRow(expectedPVZs[0].ID, expectedPVZs[0].RegistrationDate, expectedPVZs[0].CityID, expectedPVZs[0].CityName)

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`

	mock.ExpectQuery(expectedExactSQL).
		WithArgs(10, 0).
		WillReturnRows(rows)

	params := url.Values{}
	actualPVZs, err := repo.GetPVZList(params)

	assert.NoError(t, err)
	assert.Equal(t, expectedPVZs, actualPVZs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_WithPagination(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	limit := 5
	offset := 10

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"})

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`

	mock.ExpectQuery(expectedExactSQL).
		WithArgs(limit, offset).
		WillReturnRows(rows)

	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	actualPVZs, err := repo.GetPVZList(params)

	assert.NoError(t, err)
	assert.Empty(t, actualPVZs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_WithDateFilter(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	dateStr := "2025-04-14"
	limit := 15
	offset := 0

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"})

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id JOIN receptions r ON p.id = r.pvz_id WHERE DATE(r.date_time) = $1 ORDER BY p.registration_date DESC LIMIT $2 OFFSET $3`

	mock.ExpectQuery(expectedExactSQL).
		WithArgs(dateStr, limit, offset).
		WillReturnRows(rows)

	params := url.Values{}
	params.Set("receptionDate", dateStr)
	params.Set("limit", strconv.Itoa(limit))

	actualPVZs, err := repo.GetPVZList(params)

	assert.NoError(t, err)
	assert.Empty(t, actualPVZs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_WithDateFilterAndPagination(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	dateStr := "2025-04-15"
	limit := 3
	offset := 6

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"})

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id JOIN receptions r ON p.id = r.pvz_id WHERE DATE(r.date_time) = $1 ORDER BY p.registration_date DESC LIMIT $2 OFFSET $3`

	mock.ExpectQuery(expectedExactSQL).
		WithArgs(dateStr, limit, offset).
		WillReturnRows(rows)

	params := url.Values{}
	params.Set("receptionDate", dateStr)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	actualPVZs, err := repo.GetPVZList(params)

	assert.NoError(t, err)
	assert.Empty(t, actualPVZs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_InvalidParams(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)

	testCases := []struct {
		name        string
		params      url.Values
		expectedErr string
	}{
		{name: "Invalid limit", params: url.Values{"limit": []string{"abc"}}, expectedErr: "invalid parameter: 'limit' must be a positive integer"},
		{name: "Negative limit", params: url.Values{"limit": []string{"-5"}}, expectedErr: "invalid parameter: 'limit' must be a positive integer"},
		{name: "Invalid offset", params: url.Values{"offset": []string{"xyz"}}, expectedErr: "invalid parameter: 'offset' must be a non-negative integer"},
		{name: "Negative offset", params: url.Values{"offset": []string{"-1"}}, expectedErr: "invalid parameter: 'offset' must be a non-negative integer"},
		{name: "Invalid date format", params: url.Values{"receptionDate": []string{"14-04-2025"}}, expectedErr: "invalid parameter: 'receptionDate' format (use YYYY-MM-DD)"},
		{name: "Invalid date value", params: url.Values{"receptionDate": []string{"2025-02-30"}}, expectedErr: "invalid parameter: 'receptionDate' format (use YYYY-MM-DD)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.GetPVZList(tc.params)
			assert.Error(t, err)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestPVZRepo_GetPVZList_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"}).
		AddRow(uuid.New().String(), time.Now(), "not-an-int", "Москва")

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`
	mock.ExpectQuery(expectedExactSQL).WithArgs(10, 0).WillReturnRows(rows)

	params := url.Values{}
	_, err = repo.GetPVZList(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sql: Scan error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_RowsErr(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	expectedCloseError := errors.New("rows iteration error")

	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"}).
		AddRow(uuid.New().String(), time.Now(), 1, "Москва").
		CloseError(expectedCloseError)

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`
	mock.ExpectQuery(expectedExactSQL).WithArgs(10, 0).WillReturnRows(rows)

	params := url.Values{}
	_, err = repo.GetPVZList(params)

	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedCloseError)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetPVZList_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()
	repo := NewPVZRepository(db)
	queryFailError := errors.New("database query failed")

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`
	mock.ExpectQuery(expectedExactSQL).WithArgs(10, 0).WillReturnError(queryFailError)

	params := url.Values{}
	_, err = repo.GetPVZList(params)
	assert.Error(t, err)
	assert.Equal(t, queryFailError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPVZRepo_GetAllPVZ(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	repo := NewPVZRepository(db)
	expectedPVZs := []models.PVZ{
		{ID: uuid.New().String(), CityID: 1, CityName: "Москва", RegistrationDate: time.Now().Truncate(time.Millisecond)},
	}
	rows := sqlmock.NewRows([]string{"id", "registration_date", "city_id", "name"}).
		AddRow(expectedPVZs[0].ID, expectedPVZs[0].RegistrationDate, expectedPVZs[0].CityID, expectedPVZs[0].CityName)

	expectedExactSQL := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id ORDER BY p.registration_date DESC LIMIT $1 OFFSET $2`

	mock.ExpectQuery(expectedExactSQL).WithArgs(10, 0).WillReturnRows(rows)

	actualPVZs, err := repo.GetAllPVZ()

	assert.NoError(t, err)
	assert.Equal(t, expectedPVZs, actualPVZs)
	assert.NoError(t, mock.ExpectationsWereMet())
}
