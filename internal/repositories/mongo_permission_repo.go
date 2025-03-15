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

// MongoPermissionRepository handles database operations for permissions with MongoDB
type MongoPermissionRepository struct {
	db    *database.MongoDB
	cache *cache.RedisClient
}

// NewMongoPermissionRepository creates a new MongoDB permission repository
func NewMongoPermissionRepository(db *database.MongoDB, cache *cache.RedisClient) *MongoPermissionRepository {
	return &MongoPermissionRepository{
		db:    db,
		cache: cache,
	}
}

// permissionsCollection returns the MongoDB collection for permissions
func (r *MongoPermissionRepository) permissionsCollection() *mongo.Collection {
	return r.db.GetCollection("permissions")
}

// Create creates a new permission in the database
func (r *MongoPermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
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
	_, err := r.permissionsCollection().InsertOne(ctx, permission)
	if err != nil {
		return fmt.Errorf("failed to create permission in MongoDB: %w", err)
	}

	// Clear cache
	r.invalidatePermissionCache()

	return nil
}

// GetByID retrieves a permission by ID
func (r *MongoPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	cacheKey := fmt.Sprintf("permission:%s", id.String())

	// Try to get from cache first
	var permission models.Permission
	found, err := r.cache.Get(cacheKey, &permission)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get permission from cache")
	}

	if found {
		return &permission, nil
	}

	// If not in cache, get from database
	filter := bson.M{"_id": id}

	result := r.permissionsCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&permission); err != nil {
		return nil, fmt.Errorf("failed to decode permission from MongoDB: %w", err)
	}

	// Cache the permission
	if err := r.cache.Set(cacheKey, permission); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permission")
	}

	return &permission, nil
}

// GetByResourceAction retrieves a permission by resource and action
func (r *MongoPermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error) {
	cacheKey := fmt.Sprintf("permission:resource:%s:action:%s", resource, action)

	// Try to get from cache first
	var permission models.Permission
	found, err := r.cache.Get(cacheKey, &permission)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get permission from cache")
	}

	if found {
		return &permission, nil
	}

	// If not in cache, get from database
	filter := bson.M{"resource": resource, "action": action}

	result := r.permissionsCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&permission); err != nil {
		return nil, fmt.Errorf("failed to decode permission from MongoDB: %w", err)
	}

	// Cache the permission
	if err := r.cache.Set(cacheKey, permission); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permission")
	}

	return &permission, nil
}

// GetAll retrieves all permissions
func (r *MongoPermissionRepository) GetAll(ctx context.Context) ([]*models.Permission, error) {
	cacheKey := "permissions:all"

	// Try to get from cache first
	var permissions []*models.Permission
	found, err := r.cache.Get(cacheKey, &permissions)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get permissions from cache")
	}

	if found {
		return permissions, nil
	}

	// If not in cache, get from database
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}})

	cursor, err := r.permissionsCollection().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	permissions = make([]*models.Permission, 0)
	for cursor.Next(ctx) {
		var permission models.Permission
		if err := cursor.Decode(&permission); err != nil {
			return nil, fmt.Errorf("failed to decode permission from MongoDB: %w", err)
		}

		permissions = append(permissions, &permission)
	}

	// Cache the permissions
	if err := r.cache.Set(cacheKey, permissions); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permissions")
	}

	return permissions, nil
}

// Update updates a permission in the database
func (r *MongoPermissionRepository) Update(ctx context.Context, permission *models.Permission) error {
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

	result, err := r.permissionsCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update permission in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("permission not found")
	}

	// Clear cache
	r.invalidatePermissionCache()

	return nil
}

// Delete deletes a permission from the database
func (r *MongoPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}

	result, err := r.permissionsCollection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete permission from MongoDB: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("permission not found")
	}

	// Clear cache
	r.invalidatePermissionCache()

	return nil
}

// GetByResource retrieves all permissions for a specific resource
func (r *MongoPermissionRepository) GetByResource(ctx context.Context, resource string) ([]*models.Permission, error) {
	cacheKey := fmt.Sprintf("permissions:resource:%s", resource)

	// Try to get from cache first
	var permissions []*models.Permission
	found, err := r.cache.Get(cacheKey, &permissions)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get permissions from cache")
	}

	if found {
		return permissions, nil
	}

	// If not in cache, get from database
	filter := bson.M{"resource": resource}
	findOptions := options.Find().SetSort(bson.D{{Key: "action", Value: 1}})

	cursor, err := r.permissionsCollection().Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	permissions = make([]*models.Permission, 0)
	for cursor.Next(ctx) {
		var permission models.Permission
		if err := cursor.Decode(&permission); err != nil {
			return nil, fmt.Errorf("failed to decode permission from MongoDB: %w", err)
		}

		permissions = append(permissions, &permission)
	}

	// Cache the permissions
	if err := r.cache.Set(cacheKey, permissions); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permissions")
	}

	return permissions, nil
}

// invalidatePermissionCache clears all permission-related cache
func (r *MongoPermissionRepository) invalidatePermissionCache() {
	if err := r.cache.DeleteByPattern("permission:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate permission cache")
	}

	if err := r.cache.DeleteByPattern("permissions:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate permissions cache")
	}

	// Also invalidate role cache since permissions might have changed
	if err := r.cache.DeleteByPattern("role:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate role cache")
	}

	// Also invalidate user permission cache
	if err := r.cache.DeleteByPattern("user:permissions:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate user permission cache")
	}
}

// ExecuteTx executes a function within a transaction
func (r *MongoPermissionRepository) ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error {
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

	// Clear caches after successful transaction
	r.invalidatePermissionCache()

	return nil
}
