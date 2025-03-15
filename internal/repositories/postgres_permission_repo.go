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

// PermissionRepository handles database operations for permissions
type PermissionRepository struct {
	db    *database.PostgresDB
	cache *cache.RedisClient
}

// Ensure PermissionRepository implements PermissionRepositoryInterface
var _ PermissionRepositoryInterface = (*PermissionRepository)(nil)

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(db *database.PostgresDB, cache *cache.RedisClient) *PermissionRepository {
	return &PermissionRepository{
		db:    db,
		cache: cache,
	}
}

// Create creates a new permission in the database
func (r *PermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
	query := `
		INSERT INTO permissions (name, description, resource, action)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowxContext(
		ctx,
		query,
		permission.Name,
		permission.Description,
		permission.Resource,
		permission.Action,
	).Scan(&permission.ID, &permission.CreatedAt, &permission.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	// Clear permission cache
	r.invalidatePermissionCache()

	return nil
}

// GetByID retrieves a permission by ID
func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
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
	query := `
		SELECT id, name, description, resource, action, created_at, updated_at
		FROM permissions
		WHERE id = $1
	`

	if err := r.db.GetContext(ctx, &permission, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	// Cache the permission
	if err := r.cache.Set(cacheKey, permission); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permission")
	}

	return &permission, nil
}

// GetByResourceAction retrieves a permission by resource and action
func (r *PermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error) {
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
	query := `
		SELECT id, name, description, resource, action, created_at, updated_at
		FROM permissions
		WHERE resource = $1 AND action = $2
	`

	if err := r.db.GetContext(ctx, &permission, query, resource, action); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	// Cache the permission
	if err := r.cache.Set(cacheKey, permission); err != nil {
		log.Debug().Err(err).Msg("Failed to cache permission")
	}

	return &permission, nil
}

// GetAll retrieves all permissions
func (r *PermissionRepository) GetAll(ctx context.Context) ([]*models.Permission, error) {
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
	query := `
		SELECT id, name, description, resource, action, created_at, updated_at
		FROM permissions
		ORDER BY resource, action
	`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	permissions = make([]*models.Permission, 0)
	for rows.Next() {
		var permission models.Permission
		if err := rows.StructScan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
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
func (r *PermissionRepository) Update(ctx context.Context, permission *models.Permission) error {
	permission.UpdatedAt = time.Now()

	query := `
		UPDATE permissions
		SET name = $1, description = $2, resource = $3, action = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		permission.Name,
		permission.Description,
		permission.Resource,
		permission.Action,
		permission.UpdatedAt,
		permission.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	// Clear permission cache
	r.invalidatePermissionCache()

	return nil
}

// Delete deletes a permission from the database
func (r *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM permissions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("permission not found")
	}

	// Clear permission cache
	r.invalidatePermissionCache()

	return nil
}

// GetByResource retrieves all permissions for a specific resource
func (r *PermissionRepository) GetByResource(ctx context.Context, resource string) ([]*models.Permission, error) {
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
	query := `
		SELECT id, name, description, resource, action, created_at, updated_at
		FROM permissions
		WHERE resource = $1
		ORDER BY action
	`

	rows, err := r.db.QueryxContext(ctx, query, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	permissions = make([]*models.Permission, 0)
	for rows.Next() {
		var permission models.Permission
		if err := rows.StructScan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
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
func (r *PermissionRepository) invalidatePermissionCache() {
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
func (r *PermissionRepository) ExecuteTx(ctx context.Context, fn func(TxRepositoryInterface) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := &PostgresqlTxRepository{tx: tx}
	err = fn(txRepo)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx failed: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear caches after successful transaction
	r.invalidatePermissionCache()

	return nil
}
