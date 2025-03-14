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

func TestPermissionService_CreatePermission(t *testing.T) {
	t.Run("Successful permission creation", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)
		mockTxRepo := new(mocks.MockTxRepository)

		// Mock behaviors
		mockPermRepo.On("GetByResourceAction", mock.Anything, "report", "read").
			Return(nil, errors.New("permission not found"))
		mockPermRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repositories.TxRepositoryInterface) error)
				fn(mockTxRepo)
			})
		mockTxRepo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*models.Permission")).
			Run(func(args mock.Arguments) {
				perm := args.Get(1).(*models.Permission)
				perm.ID = uuid.New() // Simulate ID assignment
			}).
			Return(nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare request
		request := models.PermissionCreateRequest{
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
		}

		// Call service
		response, err := permService.CreatePermission(context.Background(), request)

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "report:read", response.Name)
		assert.Equal(t, "Permission to read reports", response.Description)
		assert.Equal(t, "report", response.Resource)
		assert.Equal(t, "read", response.Action)

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
		mockTxRepo.AssertExpectations(t)
	})

	t.Run("Permission already exists", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Mock behaviors
		existingPerm := &models.Permission{
			ID:          uuid.New(),
			Name:        "report:read",
			Description: "Existing permission",
			Resource:    "report",
			Action:      "read",
		}
		mockPermRepo.On("GetByResourceAction", mock.Anything, "report", "read").Return(existingPerm, nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare request with existing resource+action
		request := models.PermissionCreateRequest{
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
		}

		// Call service
		response, err := permService.CreatePermission(context.Background(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission already exists for this resource and action")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Transaction failure", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Mock behaviors
		mockPermRepo.On("GetByResourceAction", mock.Anything, "report", "read").
			Return(nil, errors.New("permission not found"))
		mockPermRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(errors.New("transaction failed"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare request
		request := models.PermissionCreateRequest{
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
		}

		// Call service
		response, err := permService.CreatePermission(context.Background(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "transaction failed")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})
}

func TestPermissionService_GetPermissionByID(t *testing.T) {
	t.Run("Permission exists", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission
		permID := uuid.New()
		perm := &models.Permission{
			ID:          permID,
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Mock behaviors
		mockPermRepo.On("GetByID", mock.Anything, permID).Return(perm, nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetPermissionByID(context.Background(), permID.String())

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, permID, response.ID)
		assert.Equal(t, perm.Name, response.Name)
		assert.Equal(t, perm.Resource, response.Resource)
		assert.Equal(t, perm.Action, response.Action)

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Permission not found", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission ID
		permID := uuid.New()

		// Mock behaviors
		mockPermRepo.On("GetByID", mock.Anything, permID).Return(nil, errors.New("permission not found"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetPermissionByID(context.Background(), permID.String())

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission not found")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service with invalid UUID
		response, err := permService.GetPermissionByID(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid permission ID")

		// Verify expectations - no methods should be called
		mockPermRepo.AssertExpectations(t)
	})
}

func TestPermissionService_GetAllPermissions(t *testing.T) {
	t.Run("Get all permissions successfully", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permissions
		perms := []*models.Permission{
			{
				ID:          uuid.New(),
				Name:        "user:read",
				Description: "Read user",
				Resource:    "user",
				Action:      "read",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New(),
				Name:        "user:write",
				Description: "Write user",
				Resource:    "user",
				Action:      "write",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		// Mock behaviors
		mockPermRepo.On("GetAll", mock.Anything).Return(perms, nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetAllPermissions(context.Background())

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response, 2)
		assert.Equal(t, "user:read", response[0].Name)
		assert.Equal(t, "user:write", response[1].Name)

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Error getting permissions", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Mock behaviors
		mockPermRepo.On("GetAll", mock.Anything).Return(nil, errors.New("database error"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetAllPermissions(context.Background())

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "database error")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})
}

func TestPermissionService_GetPermissionsByResource(t *testing.T) {
	t.Run("Get permissions by resource successfully", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permissions
		perms := []*models.Permission{
			{
				ID:          uuid.New(),
				Name:        "user:read",
				Description: "Read user",
				Resource:    "user",
				Action:      "read",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New(),
				Name:        "user:write",
				Description: "Write user",
				Resource:    "user",
				Action:      "write",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		// Mock behaviors
		mockPermRepo.On("GetByResource", mock.Anything, "user").Return(perms, nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetPermissionsByResource(context.Background(), "user")

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response, 2)
		assert.Equal(t, "user:read", response[0].Name)
		assert.Equal(t, "user:write", response[1].Name)

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Error getting permissions", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Mock behaviors
		mockPermRepo.On("GetByResource", mock.Anything, "user").Return(nil, errors.New("database error"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		response, err := permService.GetPermissionsByResource(context.Background(), "user")

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "database error")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})
}

func TestPermissionService_UpdatePermission(t *testing.T) {
	t.Run("Successful update", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)
		mockTxRepo := new(mocks.MockTxRepository)

		// Test permission
		permID := uuid.New()
		perm := &models.Permission{
			ID:          permID,
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Mock behaviors
		mockPermRepo.On("GetByID", mock.Anything, permID).Return(perm, nil)
		mockPermRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repositories.TxRepositoryInterface) error)
				fn(mockTxRepo)
			})
		mockTxRepo.On("UpdatePermission", mock.Anything, mock.AnythingOfType("*models.Permission")).Return(nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare update request
		request := models.PermissionUpdateRequest{
			Name:        "report:view",
			Description: "Updated description",
		}

		// Call service
		response, err := permService.UpdatePermission(context.Background(), permID.String(), request)

		// Assert results
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, permID, response.ID)
		assert.Equal(t, "report:view", response.Name)
		assert.Equal(t, "Updated description", response.Description)
		assert.Equal(t, "report", response.Resource) // Unchanged
		assert.Equal(t, "read", response.Action)     // Unchanged

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
		mockTxRepo.AssertExpectations(t)
	})

	t.Run("Permission not found", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission ID
		permID := uuid.New()

		// Mock behaviors
		mockPermRepo.On("GetByID", mock.Anything, permID).Return(nil, errors.New("permission not found"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare update request
		request := models.PermissionUpdateRequest{
			Description: "Updated description",
		}

		// Call service
		response, err := permService.UpdatePermission(context.Background(), permID.String(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission not found")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Resource+Action already exists", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission
		permID := uuid.New()
		perm := &models.Permission{
			ID:          permID,
			Name:        "report:read",
			Description: "Permission to read reports",
			Resource:    "report",
			Action:      "read",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Existing permission with different ID but same resource+action
		existingPerm := &models.Permission{
			ID:          uuid.New(), // Different ID
			Name:        "report:write",
			Description: "Permission to write reports",
			Resource:    "report",
			Action:      "write",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Mock behaviors
		mockPermRepo.On("GetByID", mock.Anything, permID).Return(perm, nil)
		mockPermRepo.On("GetByResourceAction", mock.Anything, "report", "write").Return(existingPerm, nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Prepare update request changing action to one that already exists
		request := models.PermissionUpdateRequest{
			Action: "write",
		}

		// Call service
		response, err := permService.UpdatePermission(context.Background(), permID.String(), request)

		// Assert results
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission already exists for this resource and action")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})
}

func TestPermissionService_DeletePermission(t *testing.T) {
	t.Run("Successful deletion", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission ID
		permID := uuid.New()

		// Mock behaviors
		mockPermRepo.On("Delete", mock.Anything, permID).Return(nil)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		err := permService.DeletePermission(context.Background(), permID.String())

		// Assert results
		assert.NoError(t, err)

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Permission not found", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Test permission ID
		permID := uuid.New()

		// Mock behaviors
		mockPermRepo.On("Delete", mock.Anything, permID).Return(errors.New("permission not found"))

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service
		err := permService.DeletePermission(context.Background(), permID.String())

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")

		// Verify expectations
		mockPermRepo.AssertExpectations(t)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Setup mocks
		mockPermRepo := new(mocks.MockPermissionRepository)

		// Create service
		permService := services.NewPermissionService(mockPermRepo)

		// Call service with invalid UUID
		err := permService.DeletePermission(context.Background(), "not-a-uuid")

		// Assert results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid permission ID")

		// Verify expectations - no methods should be called
		mockPermRepo.AssertExpectations(t)
	})
}
