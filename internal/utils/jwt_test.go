package utils

import (
	"testing"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseJWT(t *testing.T) {
	// สร้าง config สำหรับการทดสอบ
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// สร้างข้อมูลสำหรับทดสอบ
	userID := uuid.New()
	username := "testuser"
	roles := []string{"admin", "editor"}

	// ทดสอบ Generate JWT
	tokenString, expirationTime, err := GenerateJWT(userID, username, roles, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)
	assert.True(t, expirationTime.After(time.Now()))
	assert.True(t, expirationTime.Before(time.Now().Add(time.Hour+time.Minute)))

	// ทดสอบ Parse JWT
	claims, err := ParseJWT(tokenString, cfg)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, roles, claims.Roles)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid bearer token",
			authHeader:  "Bearer tokenvalue123",
			expected:    "tokenvalue123",
			expectError: false,
		},
		{
			name:        "Missing bearer prefix",
			authHeader:  "tokenvalue123",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty auth header",
			authHeader:  "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Bearer prefix only",
			authHeader:  "Bearer ",
			expected:    "",
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExtractBearerToken(tc.authHeader)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestParseJWTWithInvalidToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	// ทดสอบกับ token ที่ไม่ถูกต้อง
	_, err := ParseJWT("invalid.token.string", cfg)
	assert.Error(t, err)

	// ทดสอบกับ token ที่หมดอายุ
	// สร้าง token ที่หมดอายุแล้ว (กำหนด JWTExpireMinute เป็น -60 นาที)
	expiredCfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: -60, // หมดอายุไปแล้ว 1 ชั่วโมง
	}

	userID := uuid.New()
	expiredTokenString, _, err := GenerateJWT(userID, "expireduser", []string{"user"}, expiredCfg)
	assert.NoError(t, err)

	_, err = ParseJWT(expiredTokenString, cfg)
	assert.Error(t, err)
}
