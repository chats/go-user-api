package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomPassword generates a secure random password of a specified length
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8 // Minimum length for security
	}

	// Generate random bytes
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode to base64 and take only the first 'length' characters
	password := base64.URLEncoding.EncodeToString(randomBytes)
	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword checks if the provided password matches the stored hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
