package handlers

import (
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
	tracer      *tracing.Tracer
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userService *services.UserService,
	tracer *tracing.Tracer,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		tracer:      tracer,
	}
}

// GetUsers retrieves all users with pagination
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.GetUsers")
	defer span.End()

	// Get query parameters
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)

	// Get users
	users, totalCount, err := h.userService.GetAllUsers(ctx, page, pageSize)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Int("page", page).
			Int("page_size", pageSize).
			Msg("Failed to get users")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get users",
			"error":   err.Error(),
		})
	}

	// Calculate pagination info
	totalPages := (totalCount + pageSize - 1) / pageSize
	hasNextPage := page < totalPages
	hasPrevPage := page > 1

	h.tracer.SetAttributes(ctx,
		attribute.Int("total_count", totalCount),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
		attribute.Int("total_pages", totalPages),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users":        users,
			"total_count":  totalCount,
			"page":         page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"has_next":     hasNextPage,
			"has_previous": hasPrevPage,
		},
	})
}

// GetUser retrieves a user by ID
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.GetUser")
	defer span.End()

	// Get user ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("user_id", id),
	)

	// Get user
	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("Failed to get user")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// GetMe retrieves the current user information
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.GetMe")
	defer span.End()

	// Get user ID from context
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User ID not found in token",
		})
	}

	// Get user
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Failed to get current user")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get user information",
			"error":   err.Error(),
		})
	}

	// Get user permissions
	permissions, err := h.userService.GetUserPermissions(ctx, userID)
	if err != nil {
		log.Warn().Err(err).
			Str("user_id", userID).
			Msg("Failed to get user permissions")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user":        user,
			"permissions": permissions,
		},
	})
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.CreateUser")
	defer span.End()

	// Parse request body
	var request models.UserCreateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("username", request.Username),
		attribute.String("email", request.Email),
	)

	// Validate request
	if request.Username == "" || request.Email == "" || request.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Username, email, and password are required",
		})
	}

	// Create user
	user, err := h.userService.CreateUser(ctx, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("username", request.Username).
			Str("email", request.Email).
			Msg("Failed to create user")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create user",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("username", request.Username).
		Str("user_id", user.ID.String()).
		Msg("User created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.UpdateUser")
	defer span.End()

	// Get user ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	// Parse request body
	var request models.UserUpdateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("user_id", id),
	)

	// Update user
	user, err := h.userService.UpdateUser(ctx, id, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("Failed to update user")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update user",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("user_id", id).
		Msg("User updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.DeleteUser")
	defer span.End()

	// Get user ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("user_id", id),
	)

	// Get the user first for logging
	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("User not found for deletion")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
			"error":   err.Error(),
		})
	}

	// Delete user
	err = h.userService.DeleteUser(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("Failed to delete user")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete user",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("user_id", id).
		Str("username", user.Username).
		Msg("User deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}

// GetUserPermissions retrieves permissions for a user
func (h *UserHandler) GetUserPermissions(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "UserHandler.GetUserPermissions")
	defer span.End()

	// Get user ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("user_id", id),
	)

	// Check if user exists
	_, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("User not found for permissions lookup")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
			"error":   err.Error(),
		})
	}

	// Get user permissions
	permissions, err := h.userService.GetUserPermissions(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", id).
			Msg("Failed to get user permissions")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get user permissions",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    permissions,
	})
}
