package auth

import (
	"crypto/rand"
	"crypto/rsa"
)

func generateTestRSAKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}
