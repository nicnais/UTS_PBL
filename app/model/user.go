package model

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("data tidak ditemukan")
	ErrConflict     = errors.New("data sudah ada atau berbenturan")
	ErrInvalidToken = errors.New("refresh token tidak aktif")
)

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	Permissions  []string  `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u User) Can(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type LaundryService struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	PricePerKg int64     `json:"price_per_kg"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}
type CreateServiceRequest struct {
	Name       string `json:"name"`
	PricePerKg int64  `json:"price_per_kg"`
}
type PatchServiceRequest struct {
	Name       *string `json:"name"`
	PricePerKg *int64  `json:"price_per_kg"`
	IsActive   *bool   `json:"is_active"`
}

type Order struct {
	ID                 int64     `json:"id"`
	CustomerID         int64     `json:"customer_id"`
	ServiceID          int64     `json:"service_id"`
	WeightKg           *string   `json:"weight_kg"`
	PricePerKgSnapshot *int64    `json:"price_per_kg_snapshot"`
	TotalPrice         *int64    `json:"total_price"`
	Status             string    `json:"status"`
	PaymentStatus      string    `json:"payment_status"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
type CreateOrderRequest struct {
	ServiceID int64  `json:"service_id"`
	Notes     string `json:"notes"`
}
type PatchOrderRequest struct {
	WeightKg      *json.Number `json:"weight_kg"`
	Status        *string      `json:"status"`
	PaymentStatus *string      `json:"payment_status"`
}

type OrderChange struct {
	WeightKg           *string
	PricePerKgSnapshot *int64
	TotalPrice         *int64
	Status             string
	PaymentStatus      string
}
type OrderFilter struct {
	CustomerID    *int64
	Status        string
	Limit, Offset int
}
