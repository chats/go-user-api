package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chats/go-user-api/internal/mocks"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/chats/go-user-api/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPermissionService_CreatePermission(t *testing.T) {
	mockPermissionRepo := new(mocks.MockPermissionRepository)
	mockTxManager := new(mocks.Manager[transaction.Repository])

	permissionService := services.NewPermissionService(mockPermissionRepo, mockTxManager)

	request := models.PermissionCreateRequest{
		Name:        "test-permission",
		Description: "test-description",
		Resource:    "test-resource",
		Action:      "test-action",
	}

	t.Run("Successful creation", func(t *testing.T) {
		mockPermissionRepo.On("GetByResourceAction", mock.Anything, request.Resource, request.Action).Return(nil, nil)
		mockTxManager.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(transaction.Repository) error")).Return(nil).Run(func(args mock.Arguments) {
			txFunc := args.Get(1).(func(transaction.Repository) error)
			txFunc(mockPermissionRepo)
		})
		mockPermissionRepo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*models.Permission")).Return(nil)

		response, err := permissionService.CreatePermission(context.Background(), request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, request.Name, response.Name)
		mockPermissionRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Permission already exists", func(t *testing.T) {
		existingPermission := &models.Permission{}
		mockPermissionRepo.On("GetByResourceAction", mock.Anything, request.Resource, request.Action).Return(existingPermission, nil)

		response, err := permissionService.CreatePermission(context.Background(), request)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission already exists for this resource and action")
		mockPermissionRepo.AssertExpectations(t)
		mockTxManager.AssertNotCalled(t, "ExecuteTx", mock.Anything, mock.Anything)
	})
}

func TestPermissionService_UpdatePermission(t *testing.T) {
	mockPermissionRepo := new(mocks.MockPermissionRepository)
	mockTxManager := new(mocks.Manager[transaction.Repository])

	permissionService := services.NewPermissionService(mockPermissionRepo, mockTxManager)

	id := uuid.New().String()
	request := models.PermissionUpdateRequest{
		Name:        "updated-name",
		Description: "updated-description",
		Resource:    "updated-resource",
		Action:      "updated-action",
	}

	t.Run("Successful update", func(t *testing.T) {
		permission := &models.Permission{
			ID:          uuid.MustParse(id),
			Name:        "test-permission",
			Description: "test-description",
			Resource:    "test-resource",
			Action:      "test-action",
		}
		mockPermissionRepo.On("GetByID", mock.Anything, uuid.MustParse(id)).Return(permission, nil)
		mockPermissionRepo.On("GetByResourceAction", mock.Anything, request.Resource, request.Action).Return(nil, nil)
		mockTxManager.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(transaction.Repository) error")).Return(nil).Run(func(args mock.Arguments) {
			txFunc := args.Get(1).(func(transaction.Repository) error)
			txFunc(mockPermissionRepo)
		})
		mockPermissionRepo.On("UpdatePermission", mock.Anything, mock.AnythingOfType("*models.Permission")).Return(nil)

		response, err := permissionService.UpdatePermission(context.Background(), id, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, request.Name, response.Name)
		mockPermissionRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Permission not found", func(t *testing.T) {
		mockPermissionRepo.On("GetByID", mock.Anything, uuid.MustParse(id)).Return(nil, errors.New("permission not found"))

		response, err := permissionService.UpdatePermission(context.Background(), id, request)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "permission not found")
		mockPermissionRepo.AssertExpectations(t)
		mockTxManager.AssertNotCalled(t, "ExecuteTx", mock.Anything, mock.Anything)
	})
}

func TestPermissionService_DeletePermission(t *testing.T) {
	mockPermissionRepo := new(mocks.MockPermissionRepository)
	mockTxManager := new(mocks.Manager[transaction.Repository])

	permissionService := services.NewPermissionService(mockPermissionRepo, mockTxManager)

	id := uuid.New().String()

	t.Run("Successful deletion", func(t *testing.T) {
		mockPermissionRepo.On("Delete", mock.Anything, uuid.MustParse(id)).Return(nil)

		err := permissionService.DeletePermission(context.Background(), id)

		assert.NoError(t, err)
		mockPermissionRepo.AssertExpectations(t)
	})

	t.Run("Invalid permission ID", func(t *testing.T) {
		invalidID := "invalid-uuid"

		err := permissionService.DeletePermission(context.Background(), invalidID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid permission ID")
		mockPermissionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}
