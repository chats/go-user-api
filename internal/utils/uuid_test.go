package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStringToUUID(t *testing.T) {
	// ทดสอบการแปลง valid UUID string
	validUUIDStr := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	expectedUUID, _ := uuid.Parse(validUUIDStr)

	result, err := StringToUUID(validUUIDStr)
	assert.NoError(t, err)
	assert.Equal(t, expectedUUID, result)

	// ทดสอบการแปลง invalid UUID string
	invalidUUIDStr := "not-a-valid-uuid"
	_, err = StringToUUID(invalidUUIDStr)
	assert.Error(t, err)

	// ทดสอบการแปลง empty string
	emptyUUIDStr := ""
	_, err = StringToUUID(emptyUUIDStr)
	assert.Error(t, err)

	// ทดสอบ UUID ที่มีรูปแบบเกือบถูกต้องแต่ไม่ใช่ UUID ที่ถูกต้อง
	almostValidUUIDStr := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a1z" // 'z' ไม่ใช่ hex digit
	_, err = StringToUUID(almostValidUUIDStr)
	assert.Error(t, err)

	// ทดสอบ UUID v4
	uuidV4 := uuid.New()
	uuidV4Str := uuidV4.String()

	result, err = StringToUUID(uuidV4Str)
	assert.NoError(t, err)
	assert.Equal(t, uuidV4, result)
}
