package auth

import (
	"golang.org/x/crypto/bcrypt"
)

type PasswordVerifier struct{}

func (PasswordVerifier) Verify(hash string, plainText string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainText))
}

func HashPassword(plainText string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainText), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
