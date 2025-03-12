package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomPassword(t *testing.T) {
	// ทดสอบความยาวของรหัสผ่านที่สร้าง
	lengths := []int{8, 12, 16, 20}

	for _, length := range lengths {
		password, err := GenerateRandomPassword(length)
		assert.NoError(t, err)
		assert.Len(t, password, length, "Password length should match requested length")
	}

	// ทดสอบว่าการสร้างรหัสผ่านแต่ละครั้งได้ผลลัพธ์ไม่ซ้ำกัน
	password1, err := GenerateRandomPassword(12)
	assert.NoError(t, err)

	password2, err := GenerateRandomPassword(12)
	assert.NoError(t, err)

	assert.NotEqual(t, password1, password2, "Generated passwords should be different")

	// ทดสอบกรณีความยาวน้อยกว่าที่กำหนด
	shortPassword, err := GenerateRandomPassword(4)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(shortPassword), 8, "Password length should be at least 8 characters")
}

func TestHashPasswordAndCheckPassword(t *testing.T) {
	// รหัสผ่านสำหรับทดสอบ
	plainPassword := "secureP@ssw0rd"

	// ทดสอบ hash รหัสผ่าน
	hashedPassword, err := HashPassword(plainPassword)
	assert.NoError(t, err)
	assert.NotEqual(t, plainPassword, hashedPassword, "Hashed password should be different from plain password")

	// ทดสอบตรวจสอบรหัสผ่านที่ถูกต้อง
	isValid := CheckPassword(plainPassword, hashedPassword)
	assert.True(t, isValid, "Password check should return true for correct password")

	// ทดสอบตรวจสอบรหัสผ่านที่ไม่ถูกต้อง
	isValid = CheckPassword("wrongPassword", hashedPassword)
	assert.False(t, isValid, "Password check should return false for incorrect password")

	// ทดสอบ hash รหัสผ่านเดียวกันสองครั้ง ควรได้ค่า hash ต่างกัน
	hashedPassword2, err := HashPassword(plainPassword)
	assert.NoError(t, err)
	assert.NotEqual(t, hashedPassword, hashedPassword2, "Same password should generate different hashes")

	// ทดสอบว่า hash ที่สองก็ยังสามารถตรวจสอบรหัสผ่านได้ถูกต้อง
	isValid = CheckPassword(plainPassword, hashedPassword2)
	assert.True(t, isValid, "Password check should return true for correct password with different hash")
}
