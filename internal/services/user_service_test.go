package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chats/go-user-api/internal/mocks"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_CreateUser(t *testing.T) {
	t.Run("Successful user creation", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockTxRepo := new(mocks.MockTxRepository)

		// Mock behaviors
		mockUserRepo.On("GetByUsername", mock.Anything, "newuser").Return(nil, errors.New("user not found"))
		mockUserRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repositories.TxRepositoryInterface) error)
				fn(mockTxRepo)
			})
		mockTxRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

		// For role assignment
		roleID := uuid.New()
		mockTxRepo.On("AssignRolesToUser", mock.Anything, mock.AnythingOfType("uuid.UUID"),
			mock.AnythingOfType("[]uuid.UUID")).Return(nil)

		// For getting updated user
		mockUserRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
			Return(&models.User{
				ID:        uuid.New(),
				Username:  "newuser",
				Email:     "new@example.com",
				FirstName: "New",
				LastName:  "User",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Roles:     []models.Role{{ID: roleID, Name: "user"}},
			}, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare request
		request := models.UserCreateRequest{
			Username:  "newuser",
			Email:     "new@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
			RoleIDs:   []string{roleID.String()},
		}

		// Call service
		response, err := userService.CreateUser(context.Background(), request)

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "newuser", response.Username)
		assert.Equal(t, "new@example.com", response.Email)
		assert.Equal(t, "New", response.FirstName)
		assert.Equal(t, "User", response.LastName)
		assert.True(t, response.IsActive)
		assert.Len(t, response.Roles, 1)
		assert.Equal(t, "user", response.Roles[0].Name)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
		mockTxRepo.AssertExpectations(t)
	})

	t.Run("Username already exists", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Mock behaviors
		existingUser := &models.User{
			ID:       uuid.New(),
			Username: "existinguser",
			Email:    "existing@example.com",
		}
		mockUserRepo.On("GetByUsername", mock.Anything, "existinguser").Return(existingUser, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare request with existing username
		request := models.UserCreateRequest{
			Username:  "existinguser",
			Email:     "new@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
		}

		// Call service
		response, err := userService.CreateUser(context.Background(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "username already exists")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Transaction failure", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Mock behaviors
		mockUserRepo.On("GetByUsername", mock.Anything, "newuser").Return(nil, errors.New("user not found"))
		mockUserRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(errors.New("transaction failed"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare request
		request := models.UserCreateRequest{
			Username:  "newuser",
			Email:     "new@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
		}

		// Call service
		response, err := userService.CreateUser(context.Background(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "transaction failed")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	t.Run("User exists", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user
		userID := uuid.New()
		user := &models.User{
			ID:        userID,
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Roles:     []models.Role{{ID: uuid.New(), Name: "user"}},
		}

		// Mock behaviors
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		response, err := userService.GetUserByID(context.Background(), userID.String())

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, userID, response.ID)
		assert.Equal(t, user.Username, response.Username)
		assert.Equal(t, user.Email, response.Email)
		assert.Len(t, response.Roles, 1)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		response, err := userService.GetUserByID(context.Background(), userID.String())

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "user not found")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service with invalid UUID
		response, err := userService.GetUserByID(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify expectations - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	t.Run("Successful update", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)
		mockTxRepo := new(mocks.MockTxRepository)

		// Test user
		userID := uuid.New()
		user := &models.User{
			ID:        userID,
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Updated user that will be returned
		updatedUser := *user
		updatedUser.FirstName = "Updated"
		updatedUser.LastName = "Name"
		updatedUser.IsActive = false
		updatedUser.Roles = []models.Role{{ID: uuid.New(), Name: "editor"}}

		// Mock behaviors
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		mockUserRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repositories.TxRepositoryInterface) error)
				fn(mockTxRepo)
			})
		mockTxRepo.On("UpdateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

		// Mock get updated user
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(&updatedUser, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare update request
		isActive := false
		request := models.UserUpdateRequest{
			FirstName: "Updated",
			LastName:  "Name",
			IsActive:  &isActive,
		}

		// Call service
		response, err := userService.UpdateUser(context.Background(), userID.String(), request)

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, userID, response.ID)
		assert.Equal(t, "testuser", response.Username) // Unchanged
		assert.Equal(t, "Updated", response.FirstName) // Changed
		assert.Equal(t, "Name", response.LastName)     // Changed
		assert.False(t, response.IsActive)             // Changed
		assert.Len(t, response.Roles, 1)
		assert.Equal(t, "editor", response.Roles[0].Name)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
		mockTxRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare update request
		request := models.UserUpdateRequest{
			FirstName: "Updated",
			LastName:  "Name",
		}

		// Call service
		response, err := userService.UpdateUser(context.Background(), userID.String(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "user not found")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Username already exists", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user
		userID := uuid.New()
		user := &models.User{
			ID:        userID,
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Existing user with the username we want to change to
		existingUser := &models.User{
			ID:       uuid.New(), // Different ID
			Username: "existinguser",
			Email:    "existing@example.com",
		}

		// Mock behaviors
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
		mockUserRepo.On("GetByUsername", mock.Anything, "existinguser").Return(existingUser, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Prepare update request with existing username
		request := models.UserUpdateRequest{
			Username: "existinguser",
		}

		// Call service
		response, err := userService.UpdateUser(context.Background(), userID.String(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "username already exists")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Run("Successful deletion", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("Delete", mock.Anything, userID).Return(nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		err := userService.DeleteUser(context.Background(), userID.String())

		// Assert results
		assert.NoError(t, err)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("Delete", mock.Anything, userID).Return(errors.New("user not found"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		err := userService.DeleteUser(context.Background(), userID.String())

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service with invalid UUID
		err := userService.DeleteUser(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify expectations - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUserPermissions(t *testing.T) {
	t.Run("Get permissions successfully", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Test permissions
		permissions := []models.Permission{
			{ID: uuid.New(), Name: "user:read", Resource: "user", Action: "read"},
			{ID: uuid.New(), Name: "user:write", Resource: "user", Action: "write"},
		}

		// Mock behaviors
		mockUserRepo.On("GetUserPermissions", mock.Anything, userID).Return(permissions, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		response, err := userService.GetUserPermissions(context.Background(), userID.String())

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response, 2)
		assert.Equal(t, "user:read", response[0].Name)
		assert.Equal(t, "user:write", response[1].Name)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("GetUserPermissions", mock.Anything, userID).Return(nil, errors.New("user not found"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		response, err := userService.GetUserPermissions(context.Background(), userID.String())

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "user not found")

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service with invalid UUID
		response, err := userService.GetUserPermissions(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify expectations - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_HasPermission(t *testing.T) {
	t.Run("User has permission", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "read").Return(true, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		hasPermission, err := userService.HasPermission(context.Background(), userID.String(), "user", "read")

		// Assert results
		assert.NoError(t, err)
		assert.True(t, hasPermission)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User does not have permission", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "delete").Return(false, nil)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		hasPermission, err := userService.HasPermission(context.Background(), userID.String(), "user", "delete")

		// Assert results
		assert.NoError(t, err)
		assert.False(t, hasPermission)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Error checking permission", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Test user ID
		userID := uuid.New()

		// Mock behaviors
		mockUserRepo.On("HasPermission", mock.Anything, userID, "user", "write").
			Return(false, errors.New("database error"))

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service
		hasPermission, err := userService.HasPermission(context.Background(), userID.String(), "user", "write")

		// Assert results
		assert.Error(t, err)
		assert.False(t, hasPermission)

		// Verify expectations
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockUserRepo := new(mocks.MockUserRepository)
		mockRoleRepo := new(mocks.MockRoleRepository)

		// Create service
		userService := services.NewUserService(mockUserRepo, mockRoleRepo)

		// Call service with invalid UUID
		hasPermission, err := userService.HasPermission(context.Background(), "not-a-uuid", "user", "read")

		// Assert results
		assert.Error(t, err)
		assert.False(t, hasPermission)
		assert.Contains(t, err.Error(), "invalid user ID")

		// Verify expectations - no methods should be called
		mockUserRepo.AssertExpectations(t)
	})
}
