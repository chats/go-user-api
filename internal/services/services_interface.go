package services

import (
	"context"
	"time"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/google/uuid"
)

// AuthService defines the interface for authentication service operations
type AuthServiceInterface interface {
	Login(ctx context.Context, request models.LoginRequest) (*models.LoginResponse, error)
	VerifyToken(ctx context.Context, tokenString string) (*utils.JWTClaims, error)
	ChangePassword(ctx context.Context, userID string, currentPassword, newPassword string) error
	ResetPassword(ctx context.Context, userID string) (string, error)
	CheckPermission(ctx context.Context, userID string, resource, action string) (bool, error)
	GenerateToken(userID uuid.UUID, username string, roles []string) (string, time.Time, error)
}

// UserService defines the interface for user service operations
type UserServiceInterface interface {
	CreateUser(ctx context.Context, request models.UserCreateRequest) (*models.UserResponse, error)
	GetUserByID(ctx context.Context, id string) (*models.UserResponse, error)
	GetUserByUsername(ctx context.Context, username string) (*models.UserResponse, error)
	GetAllUsers(ctx context.Context, page, pageSize int) ([]models.UserResponse, int, error)
	UpdateUser(ctx context.Context, id string, request models.UserUpdateRequest) (*models.UserResponse, error)
	DeleteUser(ctx context.Context, id string) error
	GetUserPermissions(ctx context.Context, id string) ([]models.PermissionResponse, error)
	HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
}

// RoleService defines the interface for role service operations
type RoleServiceInterface interface {
	CreateRole(ctx context.Context, request models.RoleCreateRequest) (*models.RoleResponse, error)
	GetRoleByID(ctx context.Context, id string) (*models.RoleResponse, error)
	GetAllRoles(ctx context.Context) ([]models.RoleResponse, error)
	UpdateRole(ctx context.Context, id string, request models.RoleUpdateRequest) (*models.RoleResponse, error)
	DeleteRole(ctx context.Context, id string) error
	GetRolePermissions(ctx context.Context, id string) ([]models.PermissionResponse, error)
}

// PermissionService defines the interface for permission service operations
type PermissionServiceInterface interface {
	CreatePermission(ctx context.Context, request models.PermissionCreateRequest) (*models.PermissionResponse, error)
	GetPermissionByID(ctx context.Context, id string) (*models.PermissionResponse, error)
	GetAllPermissions(ctx context.Context) ([]models.PermissionResponse, error)
	GetPermissionsByResource(ctx context.Context, resource string) ([]models.PermissionResponse, error)
	UpdatePermission(ctx context.Context, id string, request models.PermissionUpdateRequest) (*models.PermissionResponse, error)
	DeletePermission(ctx context.Context, id string) error
}
