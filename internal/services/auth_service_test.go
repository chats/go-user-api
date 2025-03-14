package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/mocks"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Login(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// Test user
	userID := uuid.New()
	password := "test-password"
	hashedPassword, err := utils.HashPassword(password)
	require.NoError(t, err)

	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  hashedPassword,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles: []models.Role{
			{ID: uuid.New(), Name: "admin"},
			{ID: uuid.New(), Name: "user"},
		},
	}

	t.Run("Successful login", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Prepare login request
		loginRequest := models.LoginRequest{
			Username: "testuser",
			Password: password,
		}

		// Call service
		response, err := authService.Login(context.Background(), loginRequest)

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "bearer", response.TokenType)
		assert.Greater(t, response.ExpiresIn, 0)
		assert.Equal(t, user.ID, response.User.ID)
		assert.Equal(t, user.Username, response.User.Username)

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByUsername", mock.Anything, "nonexistent").
			Return(nil, errors.New("user not found"))

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Prepare login request
		loginRequest := models.LoginRequest{
			Username: "nonexistent",
			Password: password,
		}

		// Call service
		response, err := authService.Login(context.Background(), loginRequest)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid username or password")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Inactive user", func(t *testing.T) {
		// Setup inactive user
		inactiveUser := *user
		inactiveUser.IsActive = false

		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByUsername", mock.Anything, "inactive").Return(&inactiveUser, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Prepare login request
		loginRequest := models.LoginRequest{
			Username: "inactive",
			Password: password,
		}

		// Call service
		response, err := authService.Login(context.Background(), loginRequest)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "user account is inactive")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid password", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Prepare login request with wrong password
		loginRequest := models.LoginRequest{
			Username: "testuser",
			Password: "wrong-password",
		}

		// Call service
		response, err := authService.Login(context.Background(), loginRequest)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid username or password")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// Test user
	userID := uuid.New()
	currentPassword := "current-password"
	hashedPassword, err := utils.HashPassword(currentPassword)
	require.NoError(t, err)

	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  hashedPassword,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("Successful password change", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		mockUserRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		err := authService.ChangePassword(context.Background(), userID.String(), currentPassword, "new-password")

		// Assert results
		assert.NoError(t, err)

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found"))

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		err := authService.ChangePassword(context.Background(), userID.String(), currentPassword, "new-password")

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Incorrect current password", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service with wrong current password
		err := authService.ChangePassword(context.Background(), userID.String(), "wrong-password", "new-password")

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current password is incorrect")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service with invalid user ID
		err := authService.ChangePassword(context.Background(), "not-a-uuid", currentPassword, "new-password")

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify mock - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_CheckPermission(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// Test user ID
	userID := uuid.New()

	t.Run("User has permission", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "read").Return(true, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		hasPermission, err := authService.CheckPermission(context.Background(), userID.String(), "user", "read")

		// Assert results
		assert.NoError(t, err)
		assert.True(t, hasPermission)

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User does not have permission", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "delete").Return(false, nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		hasPermission, err := authService.CheckPermission(context.Background(), userID.String(), "user", "delete")

		// Assert results
		assert.NoError(t, err)
		assert.False(t, hasPermission)

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Error checking permission", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "write").
			Return(false, errors.New("database error"))

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		hasPermission, err := authService.CheckPermission(context.Background(), userID.String(), "user", "write")

		// Assert results
		assert.Error(t, err)
		assert.False(t, hasPermission)
		assert.Contains(t, err.Error(), "failed to check permission")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service with invalid user ID
		hasPermission, err := authService.CheckPermission(context.Background(), "not-a-uuid", "user", "read")

		// Assert results
		assert.Error(t, err)
		assert.False(t, hasPermission)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify mock - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_ResetPassword(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// Test user
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("Successful password reset", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		mockUserRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		newPassword, err := authService.ResetPassword(context.Background(), userID.String())

		// Assert results
		assert.NoError(t, err)
		assert.NotEmpty(t, newPassword)
		assert.GreaterOrEqual(t, len(newPassword), 8) // Minimum password length

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found"))

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		newPassword, err := authService.ResetPassword(context.Background(), userID.String())

		// Assert results
		assert.Error(t, err)
		assert.Empty(t, newPassword)
		assert.Contains(t, err.Error(), "user not found")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid user ID format", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service with invalid user ID
		newPassword, err := authService.ResetPassword(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Empty(t, newPassword)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify mock - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Error updating password", func(t *testing.T) {
		// Setup mock repository
		mockUserRepo := new(mocks.MockUserRepository)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		mockUserRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).
			Return(errors.New("database error"))

		// Create service
		authService := services.NewAuthService(mockUserRepo, cfg)

		// Call service
		newPassword, err := authService.ResetPassword(context.Background(), userID.String())

		// Assert results
		assert.Error(t, err)
		assert.Empty(t, newPassword)
		assert.Contains(t, err.Error(), "failed to update password")

		// Verify mock
		mockUserRepo.AssertExpectations(t)
	})
}
