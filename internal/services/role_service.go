package services

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// RoleService handles role-related operations
type RoleService struct {
	roleRepo       repositories.RoleRepositoryInterface
	permissionRepo repositories.PermissionRepositoryInterface
	txManager      transaction.Manager[transaction.Repository]
}

// NewRoleService creates a new role service
func NewRoleService(
	roleRepo repositories.RoleRepositoryInterface,
	permissionRepo repositories.PermissionRepositoryInterface,
	txManager transaction.Manager[transaction.Repository],
) *RoleService {
	return &RoleService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		txManager:      txManager,
	}
}

// CreateRole creates a new role
func (s *RoleService) CreateRole(ctx context.Context, request models.RoleCreateRequest) (*models.RoleResponse, error) {
	// Check if role name already exists
	existingRole, err := s.roleRepo.GetByName(ctx, request.Name)
	if err == nil && existingRole != nil {
		return nil, fmt.Errorf("role name already exists")
	}

	// Create role object
	role := &models.Role{
		Name:        request.Name,
		Description: request.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Start transaction
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Save role to database
		if err := tx.CreateRole(ctx, role); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}

		// Assign permissions if provided
		if len(request.PermissionIDs) > 0 {
			permissionIDs := make([]uuid.UUID, 0, len(request.PermissionIDs))
			for _, permissionIDStr := range request.PermissionIDs {
				permissionID, err := uuid.Parse(permissionIDStr)
				if err != nil {
					return fmt.Errorf("invalid permission ID: %w", err)
				}
				permissionIDs = append(permissionIDs, permissionID)
			}

			if err := tx.AssignPermissionsToRole(ctx, role.ID, permissionIDs); err != nil {
				return fmt.Errorf("failed to assign permissions: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the updated role with permissions
	updatedRole, err := s.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get updated role after creation")
		// Return the role without permissions as fallback
		// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
		response := role.ToResponse()
		return &response, nil
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := updatedRole.ToResponse()
	return &response, nil
}

// GetRoleByID retrieves a role by ID
func (s *RoleService) GetRoleByID(ctx context.Context, id string) (*models.RoleResponse, error) {
	// Parse UUID
	roleID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	// Get role
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := role.ToResponse()
	return &response, nil
}

// GetAllRoles retrieves all roles
func (s *RoleService) GetAllRoles(ctx context.Context) ([]models.RoleResponse, error) {
	// Get roles
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	roleResponses := make([]models.RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = role.ToResponse()
	}

	return roleResponses, nil
}

// UpdateRole updates a role
func (s *RoleService) UpdateRole(ctx context.Context, id string, request models.RoleUpdateRequest) (*models.RoleResponse, error) {
	// Parse UUID
	roleID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	// Get existing role
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// Check for name uniqueness if name is being updated
	if request.Name != "" && request.Name != role.Name {
		existingRole, err := s.roleRepo.GetByName(ctx, request.Name)
		if err == nil && existingRole != nil {
			return nil, fmt.Errorf("role name already exists")
		}
	}

	// Update fields if provided
	if request.Name != "" {
		role.Name = request.Name
	}
	if request.Description != "" {
		role.Description = request.Description
	}
	role.UpdatedAt = time.Now()

	// Start transaction
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Update role in database
		if err := tx.UpdateRole(ctx, role); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}

		// Update permissions if provided
		if len(request.PermissionIDs) > 0 {
			permissionIDs := make([]uuid.UUID, 0, len(request.PermissionIDs))
			for _, permissionIDStr := range request.PermissionIDs {
				permissionID, err := uuid.Parse(permissionIDStr)
				if err != nil {
					return fmt.Errorf("invalid permission ID: %w", err)
				}
				permissionIDs = append(permissionIDs, permissionID)
			}

			if err := tx.AssignPermissionsToRole(ctx, role.ID, permissionIDs); err != nil {
				return fmt.Errorf("failed to assign permissions: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the updated role with permissions
	updatedRole, err := s.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get updated role after update")
		// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
		response := role.ToResponse()
		return &response, nil
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := updatedRole.ToResponse()
	return &response, nil
}

// DeleteRole deletes a role
func (s *RoleService) DeleteRole(ctx context.Context, id string) error {
	// Parse UUID
	roleID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	// Delete role
	return s.roleRepo.Delete(ctx, roleID)
}

// GetRolePermissions retrieves all permissions for a role
func (s *RoleService) GetRolePermissions(ctx context.Context, id string) ([]models.PermissionResponse, error) {
	// Parse UUID
	roleID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	// Get permissions
	permissions, err := s.roleRepo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	permissionResponses := make([]models.PermissionResponse, len(permissions))
	for i, permission := range permissions {
		permissionResponses[i] = permission.ToResponse()
	}

	return permissionResponses, nil
}
