package handlers

import (
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// RoleHandler handles role-related HTTP requests
type RoleHandler struct {
	roleService *services.RoleService
	tracer      *tracing.Tracer
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(
	roleService *services.RoleService,
	tracer *tracing.Tracer,
) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
		tracer:      tracer,
	}
}

// GetRoles retrieves all roles
func (h *RoleHandler) GetRoles(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.GetRoles")
	defer span.End()

	// Get roles
	roles, err := h.roleService.GetAllRoles(ctx)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).Msg("Failed to get roles")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get roles",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    roles,
	})
}

// GetRole retrieves a role by ID
func (h *RoleHandler) GetRole(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.GetRole")
	defer span.End()

	// Get role ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Role ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("role_id", id),
	)

	// Get role
	role, err := h.roleService.GetRoleByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Failed to get role")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Role not found",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    role,
	})
}

// CreateRole creates a new role
func (h *RoleHandler) CreateRole(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.CreateRole")
	defer span.End()

	// Parse request body
	var request models.RoleCreateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("role_name", request.Name),
	)

	// Validate request
	if request.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Role name is required",
		})
	}

	// Create role
	role, err := h.roleService.CreateRole(ctx, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_name", request.Name).
			Msg("Failed to create role")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create role",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("role_name", request.Name).
		Str("role_id", role.ID.String()).
		Msg("Role created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    role,
	})
}

// UpdateRole updates a role
func (h *RoleHandler) UpdateRole(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.UpdateRole")
	defer span.End()

	// Get role ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Role ID is required",
		})
	}

	// Parse request body
	var request models.RoleUpdateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("role_id", id),
	)

	// Update role
	role, err := h.roleService.UpdateRole(ctx, id, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Failed to update role")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update role",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("role_id", id).
		Msg("Role updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    role,
	})
}

// DeleteRole deletes a role
func (h *RoleHandler) DeleteRole(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.DeleteRole")
	defer span.End()

	// Get role ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Role ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("role_id", id),
	)

	// Get the role first for logging
	role, err := h.roleService.GetRoleByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Role not found for deletion")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Role not found",
			"error":   err.Error(),
		})
	}

	// Delete role
	err = h.roleService.DeleteRole(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Failed to delete role")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete role",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("role_id", id).
		Str("role_name", role.Name).
		Msg("Role deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Role deleted successfully",
	})
}

// GetRolePermissions retrieves permissions for a role
func (h *RoleHandler) GetRolePermissions(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "RoleHandler.GetRolePermissions")
	defer span.End()

	// Get role ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Role ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("role_id", id),
	)

	// Check if role exists
	_, err := h.roleService.GetRoleByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Role not found for permissions lookup")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Role not found",
			"error":   err.Error(),
		})
	}

	// Get role permissions
	permissions, err := h.roleService.GetRolePermissions(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("role_id", id).
			Msg("Failed to get role permissions")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get role permissions",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    permissions,
	})
}
