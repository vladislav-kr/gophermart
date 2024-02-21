package storage

import "time"

type Balance struct {
	Current   float64 `json:"current" db:"current"`
	Withdrawn float64 `json:"withdrawn" db:"withdrawn"`
}

type CreateOrder struct {
	OrderID string
	// UserID  string
	Status  string
	Accrual float64
}
type UpdateOrderID struct {
	UserID  string `db:"user_id"`
	OrderID string `db:"order_id"`
}

type UpdateOrder struct {
	UserID  string  `db:"user_id"`
	OrderID string  `db:"order_id"`
	Status  string  `db:"status"`
	Accrual float64 `db:"accrual"`
}

type Order struct {
	OrderID    string    `db:"order_id"`
	UserID     string    `db:"user_id"`
	Status     string    `db:"status"`
	UploadedAt time.Time `db:"uploaded_at"`
	ChangedAt  time.Time `db:"changed_at"`
	Accrual    float64   `db:"accrual"`
}

type User struct {
	UserID   string `db:"user_id"`
	Login    string `db:"login"`
	Password []byte `db:"pass_hash"`
}

type WithdrawalsBonuses struct {
	Order       string    `db:"order_id"`
	Sum         float64   `db:"sum"`
	ProcessedAt time.Time `db:"processed_at"`
}

type WithdrawBonuses struct {
	Order string  `db:"order_id"`
	Sum   float64 `db:"sum"`
}
