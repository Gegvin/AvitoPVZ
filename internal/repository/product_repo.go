package repository

import (
	"database/sql"
	"errors"

	"AvitoPVZ/internal/models"
)

// ProductRepository определяет интерфейс для работы с данными товаров.
type ProductRepository interface {
	AddProduct(product *models.Product) error
	DeleteLastProduct(pvzId string) error
	// Возможно, понадобятся методы GetProductByID или GetProductsByReceptionID
}

type productRepo struct {
	db *sql.DB
}

// NewProductRepository создает новый экземпляр ProductRepository.
func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepo{db: db}
}

// AddProduct добавляет новый товар в базу данных, используя TypeID.
func (r *productRepo) AddProduct(product *models.Product) error {
	query := `INSERT INTO products (id, date_time, reception_id, type_id) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, product.ID, product.DateTime, product.ReceptionID, product.TypeID)
	return err
}

// DeleteLastProduct удаляет последний добавленный товар из открытой приемки для ПВЗ.
// Возвращает кастомную ошибку "no product to delete", если товар для удаления не найден.
func (r *productRepo) DeleteLastProduct(pvzId string) error {
	// Сложный запрос для нахождения ID последнего товара в открытой приемке ПВЗ
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
		return err // Ошибка выполнения запроса
	}

	affected, err := res.RowsAffected()
	if err != nil {
		// Игнорируем sql.ErrNoRows от RowsAffected, если драйвер его возвращает
		if errors.Is(err, sql.ErrNoRows) {
			// Если RowsAffected вернул ErrNoRows, это эквивалентно affected == 0
		} else {
			return err // Другая ошибка получения RowsAffected
		}
	}

	// Если 0 строк затронуто (или RowsAffected вернул ErrNoRows)
	if affected == 0 {
		return errors.New("no product to delete")
	}

	return nil // Успех
}
