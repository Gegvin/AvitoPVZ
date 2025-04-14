package repository

import (
	"database/sql"
	// "AvitoPVZ/internal/models" // Если GetAllProductTypes раскомментирован
)

// ProductTypeRepository определяет интерфейс для работы с типами товаров.
type ProductTypeRepository interface {
	GetProductTypeIDByName(typeName string) (int, error)
	// GetAllProductTypes() ([]models.ProductType, error) // Пример возможного метода
}

type productTypeRepo struct {
	db *sql.DB
}

// NewProductTypeRepository создает новый экземпляр ProductTypeRepository.
func NewProductTypeRepository(db *sql.DB) ProductTypeRepository {
	return &productTypeRepo{db: db}
}

// GetProductTypeIDByName находит ID типа товара по его имени.
// Возвращает sql.ErrNoRows, если тип не найден.
func (r *productTypeRepo) GetProductTypeIDByName(typeName string) (int, error) {
	query := `SELECT id FROM product_types WHERE name = $1`
	var typeID int
	err := r.db.QueryRow(query, typeName).Scan(&typeID)
	if err != nil {
		return 0, err // Возвращаем ошибку как есть (включая sql.ErrNoRows)
	}
	return typeID, nil
}
