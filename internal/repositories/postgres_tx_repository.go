package repositories

import (
	"context"
	"fmt"

	"github.com/chats/go-user-api/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresqlTxRepository provides transaction-based operations
type PostgresqlTxRepository struct {
	tx *sqlx.Tx
}

// Ensure PostgresqlTxRepository implements TxRepositoryInterface
var _ TxRepositoryInterface = (*PostgresqlTxRepository)(nil)

// CreateUser creates a new user within a transaction
func (r *PostgresqlTxRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password, first_name, last_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := r.tx.QueryRowxContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.FirstName,
		user.LastName,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user in transaction: %w", err)
	}

	return nil
}

// UpdateUser updates a user within a transaction
func (r *PostgresqlTxRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, first_name = $3, last_name = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`

	_, err := r.tx.ExecContext(
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
		return fmt.Errorf("failed to update user in transaction: %w", err)
	}

	return nil
}

// UpdateUserPassword updates a user password within a transaction
func (r *PostgresqlTxRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.tx.ExecContext(ctx, query, hashedPassword, userID)
	if err != nil {
		return fmt.Errorf("failed to update password in transaction: %w", err)
	}

	return nil
}

// AssignRolesToUser assigns roles to a user within a transaction
func (r *PostgresqlTxRepository) AssignRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Remove existing roles
	_, err := r.tx.ExecContext(ctx, "DELETE FROM user_roles WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to remove existing roles in transaction: %w", err)
	}

	// Assign new roles
	for _, roleID := range roleIDs {
		_, err = r.tx.ExecContext(
			ctx,
			"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)",
			userID,
			roleID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign role in transaction: %w", err)
		}
	}

	return nil
}

// CreateRole creates a new role within a transaction
func (r *PostgresqlTxRepository) CreateRole(ctx context.Context, role *models.Role) error {
	query := `
		INSERT INTO roles (name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.tx.QueryRowxContext(
		ctx,
		query,
		role.Name,
		role.Description,
		role.CreatedAt,
		role.UpdatedAt,
	).Scan(&role.ID)

	if err != nil {
		return fmt.Errorf("failed to create role in transaction: %w", err)
	}

	return nil
}

// UpdateRole updates a role within a transaction
func (r *PostgresqlTxRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	query := `
		UPDATE roles
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.tx.ExecContext(
		ctx,
		query,
		role.Name,
		role.Description,
		role.UpdatedAt,
		role.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update role in transaction: %w", err)
	}

	return nil
}

// AssignPermissionsToRole assigns permissions to a role within a transaction
func (r *PostgresqlTxRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Remove existing permissions
	_, err := r.tx.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("failed to remove existing permissions in transaction: %w", err)
	}

	// Assign new permissions
	for _, permissionID := range permissionIDs {
		_, err = r.tx.ExecContext(
			ctx,
			"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)",
			roleID,
			permissionID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign permission in transaction: %w", err)
		}
	}

	return nil
}

// CreatePermission creates a new permission within a transaction
func (r *PostgresqlTxRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	query := `
		INSERT INTO permissions (name, description, resource, action, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.tx.QueryRowxContext(
		ctx,
		query,
		permission.Name,
		permission.Description,
		permission.Resource,
		permission.Action,
		permission.CreatedAt,
		permission.UpdatedAt,
	).Scan(&permission.ID)

	if err != nil {
		return fmt.Errorf("failed to create permission in transaction: %w", err)
	}

	return nil
}

// UpdatePermission updates a permission within a transaction
func (r *PostgresqlTxRepository) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	query := `
		UPDATE permissions
		SET name = $1, description = $2, resource = $3, action = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.tx.ExecContext(
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
		return fmt.Errorf("failed to update permission in transaction: %w", err)
	}

	return nil
}
