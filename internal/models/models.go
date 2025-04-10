package models

import "time"

type User struct {
	ID       string
	Email    string
	Password string // Хешированный пароль
	Role     string // "employee" или "moderator"
}

type PVZ struct {
	ID               string
	RegistrationDate time.Time
	City             string // "Москва", "Санкт-Петербург", "Казань"
}

type Reception struct {
	ID       string
	DateTime time.Time
	PVZID    string
	Status   string // "in_progress" или "close"
}

type Product struct {
	ID          string
	DateTime    time.Time
	Type        string // "электроника", "одежда", "обувь"
	ReceptionID string
}
