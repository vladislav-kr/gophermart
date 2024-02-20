package models

import (
	"fmt"
	"regexp"
)

var (
	regLogin *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9-_\.]{4,30}$`)
	regPass  *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9-_\.]{7,32}$`)
)

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (c Credentials) Validate() error {
	if ok := regLogin.MatchString(c.Login) && regPass.MatchString(c.Password); !ok {
		return fmt.Errorf("invalid login or password")
	}
	return nil
}
