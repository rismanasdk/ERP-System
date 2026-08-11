package password

import (
	"errors"

	"github.com/alexedwards/argon2id"
)

func Hash(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func Compare(hashedPassword, password string) error {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	if err != nil {
		return err
	}
	if !match {
		return errors.New("invalid credentials")
	}
	return nil
}
