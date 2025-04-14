package repository

import (
	"database/sql"
	"errors"

	"AvitoPVZ/internal/models"
)

// ReceptionRepository определяет интерфейс для работы с данными приемок.
type ReceptionRepository interface {
	OpenReceptionExists(pvzId string) (bool, error)
	CreateReception(reception *models.Reception) error
	GetOpenReception(pvzId string) (*models.Reception, error)
	CloseReception(receptionId string) error
}

type receptionRepo struct {
	db *sql.DB
}

// NewReceptionRepository создает новый экземпляр ReceptionRepository.
func NewReceptionRepository(db *sql.DB) ReceptionRepository {
	return &receptionRepo{db: db}
}

// OpenReceptionExists проверяет, существует ли открытая приемка для данного ПВЗ.
func (r *receptionRepo) OpenReceptionExists(pvzId string) (bool, error) {
	query := `SELECT id FROM receptions WHERE pvz_id=$1 AND status='in_progress'`
	row := r.db.QueryRow(query, pvzId)
	var id string
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // Явная проверка на sql.ErrNoRows
			return false, nil // Не найдено - не ошибка
		}
		return false, err // Другая ошибка БД
	}
	return true, nil // Найдено
}

// CreateReception создает новую запись о приемке в БД.
func (r *receptionRepo) CreateReception(reception *models.Reception) error {
	query := `INSERT INTO receptions (id, date_time, pvz_id, status) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, reception.ID, reception.DateTime, reception.PVZID, reception.Status)
	return err
}

// GetOpenReception возвращает последнюю открытую приемку для ПВЗ.
// Возвращает sql.ErrNoRows, если открытых приемок нет.
func (r *receptionRepo) GetOpenReception(pvzId string) (*models.Reception, error) {
	query := `SELECT id, date_time, pvz_id, status FROM receptions WHERE pvz_id=$1 AND status='in_progress' ORDER BY date_time DESC LIMIT 1`
	row := r.db.QueryRow(query, pvzId)
	reception := &models.Reception{}
	err := row.Scan(&reception.ID, &reception.DateTime, &reception.PVZID, &reception.Status)
	if err != nil {
		return nil, err // Ошибка включает sql.ErrNoRows
	}
	return reception, nil
}

// CloseReception закрывает приемку по ее ID, изменяя статус на 'close'.
// Возвращает кастомную ошибку "no reception updated", если ни одна строка не была обновлена.
func (r *receptionRepo) CloseReception(receptionId string) error {
	query := `UPDATE receptions SET status='close' WHERE id=$1`
	res, err := r.db.Exec(query, receptionId)
	if err != nil {
		return err // Ошибка выполнения запроса
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err // Ошибка при получении кол-ва затронутых строк
	}

	if affected == 0 {
		// Возвращаем специфичную ошибку, если ничего не обновилось
		return errors.New("no reception updated")
	}

	return nil // Успех
}
