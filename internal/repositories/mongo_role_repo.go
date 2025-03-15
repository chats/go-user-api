package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/cache"
	"github.com/chats/go-user-api/internal/database"
	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRoleRepository handles database operations for roles with MongoDB
type MongoRoleRepository struct {
	db    *database.MongoDB
	cache *cache.RedisClient
}

// NewMongoRoleRepository creates a new MongoDB role repository
func NewMongoRoleRepository(db *database.MongoDB, cache *cache.RedisClient) *MongoRoleRepository {
	return &MongoRoleRepository{
		db:    db,
		cache: cache,
	}
}

// rolesCollection returns the MongoDB collection for roles
func (r *MongoRoleRepository) rolesCollection() *mongo.Collection {
	return r.db.GetCollection("roles")
}

// rolePermissionsCollection returns the MongoDB collection for role-permissions relationship
func (r *MongoRoleRepository) rolePermissionsCollection() *mongo.Collection {
	return r.db.GetCollection("role_permissions")
}

// permissionsCollection returns the MongoDB collection for permissions
func (r *MongoRoleRepository) permissionsCollection() *mongo.Collection {
	return r.db.GetCollection("permissions")
}

// Create creates a new role in the database
func (r *MongoRoleRepository) Create(ctx context.Context, role *models.Role) error {
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
	_, err := r.rolesCollection().InsertOne(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to create role in MongoDB: %w", err)
	}

	// Clear cache
	r.invalidateRoleCache()

	return nil
}

// GetByID retrieves a role by ID
func (r *MongoRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	cacheKey := fmt.Sprintf("role:%s", id.String())

	// Try to get from cache first
	var role models.Role
	found, err := r.cache.Get(cacheKey, &role)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get role from cache")
	}

	if found {
		// Get permissions for the role
		permissions, err := r.GetRolePermissions(ctx, id)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
		return &role, nil
	}

	// If not in cache, get from database
	filter := bson.M{"_id": id}

	result := r.rolesCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode role from MongoDB: %w", err)
	}

	// Get permissions for the role
	permissions, err := r.GetRolePermissions(ctx, id)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions

	// Cache the role
	if err := r.cache.Set(cacheKey, role); err != nil {
		log.Debug().Err(err).Msg("Failed to cache role")
	}

	return &role, nil
}

// GetByName retrieves a role by name
func (r *MongoRoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	cacheKey := fmt.Sprintf("role:name:%s", name)

	// Try to get from cache first
	var role models.Role
	found, err := r.cache.Get(cacheKey, &role)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get role from cache")
	}

	if found {
		// Get permissions for the role
		permissions, err := r.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
		return &role, nil
	}

	// If not in cache, get from database
	filter := bson.M{"name": name}

	result := r.rolesCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode role from MongoDB: %w", err)
	}

	// Get permissions for the role
	permissions, err := r.GetRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions

	// Cache the role
	if err := r.cache.Set(cacheKey, role); err != nil {
		log.Debug().Err(err).Msg("Failed to cache role")
	}

	return &role, nil
}

// GetAll retrieves all roles
func (r *MongoRoleRepository) GetAll(ctx context.Context) ([]*models.Role, error) {
	cacheKey := "roles:all"

	// Try to get from cache first
	var roles []*models.Role
	found, err := r.cache.Get(cacheKey, &roles)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get roles from cache")
	}

	if found {
		// Get permissions for each role
		for i, role := range roles {
			permissions, err := r.GetRolePermissions(ctx, role.ID)
			if err != nil {
				return nil, err
			}
			roles[i].Permissions = permissions
		}
		return roles, nil
	}

	// If not in cache, get from database
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.rolesCollection().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	roles = make([]*models.Role, 0)
	for cursor.Next(ctx) {
		var role models.Role
		if err := cursor.Decode(&role); err != nil {
			return nil, fmt.Errorf("failed to decode role from MongoDB: %w", err)
		}

		// Get permissions for the role
		permissions, err := r.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions

		roles = append(roles, &role)
	}

	// Cache the roles
	if err := r.cache.Set(cacheKey, roles); err != nil {
		log.Debug().Err(err).Msg("Failed to cache roles")
	}

	return roles, nil
}

// Update updates a role in the database
func (r *MongoRoleRepository) Update(ctx context.Context, role *models.Role) error {
	role.UpdatedAt = time.Now()

	filter := bson.M{"_id": role.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        role.Name,
			"description": role.Description,
			"updated_at":  role.UpdatedAt,
		},
	}

	result, err := r.rolesCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update role in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role not found")
	}

	// Clear cache
	r.invalidateRoleCache()

	return nil
}

// Delete deletes a role from the database
func (r *MongoRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}

	result, err := r.rolesCollection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete role from MongoDB: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("role not found")
	}

	// Also delete role-permissions relationships
	_, err = r.rolePermissionsCollection().DeleteMany(ctx, bson.M{"role_id": id})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete role-permissions relationships")
	}

	// Clear cache
	r.invalidateRoleCache()

	return nil
}

// AssignPermissionsToRole assigns permissions to a role
func (r *MongoRoleRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Start a session for transaction
	session, err := r.db.Client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start MongoDB session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		// Remove existing permissions
		_, err := r.rolePermissionsCollection().DeleteMany(sessionContext, bson.M{"role_id": roleID})
		if err != nil {
			return fmt.Errorf("failed to remove existing permissions: %w", err)
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

			_, err = r.rolePermissionsCollection().InsertMany(sessionContext, rolePermissions)
			if err != nil {
				return fmt.Errorf("failed to assign permissions: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to assign permissions transaction: %w", err)
	}

	// Clear role cache
	r.invalidateRoleCache()
	// Also invalidate user cache since permissions may have changed
	r.invalidateUserPermissionCache()

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *MongoRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	// Get permission IDs assigned to the role
	cursor, err := r.rolePermissionsCollection().Find(ctx, bson.M{"role_id": roleID})
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	permissionIDs := make([]uuid.UUID, 0)
	for cursor.Next(ctx) {
		var rolePermission struct {
			PermissionID uuid.UUID `bson:"permission_id"`
		}
		if err := cursor.Decode(&rolePermission); err != nil {
			return nil, fmt.Errorf("failed to decode role permission: %w", err)
		}
		permissionIDs = append(permissionIDs, rolePermission.PermissionID)
	}

	// Get permission details for each permission ID
	permissions := make([]models.Permission, 0, len(permissionIDs))
	for _, permID := range permissionIDs {
		filter := bson.M{"_id": permID}
		var permission models.Permission

		err := r.permissionsCollection().FindOne(ctx, filter).Decode(&permission)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Debug().Str("permission_id", permID.String()).Msg("Permission not found")
				continue
			}
			return nil, fmt.Errorf("failed to get permission from MongoDB: %w", err)
		}

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// invalidateRoleCache clears all role-related cache
func (r *MongoRoleRepository) invalidateRoleCache() {
	if err := r.cache.DeleteByPattern("role:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate role cache")
	}

	if err := r.cache.DeleteByPattern("roles:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate roles cache")
	}
}

// invalidateUserPermissionCache clears user permission cache
func (r *MongoRoleRepository) invalidateUserPermissionCache() {
	if err := r.cache.DeleteByPattern("user:permissions:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate user permission cache")
	}
}

// ExecuteTx executes a function within a transaction
func (r *MongoRoleRepository) ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error {
	// Start a session for transaction
	session, err := r.db.Client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start MongoDB session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute the transaction
	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		if err := sessionContext.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Create a special MongoDB transaction repository
		txRepo := &MongoDBTxRepository{
			db:      r.db,
			session: session,
			ctx:     sessionContext,
		}

		// Execute the function with the tx repository
		if err := fn(txRepo); err != nil {
			if abortErr := sessionContext.AbortTransaction(sessionContext); abortErr != nil {
				log.Error().Err(abortErr).Msg("Failed to abort transaction")
			}
			return err
		}

		// Commit the transaction
		if err := sessionContext.CommitTransaction(sessionContext); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Clear cache after transaction
	r.invalidateRoleCache()

	return nil
}
