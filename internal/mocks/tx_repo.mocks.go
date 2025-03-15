// MockTxRepository mocks the TxRepositoryInterface for transaction testing
package mocks

import (
	context "context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

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

// MockTransactionManager mocks a transaction manager
type MockTransactionManager struct {
	mock.Mock
}

// BeginTx begins a new transaction
func (m *MockTransactionManager) BeginTx(ctx context.Context) (transaction.Repository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(transaction.Repository), args.Error(1)
}

// CommitTx commits a transaction
func (m *MockTransactionManager) CommitTx(ctx context.Context, tx transaction.Repository) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// RollbackTx rolls back a transaction
func (m *MockTransactionManager) RollbackTx(ctx context.Context, tx transaction.Repository) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}
