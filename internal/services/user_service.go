package services

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/repositories/transaction"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// UserService handles user-related operations
type UserService struct {
	userRepo  repositories.UserRepositoryInterface
	roleRepo  repositories.RoleRepositoryInterface
	txManager transaction.Manager[transaction.Repository]
}

// NewUserService creates a new user service
func NewUserService(
	userRepo repositories.UserRepositoryInterface,
	roleRepo repositories.RoleRepositoryInterface,
	txManager transaction.Manager[transaction.Repository],
) *UserService {
	return &UserService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		txManager: txManager,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, request models.UserCreateRequest) (*models.UserResponse, error) {
	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, request.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Create user object
	user := &models.User{
		Username:  request.Username,
		Email:     request.Email,
		FirstName: request.FirstName,
		LastName:  request.LastName,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Hash password
	if err := user.HashPassword(request.Password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Execute transaction with the unified transaction manager
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Save user to database
		if err := tx.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Assign roles if provided
		if len(request.RoleIDs) > 0 {
			roleIDs := make([]uuid.UUID, 0, len(request.RoleIDs))
			for _, roleIDStr := range request.RoleIDs {
				roleID, err := uuid.Parse(roleIDStr)
				if err != nil {
					return fmt.Errorf("invalid role ID: %w", err)
				}
				roleIDs = append(roleIDs, roleID)
			}

			if err := tx.AssignRolesToUser(ctx, user.ID, roleIDs); err != nil {
				return fmt.Errorf("failed to assign roles: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the updated user with roles
	updatedUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get updated user after creation")
		// Return the user without roles as fallback
		// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
		response := user.ToResponse()
		return &response, nil
	}

	response := updatedUser.ToResponse()
	return &response, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.UserResponse, error) {
	// Parse UUID
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := user.ToResponse()
	return &response, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.UserResponse, error) {
	// Get user
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := user.ToResponse()
	return &response, nil
}

// GetAllUsers retrieves all users with pagination
func (s *UserService) GetAllUsers(ctx context.Context, page, pageSize int) ([]models.UserResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get users
	users, err := s.userRepo.GetAll(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	totalCount, err := s.userRepo.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToResponse()
	}

	return userResponses, totalCount, nil
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, id string, request models.UserUpdateRequest) (*models.UserResponse, error) {
	// Parse UUID
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check for username uniqueness if username is being updated
	if request.Username != "" && request.Username != user.Username {
		existingUser, err := s.userRepo.GetByUsername(ctx, request.Username)
		if err == nil && existingUser != nil {
			return nil, fmt.Errorf("username already exists")
		}
	}

	// Update fields if provided
	if request.Username != "" {
		user.Username = request.Username
	}
	if request.Email != "" {
		user.Email = request.Email
	}
	if request.FirstName != "" {
		user.FirstName = request.FirstName
	}
	if request.LastName != "" {
		user.LastName = request.LastName
	}
	if request.IsActive != nil {
		user.IsActive = *request.IsActive
	}
	user.UpdatedAt = time.Now()

	// Start transaction
	err = s.txManager.ExecuteTx(ctx, func(tx transaction.Repository) error {
		// Update user in database
		if err := tx.UpdateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		if request.Password != "" {
			hashedPassword, err := utils.HashPassword(request.Password)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			if err := tx.UpdateUserPassword(ctx, user.ID, hashedPassword); err != nil {
				return fmt.Errorf("failed to update password: %w", err)
			}
		}
		if len(request.RoleIDs) > 0 {
			roleIDs := make([]uuid.UUID, 0, len(request.RoleIDs))
			for _, roleIDStr := range request.RoleIDs {
				roleID, err := uuid.Parse(roleIDStr)
				if err != nil {
					return fmt.Errorf("invalid role ID: %w", err)
				}
				roleIDs = append(roleIDs, roleID)
			}

			if err := tx.AssignRolesToUser(ctx, user.ID, roleIDs); err != nil {
				return fmt.Errorf("failed to assign roles: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the updated user with roles
	updatedUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get updated user after update")
		// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
		response := user.ToResponse()
		return &response, nil
	}

	// แก้ไขตรงนี้: สร้างตัวแปรก่อนแล้วค่อย return address ของตัวแปรนั้น
	response := updatedUser.ToResponse()
	return &response, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	// Parse UUID
	userID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Delete user
	return s.userRepo.Delete(ctx, userID)
}

// GetUserPermissions retrieves all permissions for a user
func (s *UserService) GetUserPermissions(ctx context.Context, id string) ([]models.PermissionResponse, error) {
	// Parse UUID
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get permissions
	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
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

// HasPermission checks if a user has a specific permission
func (s *UserService) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.userRepo.HasPermission(ctx, id, resource, action)
}
