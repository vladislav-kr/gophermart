package passwordgenerator

import (
	"errors"
	"fmt"

	"github.com/vladislav-kr/gofermart-bonus/internal/domain/models"
	"golang.org/x/crypto/bcrypt"
)

type generator struct {
	// bcrypt package
	// const (
	// 	MinCost     int = 4  // the minimum allowable cost as passed in to GenerateFromPassword
	// 	MaxCost     int = 31 // the maximum allowable cost as passed in to GenerateFromPassword
	// 	DefaultCost int = 10 // the cost that will actually be set if a cost below MinCost is passed into GenerateFromPassword
	// )
	cost int
}

func New(cost int) *generator {
	return &generator{
		cost: cost,
	}
}

func (g *generator) CompareHashAndPassword(hashedPassword, password []byte) error {
	if err := bcrypt.CompareHashAndPassword(
		hashedPassword,
		password,
	); err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return models.ErrMismatchedHashAndPassword
		default:
			return fmt.Errorf("compare hash and password: %w", err)
		}
	}
	return nil
}
func (g *generator) GenerateFromPassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, g.cost)
}
