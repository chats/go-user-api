package services

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/google/uuid"
)

// AuthService handles authentication-related operations
type AuthService struct {
	userRepo repositories.UserRepositoryInterface
	config   *config.Config
}

func NewAuthService(userRepo repositories.UserRepositoryInterface, config *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		config:   config,
	}
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, request models.LoginRequest) (*models.LoginResponse, error) {
	// Find user by username
	user, err := s.userRepo.GetByUsername(ctx, request.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Verify password
	if !user.CheckPassword(request.Password) {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Extract role names for JWT
	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.Name
	}

	// Generate JWT token
	tokenString, expirationTime, err := s.GenerateToken(user.ID, user.Username, roleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create response
	response := &models.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "bearer",
		ExpiresIn:   int(time.Until(expirationTime).Seconds()),
		User:        user.ToResponse(),
	}

	return response, nil
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(userID uuid.UUID, username string, roles []string) (string, time.Time, error) {
	return utils.GenerateJWT(userID, username, roles, s.config)
}

// VerifyToken verifies a JWT token and returns the claims
func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*utils.JWTClaims, error) {
	// Parse and verify the token
	claims, err := utils.ParseJWT(tokenString, s.config)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID string, currentPassword, newPassword string) error {
	// Parse user ID
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if !user.CheckPassword(currentPassword) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	err = s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ResetPassword resets a user's password (admin function)
func (s *AuthService) ResetPassword(ctx context.Context, userID string) (string, error) {
	// Parse user ID
	id, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Generate random password
	newPassword, err := utils.GenerateRandomPassword(12)
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	err = s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to update password: %w", err)
	}

	return newPassword, nil
}

// CheckPermission checks if a user has a specific permission
func (s *AuthService) CheckPermission(ctx context.Context, userID string, resource, action string) (bool, error) {
	// Parse user ID
	id, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check permission
	hasPermission, err := s.userRepo.HasPermission(ctx, id, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return hasPermission, nil
}
