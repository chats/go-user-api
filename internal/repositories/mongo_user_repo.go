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

// MongoUserRepository handles database operations for users with MongoDB
type MongoUserRepository struct {
	db    *database.MongoDB
	cache *cache.RedisClient
}

// NewMongoUserRepository creates a new MongoDB user repository
func NewMongoUserRepository(db *database.MongoDB, cache *cache.RedisClient) *MongoUserRepository {
	return &MongoUserRepository{
		db:    db,
		cache: cache,
	}
}

// usersCollection returns the MongoDB collection for users
func (r *MongoUserRepository) usersCollection() *mongo.Collection {
	return r.db.GetCollection("users")
}

// userRolesCollection returns the MongoDB collection for user-roles relationship
func (r *MongoUserRepository) userRolesCollection() *mongo.Collection {
	return r.db.GetCollection("user_roles")
}

// rolesCollection returns the MongoDB collection for roles
func (r *MongoUserRepository) rolesCollection() *mongo.Collection {
	return r.db.GetCollection("roles")
}

// permissionsCollection returns the MongoDB collection for permissions
func (r *MongoUserRepository) permissionsCollection() *mongo.Collection {
	return r.db.GetCollection("permissions")
}

// rolePermissionsCollection returns the MongoDB collection for role-permissions relationship
func (r *MongoUserRepository) rolePermissionsCollection() *mongo.Collection {
	return r.db.GetCollection("role_permissions")
}

// Create creates a new user in the database
func (r *MongoUserRepository) Create(ctx context.Context, user *models.User) error {
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
	_, err := r.usersCollection().InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user in MongoDB: %w", err)
	}

	// Clear cache
	r.invalidateUserCache()

	return nil
}

// GetByID retrieves a user by ID
func (r *MongoUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:%s", id.String())

	// Try to get from cache first
	var user models.User
	found, err := r.cache.Get(cacheKey, &user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user from cache")
	}

	if found {
		// Get roles for the user
		roles, err := r.GetUserRoles(ctx, id)
		if err != nil {
			return nil, err
		}
		user.Roles = roles
		return &user, nil
	}

	// If not in cache, get from database
	filter := bson.M{"_id": id}

	result := r.usersCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user from MongoDB: %w", err)
	}

	// Get roles for the user
	roles, err := r.GetUserRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	// Cache the user
	if err := r.cache.Set(cacheKey, user); err != nil {
		log.Debug().Err(err).Msg("Failed to cache user")
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *MongoUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:username:%s", username)

	// Try to get from cache first
	var user models.User
	found, err := r.cache.Get(cacheKey, &user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user from cache")
	}

	if found {
		// Get roles for the user
		roles, err := r.GetUserRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		user.Roles = roles
		return &user, nil
	}

	// If not in cache, get from database
	filter := bson.M{"username": username}

	result := r.usersCollection().FindOne(ctx, filter)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user from MongoDB: %w", result.Err())
	}

	if err := result.Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user from MongoDB: %w", err)
	}

	// Get roles for the user
	roles, err := r.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	// Cache the user
	if err := r.cache.Set(cacheKey, user); err != nil {
		log.Debug().Err(err).Msg("Failed to cache user")
	}

	return &user, nil
}

// GetAll retrieves all users with pagination
func (r *MongoUserRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.User, error) {
	cacheKey := fmt.Sprintf("users:limit:%d:offset:%d", limit, offset)

	// Try to get from cache first
	var users []*models.User
	found, err := r.cache.Get(cacheKey, &users)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get users from cache")
	}

	if found {
		// Get roles for each user
		for i, user := range users {
			roles, err := r.GetUserRoles(ctx, user.ID)
			if err != nil {
				return nil, err
			}
			users[i].Roles = roles
		}
		return users, nil
	}

	// If not in cache, get from database
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(offset))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.usersCollection().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get users from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	users = make([]*models.User, 0)
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user from MongoDB: %w", err)
		}

		// Get roles for the user
		roles, err := r.GetUserRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		user.Roles = roles

		users = append(users, &user)
	}

	// Cache the users
	if err := r.cache.Set(cacheKey, users); err != nil {
		log.Debug().Err(err).Msg("Failed to cache users")
	}

	return users, nil
}

// Update updates a user in the database
func (r *MongoUserRepository) Update(ctx context.Context, user *models.User) error {
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

	result, err := r.usersCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	// Clear cache
	r.invalidateUserCache()

	return nil
}

// UpdatePassword updates a user's password
func (r *MongoUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		},
	}

	result, err := r.usersCollection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update password in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	// Clear cache
	r.invalidateUserCache()

	return nil
}

// Delete deletes a user from the database
func (r *MongoUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}

	result, err := r.usersCollection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user from MongoDB: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user not found")
	}

	// Also delete user roles relationships
	_, err = r.userRolesCollection().DeleteMany(ctx, bson.M{"user_id": id})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete user roles relationships")
	}

	// Clear cache
	r.invalidateUserCache()

	return nil
}

// AssignRolesToUser assigns roles to a user
func (r *MongoUserRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Start a session for transaction
	session, err := r.db.Client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start MongoDB session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		// Remove existing roles
		_, err := r.userRolesCollection().DeleteMany(sessionContext, bson.M{"user_id": userID})
		if err != nil {
			return fmt.Errorf("failed to remove existing roles: %w", err)
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

			_, err = r.userRolesCollection().InsertMany(sessionContext, userRoles)
			if err != nil {
				return fmt.Errorf("failed to assign roles: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to assign roles transaction: %w", err)
	}

	// Clear cache
	r.invalidateUserCache()

	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *MongoUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	// Get role IDs assigned to the user
	cursor, err := r.userRolesCollection().Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	roleIDs := make([]uuid.UUID, 0)
	for cursor.Next(ctx) {
		var userRole struct {
			RoleID uuid.UUID `bson:"role_id"`
		}
		if err := cursor.Decode(&userRole); err != nil {
			return nil, fmt.Errorf("failed to decode user role: %w", err)
		}
		roleIDs = append(roleIDs, userRole.RoleID)
	}

	// Get role details for each role ID
	roles := make([]models.Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		filter := bson.M{"_id": roleID}
		var role models.Role

		err := r.rolesCollection().FindOne(ctx, filter).Decode(&role)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Debug().Str("role_id", roleID.String()).Msg("Role not found")
				continue
			}
			return nil, fmt.Errorf("failed to get role from MongoDB: %w", err)
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *MongoUserRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	// First, get all role IDs assigned to the user
	userRolesCursor, err := r.userRolesCollection().Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles from MongoDB: %w", err)
	}
	defer userRolesCursor.Close(ctx)

	roleIDs := make([]uuid.UUID, 0)
	for userRolesCursor.Next(ctx) {
		var userRole struct {
			RoleID uuid.UUID `bson:"role_id"`
		}
		if err := userRolesCursor.Decode(&userRole); err != nil {
			return nil, fmt.Errorf("failed to decode user role: %w", err)
		}
		roleIDs = append(roleIDs, userRole.RoleID)
	}

	// Now, get all permission IDs assigned to these roles
	permissionMap := make(map[uuid.UUID]bool)
	for _, roleID := range roleIDs {
		rolePermsCursor, err := r.rolePermissionsCollection().Find(ctx, bson.M{"role_id": roleID})
		if err != nil {
			return nil, fmt.Errorf("failed to get role permissions from MongoDB: %w", err)
		}

		for rolePermsCursor.Next(ctx) {
			var rolePerm struct {
				PermissionID uuid.UUID `bson:"permission_id"`
			}
			if err := rolePermsCursor.Decode(&rolePerm); err != nil {
				rolePermsCursor.Close(ctx)
				return nil, fmt.Errorf("failed to decode role permission: %w", err)
			}
			permissionMap[rolePerm.PermissionID] = true
		}
		rolePermsCursor.Close(ctx)
	}

	// Finally, get permission details for each permission ID
	permissionIDs := make([]uuid.UUID, 0, len(permissionMap))
	for permID := range permissionMap {
		permissionIDs = append(permissionIDs, permID)
	}

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

// HasPermission checks if a user has a specific permission
func (r *MongoUserRepository) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	// Get all role IDs assigned to the user
	userRolesCursor, err := r.userRolesCollection().Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return false, fmt.Errorf("failed to get user roles from MongoDB: %w", err)
	}
	defer userRolesCursor.Close(ctx)

	roleIDs := make([]uuid.UUID, 0)
	for userRolesCursor.Next(ctx) {
		var userRole struct {
			RoleID uuid.UUID `bson:"role_id"`
		}
		if err := userRolesCursor.Decode(&userRole); err != nil {
			return false, fmt.Errorf("failed to decode user role: %w", err)
		}
		roleIDs = append(roleIDs, userRole.RoleID)
	}

	if len(roleIDs) == 0 {
		return false, nil
	}

	// First, find the permission with the specified resource and action
	filter := bson.M{"resource": resource, "action": action}
	var permission models.Permission

	err = r.permissionsCollection().FindOne(ctx, filter).Decode(&permission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Permission does not exist
			return false, nil
		}
		return false, fmt.Errorf("failed to get permission from MongoDB: %w", err)
	}

	// Now check if any of the user's roles has this permission
	for _, roleID := range roleIDs {
		filter = bson.M{"role_id": roleID, "permission_id": permission.ID}
		count, err := r.rolePermissionsCollection().CountDocuments(ctx, filter)
		if err != nil {
			return false, fmt.Errorf("failed to check role permission: %w", err)
		}

		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}

// CountUsers counts the total number of users
func (r *MongoUserRepository) CountUsers(ctx context.Context) (int, error) {
	cacheKey := "users:count"

	// Try to get from cache first
	var count int
	found, err := r.cache.Get(cacheKey, &count)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user count from cache")
	}

	if found {
		return count, nil
	}

	// If not in cache, get from database
	count64, err := r.usersCollection().CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count users in MongoDB: %w", err)
	}

	count = int(count64)

	// Cache the count
	if err := r.cache.Set(cacheKey, count); err != nil {
		log.Debug().Err(err).Msg("Failed to cache user count")
	}

	return count, nil
}

// invalidateUserCache clears all user-related cache
func (r *MongoUserRepository) invalidateUserCache() {
	if err := r.cache.DeleteByPattern("user:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate user cache")
	}

	if err := r.cache.DeleteByPattern("users:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate users cache")
	}
}
