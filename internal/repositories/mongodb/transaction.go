package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/database"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoTx wraps MongoDB session for transaction management
type MongoTx struct {
	session mongo.Session
	ctx     mongo.SessionContext
}

// Commit implements the Executor interface
func (tx *MongoTx) Commit() error {
	return tx.ctx.CommitTransaction(tx.ctx)
}

// Rollback implements the Executor interface
func (tx *MongoTx) Rollback() error {
	return tx.ctx.AbortTransaction(tx.ctx)
}

// TxRepository implements transaction.Repository for MongoDB
type TxRepository struct {
	db  *database.MongoDB
	ctx mongo.SessionContext
}

// Ensure TxRepository implements transaction.Repository
var _ transaction.Repository = (*TxRepository)(nil)

// usersCollection returns the MongoDB collection for users
func (r *TxRepository) usersCollection() *mongo.Collection {
	return r.db.GetCollection("users")
}

// userRolesCollection returns the MongoDB collection for user-roles relationship
func (r *TxRepository) userRolesCollection() *mongo.Collection {
	return r.db.GetCollection("user_roles")
}

// rolesCollection returns the MongoDB collection for roles
func (r *TxRepository) rolesCollection() *mongo.Collection {
	return r.db.GetCollection("roles")
}

// permissionsCollection returns the MongoDB collection for permissions
func (r *TxRepository) permissionsCollection() *mongo.Collection {
	return r.db.GetCollection("permissions")
}

// rolePermissionsCollection returns the MongoDB collection for role-permissions relationship
func (r *TxRepository) rolePermissionsCollection() *mongo.Collection {
	return r.db.GetCollection("role_permissions")
}

// NewTransactionManager creates a new transaction manager for MongoDB
func NewTransactionManager(db *database.MongoDB) transaction.Manager[transaction.Repository] {
	beginTx := func(ctx context.Context) (*MongoTx, error) {
		session, err := db.Client.StartSession()
		if err != nil {
			return nil, fmt.Errorf("failed to start MongoDB session: %w", err)
		}

		sessCtx, err := session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
			return sc, nil
		})

		if err != nil {
			session.EndSession(ctx)
			return nil, fmt.Errorf("failed to create transaction context: %w", err)
		}

		return &MongoTx{
			session: session,
			ctx:     sessCtx.(mongo.SessionContext),
		}, nil
	}

	createRepo := func(tx *MongoTx) transaction.Repository {
		return &TxRepository{
			db:  db,
			ctx: tx.ctx,
		}
	}

	return transaction.NewGenericManager(beginTx, createRepo)
}

// CreateUser creates a new user within a transaction
func (r *TxRepository) CreateUser(ctx context.Context, user *models.User) error {
	// Generate UUID if not provided
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	// Set timestamps if not provided
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// Insert into database
	_, err := r.usersCollection().InsertOne(r.ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user in MongoDB transaction: %w", err)
	}

	return nil
}

// UpdateUser updates a user within a transaction
func (r *TxRepository) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	filter := bson.M{"_id": user.ID}
	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"is_active":  user.IsActive,
			"updated_at": user.UpdatedAt,
		},
	}

	result, err := r.usersCollection().UpdateOne(r.ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user in MongoDB transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserPassword updates a user password within a transaction
func (r *TxRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		},
	}

	result, err := r.usersCollection().UpdateOne(r.ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update password in MongoDB transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// AssignRolesToUser assigns roles to a user within a transaction
func (r *TxRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Remove existing roles
	_, err := r.userRolesCollection().DeleteMany(r.ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to remove existing roles in MongoDB transaction: %w", err)
	}

	// Assign new roles
	if len(roleIDs) > 0 {
		userRoles := make([]interface{}, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			userRoles = append(userRoles, bson.M{
				"user_id":    userID,
				"role_id":    roleID,
				"created_at": time.Now(),
			})
		}

		_, err = r.userRolesCollection().InsertMany(r.ctx, userRoles)
		if err != nil {
			return fmt.Errorf("failed to assign roles in MongoDB transaction: %w", err)
		}
	}

	return nil
}

// CreateRole creates a new role within a transaction
func (r *TxRepository) CreateRole(ctx context.Context, role *models.Role) error {
	// Generate UUID if not provided
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}

	// Set timestamps if not provided
	now := time.Now()
	if role.CreatedAt.IsZero() {
		role.CreatedAt = now
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = now
	}

	// Insert into database
	_, err := r.rolesCollection().InsertOne(r.ctx, role)
	if err != nil {
		return fmt.Errorf("failed to create role in MongoDB transaction: %w", err)
	}

	return nil
}

// UpdateRole updates a role within a transaction
func (r *TxRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	role.UpdatedAt = time.Now()

	filter := bson.M{"_id": role.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        role.Name,
			"description": role.Description,
			"updated_at":  role.UpdatedAt,
		},
	}

	result, err := r.rolesCollection().UpdateOne(r.ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update role in MongoDB transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

// AssignPermissionsToRole assigns permissions to a role within a transaction
func (r *TxRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Remove existing permissions
	_, err := r.rolePermissionsCollection().DeleteMany(r.ctx, bson.M{"role_id": roleID})
	if err != nil {
		return fmt.Errorf("failed to remove existing permissions in MongoDB transaction: %w", err)
	}

	// Assign new permissions
	if len(permissionIDs) > 0 {
		rolePermissions := make([]interface{}, 0, len(permissionIDs))
		for _, permissionID := range permissionIDs {
			rolePermissions = append(rolePermissions, bson.M{
				"role_id":       roleID,
				"permission_id": permissionID,
				"created_at":    time.Now(),
			})
		}

		_, err = r.rolePermissionsCollection().InsertMany(r.ctx, rolePermissions)
		if err != nil {
			return fmt.Errorf("failed to assign permissions in MongoDB transaction: %w", err)
		}
	}

	return nil
}

// CreatePermission creates a new permission within a transaction
func (r *TxRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	// Generate UUID if not provided
	if permission.ID == uuid.Nil {
		permission.ID = uuid.New()
	}

	// Set timestamps if not provided
	now := time.Now()
	if permission.CreatedAt.IsZero() {
		permission.CreatedAt = now
	}
	if permission.UpdatedAt.IsZero() {
		permission.UpdatedAt = now
	}

	// Insert into database
	_, err := r.permissionsCollection().InsertOne(r.ctx, permission)
	if err != nil {
		return fmt.Errorf("failed to create permission in MongoDB transaction: %w", err)
	}

	return nil
}

// UpdatePermission updates a permission within a transaction
func (r *TxRepository) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	permission.UpdatedAt = time.Now()

	filter := bson.M{"_id": permission.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        permission.Name,
			"description": permission.Description,
			"resource":    permission.Resource,
			"action":      permission.Action,
			"updated_at":  permission.UpdatedAt,
		},
	}

	result, err := r.permissionsCollection().UpdateOne(r.ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update permission in MongoDB transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}
