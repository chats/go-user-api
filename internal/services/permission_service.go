package services

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/google/uuid"
)

// PermissionService handles permission-related operations
type PermissionService struct {
	permissionRepo repositories.PermissionRepositoryInterface
	txManager      transaction.Manager[transaction.Repository]
}

// NewPermissionService creates a new permission service
func NewPermissionService(
	permissionRepo repositories.PermissionRepositoryInterface,
	txManager transaction.Manager[transaction.Repository],
) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
		txManager:      txManager,
	}
}

// CreatePermission creates a new permission
func (s *PermissionService) CreatePermission(ctx context.Context, request models.PermissionCreateRequest) (*models.PermissionResponse, error) {
	// Check if permission already exists for the resource and action
	existingPermission, err := s.permissionRepo.GetByResourceAction(ctx, request.Resource, request.Action)
	if err == nil && existingPermission != nil {
		return nil, fmt.Errorf("permission already exists for this resource and action")
	}

	// Create permission object
	permission := &models.Permission{
		Name:        request.Name,
		Description: request.Description,
		Resource:    request.Resource,
		Action:      request.Action,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Start transaction
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Save permission to database
		if err := tx.CreatePermission(ctx, permission); err != nil {
			return fmt.Errorf("failed to create permission: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := permission.ToResponse()
	return &response, nil
}

// GetPermissionByID retrieves a permission by ID
func (s *PermissionService) GetPermissionByID(ctx context.Context, id string) (*models.PermissionResponse, error) {
	// Parse UUID
	permissionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid permission ID: %w", err)
	}

	// Get permission
	permission, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := permission.ToResponse()
	return &response, nil
}

// GetAllPermissions retrieves all permissions
func (s *PermissionService) GetAllPermissions(ctx context.Context) ([]models.PermissionResponse, error) {
	// Get permissions
	permissions, err := s.permissionRepo.GetAll(ctx)
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

// GetPermissionsByResource retrieves all permissions for a specific resource
func (s *PermissionService) GetPermissionsByResource(ctx context.Context, resource string) ([]models.PermissionResponse, error) {
	// Get permissions
	permissions, err := s.permissionRepo.GetByResource(ctx, resource)
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

// UpdatePermission updates a permission
func (s *PermissionService) UpdatePermission(ctx context.Context, id string, request models.PermissionUpdateRequest) (*models.PermissionResponse, error) {
	// Parse UUID
	permissionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid permission ID: %w", err)
	}

	// Get existing permission
	permission, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}

	// Check for resource/action uniqueness if being updated
	if (request.Resource != "" && request.Resource != permission.Resource) ||
		(request.Action != "" && request.Action != permission.Action) {
		resourceToCheck := permission.Resource
		if request.Resource != "" {
			resourceToCheck = request.Resource
		}

		actionToCheck := permission.Action
		if request.Action != "" {
			actionToCheck = request.Action
		}

		existingPermission, err := s.permissionRepo.GetByResourceAction(ctx, resourceToCheck, actionToCheck)
		if err == nil && existingPermission != nil && existingPermission.ID != permission.ID {
			return nil, fmt.Errorf("permission already exists for this resource and action")
		}
	}

	// Update fields if provided
	if request.Name != "" {
		permission.Name = request.Name
	}
	if request.Description != "" {
		permission.Description = request.Description
	}
	if request.Resource != "" {
		permission.Resource = request.Resource
	}
	if request.Action != "" {
		permission.Action = request.Action
	}
	permission.UpdatedAt = time.Now()

	// Start transaction
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Update permission in database
		if err := tx.UpdatePermission(ctx, permission); err != nil {
			return fmt.Errorf("failed to update permission: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := permission.ToResponse()
	return &response, nil
}

// DeletePermission deletes a permission
func (s *PermissionService) DeletePermission(ctx context.Context, id string) error {
	// Parse UUID
	permissionID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid permission ID: %w", err)
	}

	// Delete permission
	return s.permissionRepo.Delete(ctx, permissionID)
}
