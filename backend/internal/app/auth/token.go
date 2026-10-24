package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

func newToken() (string, []byte, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", nil, err
	}

	hash := sha256.Sum256(value)

	return base64.RawURLEncoding.EncodeToString(value), hash[:], nil
}

func tokenHash(token string) ([]byte, error) {
	value, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(value) != 32 {
		return nil, errors.New("invalid token")
	}

	hash := sha256.Sum256(value)

	return hash[:], nil
}
