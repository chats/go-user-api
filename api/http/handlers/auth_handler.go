package handlers

import (
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService
	tracer      *tracing.Tracer
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	authService *services.AuthService,
	userService *services.UserService,
	tracer *tracing.Tracer,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		tracer:      tracer,
	}
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "AuthHandler.Login")
	defer span.End()

	// Parse request body
	var request models.LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("username", request.Username),
	)

	// Validate request
	if request.Username == "" || request.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Username and password are required",
		})
	}

	// Authenticate user
	response, err := h.authService.Login(ctx, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("username", request.Username).
			Msg("Login failed")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid username or password",
		})
	}

	// Log successful login
	log.Info().
		Str("username", request.Username).
		Str("user_id", response.User.ID.String()).
		Msg("User logged in successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "AuthHandler.ChangePassword")
	defer span.End()

	// Get user ID from context
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User ID not found in token",
		})
	}

	// Parse request body
	var request struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	// Validate request
	if request.CurrentPassword == "" || request.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Current password and new password are required",
		})
	}

	if len(request.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "New password must be at least 8 characters long",
		})
	}

	// Change password
	err := h.authService.ChangePassword(ctx, userID, request.CurrentPassword, request.NewPassword)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Password change failed")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	log.Info().
		Str("user_id", userID).
		Msg("Password changed successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Password changed successfully",
	})
}

// ResetPassword handles password reset (admin only)
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "AuthHandler.ResetPassword")
	defer span.End()

	// Get admin ID from context
	adminID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User ID not found in token",
		})
	}

	// Parse request body
	var request struct {
		UserID string `json:"user_id" validate:"required"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	// Validate request
	if request.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	// Reset password
	newPassword, err := h.authService.ResetPassword(ctx, request.UserID)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("admin_id", adminID).
			Str("user_id", request.UserID).
			Msg("Password reset failed")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// Get user to send email
	/*
		user, err := h.userService.GetUserByID(ctx, request.UserID)
		if err == nil && user != nil {
			// Send password reset email
			go func() {
				mailErr := h.mailService.SendPasswordResetEmail(
					context.Background(),
					user.Email,
					user.Username,
					newPassword,
				)
				if mailErr != nil {
					log.Error().Err(mailErr).
						Str("user_id", user.ID.String()).
						Msg("Failed to send password reset email")
				}
			}()
		}*/

	log.Info().
		Str("admin_id", adminID).
		Str("user_id", request.UserID).
		Msg("Password reset successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"message":      "Password reset successfully",
		"new_password": newPassword,
	})
}
