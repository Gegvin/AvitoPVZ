package repository

import (
	"database/sql"
	"errors"

	"AvitoPVZ/internal/models"
)

type ReceptionRepository interface {
	OpenReceptionExists(pvzId string) (bool, error)
	CreateReception(reception *models.Reception) error
	GetOpenReception(pvzId string) (*models.Reception, error)
	CloseReception(receptionId string) error
}

type receptionRepo struct {
	db *sql.DB
}

func NewReceptionRepository(db *sql.DB) ReceptionRepository {
	return &receptionRepo{db: db}
}

func (r *receptionRepo) OpenReceptionExists(pvzId string) (bool, error) {
	query := `SELECT id FROM receptions WHERE pvz_id=$1 AND status='in_progress'`
	row := r.db.QueryRow(query, pvzId)
	var id string
	err := row.Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *receptionRepo) CreateReception(reception *models.Reception) error {
	query := `INSERT INTO receptions (id, date_time, pvz_id, status) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, reception.ID, reception.DateTime, reception.PVZID, reception.Status)
	return err
}

func (r *receptionRepo) GetOpenReception(pvzId string) (*models.Reception, error) {
	query := `SELECT id, date_time, pvz_id, status FROM receptions WHERE pvz_id=$1 AND status='in_progress' ORDER BY date_time DESC LIMIT 1`
	row := r.db.QueryRow(query, pvzId)
	reception := &models.Reception{}
	err := row.Scan(&reception.ID, &reception.DateTime, &reception.PVZID, &reception.Status)
	if err != nil {
		return nil, err
	}
	return reception, nil
}

func (r *receptionRepo) CloseReception(receptionId string) error {
	query := `UPDATE receptions SET status='close' WHERE id=$1`
	res, err := r.db.Exec(query, receptionId)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		return errors.New("no reception updated")
	}
	return nil
}
