package mocks

import (
	"context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPermissionRepository mocks the PermissionRepositoryInterface
type MockPermissionRepository struct {
	mock.Mock
}

func (m *MockPermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockPermissionRepository) CreateRole(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockPermissionRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockPermissionRepository) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockPermissionRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	args := m.Called(ctx, userID, password)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error) {
	args := m.Called(ctx, resource, action)
	return args.Get(0).(*models.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetAll(ctx context.Context) ([]*models.Permission, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetByResource(ctx context.Context, resource string) ([]*models.Permission, error) {
	args := m.Called(ctx, resource)
	return args.Get(0).([]*models.Permission), args.Error(1)
}

func (m *MockPermissionRepository) Update(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPermissionRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

func (m *MockPermissionRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	args := m.Called(ctx, userID, roleIDs)
	return args.Error(0)
}

func (m *MockPermissionRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) ExecuteTx(ctx context.Context, fn func(transaction.Repository) error) error {
	args := m.Called(ctx, fn)

	// If no error and a function was provided, execute it with a mock transaction repository
	if args.Error(0) == nil && fn != nil {
		mockTxRepo := new(MockTxRepository)
		return fn(mockTxRepo)
	}

	return args.Error(0)
}
