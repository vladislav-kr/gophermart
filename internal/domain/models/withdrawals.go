package models

import "time"

type WithdrawalsBonuses struct {
	Order       OrderID   `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type WithdrawBonuses struct {
	Order OrderID `json:"order"`
	Sum   float64 `json:"sum"`
}
