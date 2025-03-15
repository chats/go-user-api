package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/cache"
	"github.com/chats/go-user-api/internal/database"
	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db    *database.PostgresDB
	cache *cache.RedisClient
}

// Ensure UserRepository implements UserRepositoryInterface
var _ UserRepositoryInterface = (*UserRepository)(nil)

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.PostgresDB, cache *cache.RedisClient) *UserRepository {
	return &UserRepository{
		db:    db,
		cache: cache,
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowxContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.FirstName,
		user.LastName,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Clear user cache
	r.invalidateUserCache()

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
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
	query := `
		SELECT id, username, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
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
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
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
	query := `
		SELECT id, username, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	if err := r.db.GetContext(ctx, &user, query, username); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
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
func (r *UserRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.User, error) {
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
	query := `
		SELECT id, username, email, password, first_name, last_name, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryxContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	users = make([]*models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.StructScan(&user); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
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
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET username = $1, email = $2, first_name = $3, last_name = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.FirstName,
		user.LastName,
		user.IsActive,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Clear user cache
	r.invalidateUserCache()

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		hashedPassword,
		time.Now(),
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Clear user cache
	r.invalidateUserCache()

	return nil
}

// Delete deletes a user from the database
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	// Clear user cache
	r.invalidateUserCache()

	return nil
}

// AssignRolesToUser assigns roles to a user
func (r *UserRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Remove existing roles
	_, err = tx.ExecContext(ctx, "DELETE FROM user_roles WHERE user_id = $1", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove existing roles: %w", err)
	}

	// Assign new roles
	for _, roleID := range roleIDs {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)",
			userID,
			roleID,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to assign role: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear user cache
	r.invalidateUserCache()

	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *UserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	var roles []models.Role
	err := r.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return roles, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *UserRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.resource, p.action, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
	`

	var permissions []models.Permission
	err := r.db.SelectContext(ctx, &permissions, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return permissions, nil
}

// HasPermission checks if a user has a specific permission
func (r *UserRepository) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN user_roles ur ON rp.role_id = ur.role_id
			WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3
		)
	`

	var hasPermission bool
	err := r.db.GetContext(ctx, &hasPermission, query, userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return hasPermission, nil
}

// CountUsers counts the total number of users
func (r *UserRepository) CountUsers(ctx context.Context) (int, error) {
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
	query := `SELECT COUNT(*) FROM users`

	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Cache the count
	if err := r.cache.Set(cacheKey, count); err != nil {
		log.Debug().Err(err).Msg("Failed to cache user count")
	}

	return count, nil
}

// invalidateUserCache clears all user-related cache
func (r *UserRepository) invalidateUserCache() {
	if err := r.cache.DeleteByPattern("user:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate user cache")
	}

	if err := r.cache.DeleteByPattern("users:*"); err != nil {
		log.Debug().Err(err).Msg("Failed to invalidate users cache")
	}
}
