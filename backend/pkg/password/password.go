package password

import "github.com/alexedwards/argon2id"

func Hash(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func Compare(hashedPassword, password string) error {
	_, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	return err
}
