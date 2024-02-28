package clients

type OrderAccrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accural float64 `json:"accrual,omitempty"`
}
