package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"AvitoPVZ/internal/models"
)

// PVZRepository определяет интерфейс для работы с данными ПВЗ.
type PVZRepository interface {
	GetCityIDByName(cityName string) (int, error)
	GetAllowedCities() ([]models.AllowedCity, error)
	CreatePVZ(pvz *models.PVZ) error
	GetPVZList(params url.Values) ([]models.PVZ, error) // Принимает параметры запроса
	GetAllPVZ() ([]models.PVZ, error)
}

type pvzRepo struct {
	db *sql.DB
}

func NewPVZRepository(db *sql.DB) PVZRepository {
	return &pvzRepo{db: db}
}

// GetCityIDByName получает ID города по его названию.
func (r *pvzRepo) GetCityIDByName(cityName string) (int, error) {
	query := `SELECT id FROM allowed_cities WHERE name = $1`
	var cityID int
	err := r.db.QueryRow(query, cityName).Scan(&cityID)
	if err != nil {
		return 0, err // Включая sql.ErrNoRows
	}
	return cityID, nil
}

// GetAllowedCities возвращает список всех разрешенных городов.
func (r *pvzRepo) GetAllowedCities() ([]models.AllowedCity, error) {
	query := `SELECT id, name FROM allowed_cities ORDER BY name`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []models.AllowedCity
	for rows.Next() {
		var c models.AllowedCity
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				return nil, rowsErr
			} // Ошибка итерации
			return nil, err // Ошибка сканирования
		}
		cities = append(cities, c)
	}
	if err = rows.Err(); err != nil { // Ошибка после итерации
		return nil, err
	}
	return cities, nil
}

// CreatePVZ добавляет новый ПВЗ в базу данных.
func (r *pvzRepo) CreatePVZ(pvz *models.PVZ) error {
	query := `INSERT INTO pvz (id, registration_date, city_id) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(query, pvz.ID, pvz.RegistrationDate, pvz.CityID)
	return err
}

// Парсит параметры 'limit', 'offset', 'receptionDate' из url.Values.
func (r *pvzRepo) GetPVZList(params url.Values) ([]models.PVZ, error) {
	// Парсинг параметров
	limitStr := params.Get("limit")
	offsetStr := params.Get("offset")
	dateStr := params.Get("receptionDate")

	// Значения по умолчанию
	limit := 10
	offset := 0
	var receptionDate *time.Time

	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l <= 0 {
			return nil, errors.New("invalid parameter: 'limit' must be a positive integer")
		}
		limit = l
	}
	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil || o < 0 {
			return nil, errors.New("invalid parameter: 'offset' must be a non-negative integer")
		}
		offset = o
	}
	if dateStr != "" {
		layout := "2006-01-02"
		t, err := time.Parse(layout, dateStr)
		if err != nil {
			return nil, errors.New("invalid parameter: 'receptionDate' format (use YYYY-MM-DD)")
		}
		receptionDate = &t
	}

	// SQL запрос
	baseQuery := `SELECT DISTINCT p.id, p.registration_date, p.city_id, c.name
                  FROM pvz p
                  JOIN allowed_cities c ON p.city_id = c.id`
	var args []interface{}
	var whereClauses []string
	placeholderCount := 1

	if receptionDate != nil {

		baseQuery += ` JOIN receptions r ON p.id = r.pvz_id`
		whereClauses = append(whereClauses, fmt.Sprintf("DATE(r.date_time) = $%d", placeholderCount))
		args = append(args, receptionDate.Format("2006-01-02"))
		placeholderCount++
	}

	if len(whereClauses) > 0 {
		baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	baseQuery += " ORDER BY p.registration_date DESC" // Сортировка по умолчанию

	// Пагинация
	baseQuery += fmt.Sprintf(" LIMIT $%d", placeholderCount)
	args = append(args, limit)
	placeholderCount++
	baseQuery += fmt.Sprintf(" OFFSET $%d", placeholderCount)
	args = append(args, offset)

	// Выполнение запроса
	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		// Можно добавить логирование SQL и аргументов для отладки
		// log.Printf("SQL Query failed: %s\nArgs: %v\nError: %v", baseQuery, args, err)
		return nil, err
	}
	defer rows.Close()

	var pvzs []models.PVZ
	for rows.Next() {
		var p models.PVZ
		if err := rows.Scan(&p.ID, &p.RegistrationDate, &p.CityID, &p.CityName); err != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				return nil, rowsErr
			}
			return nil, err
		}
		pvzs = append(pvzs, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return pvzs, nil
}

// GetAllPVZ возвращает все ПВЗ без пагинации/фильтрации
func (r *pvzRepo) GetAllPVZ() ([]models.PVZ, error) {
	return r.GetPVZList(nil) // Передаем nil для использования значений по умолчанию
}
