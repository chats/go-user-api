package mocks

import (
	context "context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

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
