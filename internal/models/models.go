package models

import (
	"github.com/google/uuid"
	"time"
)

var (
	NEW        = "NEW"
	REGISTERED = "REGISTERED"
	INVALID    = "INVALID"
	PROCESSING = "PROCESSING"
	PROCESSED  = "PROCESSED"
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type Order struct {
	ID          int       `db:"id"`
	UserID      uuid.UUID `db:"user_id"`
	OrderNumber string    `db:"order_number"`
	Status      string    `db:"status"`
	Accrual     float64   `db:"accrual"`
	UploadedAt  time.Time `db:"uploaded_at"`
}

type UserBalance struct {
	ID             int       `db:"id"`
	UserID         uuid.UUID `db:"user_id"`
	CurrentBalance float64   `db:"current_balance"`
	WithdrawnTotal float64   `db:"withdrawn_total"`
}

type Withdrawal struct {
	ID          int       `db:"id"`
	UserID      uuid.UUID `db:"user_id"`
	OrderNumber string    `db:"order_number"`
	Sum         float64   `db:"sum"`
	ProcessedAt time.Time `db:"processed_at"`
}
