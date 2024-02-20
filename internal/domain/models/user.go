package models

import "github.com/google/uuid"

type UserID string

func (u UserID) Validate() bool {
	_, err := uuid.Parse(string(u))
	return err == nil
}
