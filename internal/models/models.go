package models

import "time"

// User представляет пользователя системы.
type User struct {
	ID       string
	Email    string
	Password string // Хешированный пароль
	Role     string // "employee" или "moderator"
}

// AllowedCity представляет город, разрешенный для регистрации ПВЗ.
type AllowedCity struct {
	ID   int    `json:"id"`   // Внутренний ID
	Name string `json:"name"` // Название города
}

// PVZ представляет пункт выдачи заказов.
type PVZ struct {
	ID               string    `json:"id"`
	RegistrationDate time.Time `json:"registrationDate"`
	CityID           int       `json:"-"`    // Внешний ключ к allowed_cities (скрыт из JSON)
	CityName         string    `json:"city"` // Название города (заполняется при выборке)
}

// Reception представляет приемку товаров в ПВЗ.
type Reception struct {
	ID       string    `json:"id"`
	DateTime time.Time `json:"dateTime"`
	PVZID    string    `json:"pvzId"`
	Status   string    `json:"status"` // "in_progress" или "close"
}

// ProductType представляет тип товара.
type ProductType struct {
	ID   int    `json:"id"`   // Внутренний ID
	Name string `json:"name"` // Название типа (напр., "электроника")
}

// Product представляет товар, принятый в ПВЗ.
type Product struct {
	ID          string    `json:"id"`
	DateTime    time.Time `json:"dateTime"`
	ReceptionID string    `json:"receptionId"`
	TypeID      int       `json:"-"` // Внутренний ID типа товара (скрыт из JSON)
	// TypeName заполняется при выборке для ответа API.
	TypeName string `json:"type,omitempty"` // Название типа для JSON ответа
}
