package utils

import (
	"fmt"

	"github.com/google/uuid"
)

// StringToUUID converts a string to a UUID
func StringToUUID(id string) (uuid.UUID, error) {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		// ใช้ UUID ว่างเปล่าแทน uuid.Nil
		return uuid.UUID{}, fmt.Errorf("invalid UUID format: %w", err)
	}
	return parsedUUID, nil
}
