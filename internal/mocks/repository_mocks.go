package mocks

import (
	"context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository mocks the UserRepositoryInterface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.User, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockUserRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Permission), args.Error(1)
}

func (m *MockUserRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	args := m.Called(ctx, userID, roleIDs)
	return args.Error(0)
}

func (m *MockUserRepository) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) CountUsers(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) ExecuteTx(ctx context.Context, fn func(transaction.Repository) error) error {
	args := m.Called(ctx, fn)

	// If no error and a function was provided, execute it with a mock transaction repository
	if args.Error(0) == nil && fn != nil {
		mockTxRepo := new(MockTxRepository)
		return fn(mockTxRepo)
	}

	return args.Error(0)
}

// MockRoleRepository mocks the RoleRepositoryInterface
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepository) GetAll(ctx context.Context) ([]*models.Role, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]models.Permission), args.Error(1)
}

func (m *MockRoleRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

func (m *MockRoleRepository) ExecuteTx(ctx context.Context, fn func(transaction.Repository) error) error {
	args := m.Called(ctx, fn)

	// If no error and a function was provided, execute it with a mock transaction repository
	if args.Error(0) == nil && fn != nil {
		mockTxRepo := new(MockTxRepository)
		return fn(mockTxRepo)
	}

	return args.Error(0)
}

// MockPermissionRepository mocks the PermissionRepositoryInterface
type MockPermissionRepository struct {
	mock.Mock
}

func (m *MockPermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Permission), args.Error(1)
}

func (m *MockPermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error) {
	args := m.Called(ctx, resource, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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

func (m *MockPermissionRepository) ExecuteTx(ctx context.Context, fn func(transaction.Repository) error) error {
	args := m.Called(ctx, fn)

	// If no error and a function was provided, execute it with a mock transaction repository
	if args.Error(0) == nil && fn != nil {
		mockTxRepo := new(MockTxRepository)
		return fn(mockTxRepo)
	}

	return args.Error(0)
}

// MockTxRepository mocks the TxRepositoryInterface for transaction testing
type MockTxRepository struct {
	mock.Mock
}

func (m *MockTxRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockTxRepository) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockTxRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Error(0)
}

func (m *MockTxRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	args := m.Called(ctx, userID, roleIDs)
	return args.Error(0)
}

func (m *MockTxRepository) CreateRole(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockTxRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockTxRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionIDs)
	return args.Error(0)
}

func (m *MockTxRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockTxRepository) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}
