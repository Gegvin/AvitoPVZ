package repository

import (
	"database/sql"
	"net/url"

	"AvitoPVZ/internal/models"
)

type PVZRepository interface {
	CreatePVZ(pvz *models.PVZ) error
	GetPVZList(params url.Values) ([]models.PVZ, error)
	GetAllPVZ() ([]models.PVZ, error)
}

type pvzRepo struct {
	db *sql.DB
}

func NewPVZRepository(db *sql.DB) PVZRepository {
	return &pvzRepo{db: db}
}

func (r *pvzRepo) CreatePVZ(pvz *models.PVZ) error {
	query := `INSERT INTO pvz (id, registration_date, city) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(query, pvz.ID, pvz.RegistrationDate, pvz.City)
	return err
}

func (r *pvzRepo) GetPVZList(params url.Values) ([]models.PVZ, error) {
	// Для простоты возвращаем все записи
	query := `SELECT id, registration_date, city FROM pvz`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pvzs []models.PVZ
	for rows.Next() {
		var p models.PVZ
		if err := rows.Scan(&p.ID, &p.RegistrationDate, &p.City); err != nil {
			return nil, err
		}
		pvzs = append(pvzs, p)
	}
	return pvzs, nil
}

func (r *pvzRepo) GetAllPVZ() ([]models.PVZ, error) {
	return r.GetPVZList(nil)
}
