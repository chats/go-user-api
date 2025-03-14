package repositories

import (
	"context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user repository operations
type UserRepositoryInterface interface {
	ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetAll(ctx context.Context, limit, offset int) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
	AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
	HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	CountUsers(ctx context.Context) (int, error)
}

// RoleRepository defines the interface for role repository operations
type RoleRepositoryInterface interface {
	ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	GetByName(ctx context.Context, name string) (*models.Role, error)
	GetAll(ctx context.Context) ([]*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error)
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}

// PermissionRepository defines the interface for permission repository operations
type PermissionRepositoryInterface interface {
	ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error
	Create(ctx context.Context, permission *models.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	GetByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error)
	GetAll(ctx context.Context) ([]*models.Permission, error)
	GetByResource(ctx context.Context, resource string) ([]*models.Permission, error)
	Update(ctx context.Context, permission *models.Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
}
