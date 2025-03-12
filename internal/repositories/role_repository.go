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

// RoleRepository handles database operations for roles
type RoleRepository struct {
	db    *database.DB
	cache *cache.RedisClient
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *database.DB, cache *cache.RedisClient) *RoleRepository {
	return &RoleRepository{
		db:    db,
		cache: cache,
	}
}

// Create creates a new role in the database
func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	query := `
		INSERT INTO roles (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowxContext(
		ctx,
		query,
		role.Name,
		role.Description,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// Clear role cache
	r.invalidateRoleCache()

	return nil
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	cacheKey := fmt.Sprintf("role:%s", id.String())

	// Try to get from cache first
	var role models.Role
	found, err := r.cache.Get(cacheKey, &role)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get role from cache")
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
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	if err := r.db.GetContext(ctx, &role, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Get permissions for the role
	permissions, err := r.GetRolePermissions(ctx, id)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions

	// Cache the role
	if err := r.cache.Set(cacheKey, role); err != nil {
		log.Warn().Err(err).Msg("Failed to cache role")
	}

	return &role, nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	cacheKey := fmt.Sprintf("role:name:%s", name)

	// Try to get from cache first
	var role models.Role
	found, err := r.cache.Get(cacheKey, &role)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get role from cache")
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
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	if err := r.db.GetContext(ctx, &role, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Get permissions for the role
	permissions, err := r.GetRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions

	// Cache the role
	if err := r.cache.Set(cacheKey, role); err != nil {
		log.Warn().Err(err).Msg("Failed to cache role")
	}

	return &role, nil
}

// GetAll retrieves all roles
func (r *RoleRepository) GetAll(ctx context.Context) ([]*models.Role, error) {
	cacheKey := "roles:all"

	// Try to get from cache first
	var roles []*models.Role
	found, err := r.cache.Get(cacheKey, &roles)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get roles from cache")
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
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name
	`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	defer rows.Close()

	roles = make([]*models.Role, 0)
	for rows.Next() {
		var role models.Role
		if err := rows.StructScan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
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
		log.Warn().Err(err).Msg("Failed to cache roles")
	}

	return roles, nil
}

// Update updates a role in the database
func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	role.UpdatedAt = time.Now()

	query := `
		UPDATE roles
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		role.Name,
		role.Description,
		role.UpdatedAt,
		role.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	// Clear role cache
	r.invalidateRoleCache()

	return nil
}

// Delete deletes a role from the database
func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM roles WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not found")
	}

	// Clear role cache
	r.invalidateRoleCache()

	return nil
}

// AssignPermissionsToRole assigns permissions to a role
func (r *RoleRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Remove existing permissions
	_, err = tx.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove existing permissions: %w", err)
	}

	// Assign new permissions
	for _, permissionID := range permissionIDs {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)",
			roleID,
			permissionID,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to assign permission: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clear role cache
	r.invalidateRoleCache()
	// Also invalidate user cache since permissions may have changed
	r.invalidateUserPermissionCache()

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	query := `
		SELECT p.id, p.name, p.description, p.resource, p.action, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`

	var permissions []models.Permission
	err := r.db.SelectContext(ctx, &permissions, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return permissions, nil
}

// invalidateRoleCache clears all role-related cache
func (r *RoleRepository) invalidateRoleCache() {
	if err := r.cache.DeleteByPattern("role:*"); err != nil {
		log.Warn().Err(err).Msg("Failed to invalidate role cache")
	}

	if err := r.cache.DeleteByPattern("roles:*"); err != nil {
		log.Warn().Err(err).Msg("Failed to invalidate roles cache")
	}
}

// invalidateUserPermissionCache clears user permission cache
func (r *RoleRepository) invalidateUserPermissionCache() {
	if err := r.cache.DeleteByPattern("user:permissions:*"); err != nil {
		log.Warn().Err(err).Msg("Failed to invalidate user permission cache")
	}
}

// ExecuteTx executes a function within a transaction
func (r *RoleRepository) ExecuteTx(ctx context.Context, fn func(*TxRepository) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := &TxRepository{tx: tx}
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
	r.invalidateRoleCache()

	return nil
}
