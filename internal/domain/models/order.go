package models

import (
	"strconv"
	"time"
)

type OrderID string

func (o OrderID) Validate() bool {
	if len(o) == 0 {
		return false
	}
	numbers := []rune(o)
	result := 0
	idx := 0
	for i := len(numbers) - 1; i >= 0; i-- {
		idx++
		number, err := strconv.Atoi(string(numbers[i]))
		if err != nil {
			return false
		}
		if idx%2 == 0 {
			number *= 2
			if number > 9 {
				number -= 9
			}

		}
		result += number
	}

	return result%10 == 0
}

type Order struct {
	OrderID    OrderID   `json:"number"`
	Status     string    `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	Accrual    float64   `json:"accrual,omitempty"`
}

type UpdateOrderID struct {
	OrderID OrderID
}

const (
	StatusNew        string = "NEW"        // заказ зарегистрирован, но вознаграждение не рассчитано;
	StatusInvalid    string = "INVALID"    // заказ не принят к расчёту, и вознаграждение не будет начислено;
	StatusProcessing string = "PROCESSING" // расчёт начисления в процессе;
	StatusProcessed  string = "PROCESSED"  // расчёт начисления окончен;
)
