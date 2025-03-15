package transaction

import (
	"context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
)

// UserOperations defines user-related transaction operations
type UserOperations interface {
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
	AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
}

// RoleOperations defines role-related transaction operations
type RoleOperations interface {
	CreateRole(ctx context.Context, role *models.Role) error
	UpdateRole(ctx context.Context, role *models.Role) error
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}

// PermissionOperations defines permission-related transaction operations
type PermissionOperations interface {
	CreatePermission(ctx context.Context, permission *models.Permission) error
	UpdatePermission(ctx context.Context, permission *models.Permission) error
}

// Repository combines all transaction operations
type Repository interface {
	UserOperations
	RoleOperations
	PermissionOperations
}
