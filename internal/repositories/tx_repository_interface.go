package repositories

import (
	"context"

	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
)

// TxRepository defines the interface for transaction repositories
type TxRepositoryInterface interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
	AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error

	// Role operations
	CreateRole(ctx context.Context, role *models.Role) error
	UpdateRole(ctx context.Context, role *models.Role) error
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// Permission operations
	CreatePermission(ctx context.Context, permission *models.Permission) error
	UpdatePermission(ctx context.Context, permission *models.Permission) error
}

// TxRepository is the PostgreSQL implementation of TxRepositoryInterface
type TxRepository struct {
	tx interface{} // This is a sqlx.Tx for PostgreSQL
}

// MongoTxRepository is the MongoDB implementation of TxRepositoryInterface
type MongoTxRepository struct {
	db      interface{} // This is a *database.MongoDB
	session interface{} // This is a mongo.Session
	ctx     interface{} // This is a mongo.SessionContext
}
