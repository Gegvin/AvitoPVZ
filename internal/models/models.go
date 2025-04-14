package models

import "time"

type User struct {
	ID       string
	Email    string
	Password string
	Role     string
}

type AllowedCity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PVZ struct {
	ID               string    `json:"id"`
	RegistrationDate time.Time `json:"registrationDate"`
	CityID           int       `json:"-"`
	CityName         string    `json:"city"`
}

type Reception struct {
	ID       string    `json:"id"`
	DateTime time.Time `json:"dateTime"`
	PVZID    string    `json:"pvzId"`
	Status   string    `json:"status"`
}

type ProductType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Product struct {
	ID          string    `json:"id"`
	DateTime    time.Time `json:"dateTime"`
	ReceptionID string    `json:"receptionId"`
	TypeID      int       `json:"-"`
	TypeName    string    `json:"type,omitempty"`
}
