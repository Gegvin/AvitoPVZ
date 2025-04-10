package repository

import (
	"database/sql"
	"errors"

	"AvitoPVZ/internal/models"
)

type ProductRepository interface {
	AddProduct(product *models.Product) error
	DeleteLastProduct(pvzId string) error
}

type productRepo struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) AddProduct(product *models.Product) error {
	query := `INSERT INTO products (id, date_time, type, reception_id) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, product.ID, product.DateTime, product.Type, product.ReceptionID)
	return err
}

func (r *productRepo) DeleteLastProduct(pvzId string) error {
	// Удаляем последний добавленный товар из открытой приёмки в ПВЗ по принципу LIFO
	query := `
	DELETE FROM products
	WHERE id = (
		SELECT p.id FROM products p
		JOIN receptions r ON p.reception_id = r.id
		WHERE r.pvz_id = $1 AND r.status = 'in_progress'
		ORDER BY p.date_time DESC
		LIMIT 1
	)
	`
	res, err := r.db.Exec(query, pvzId)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		return errors.New("no product to delete")
	}
	return nil
}
