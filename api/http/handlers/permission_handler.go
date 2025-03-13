package handlers

import (
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// PermissionHandler handles permission-related HTTP requests
type PermissionHandler struct {
	permissionService *services.PermissionService
	tracer            *tracing.Tracer
}

// NewPermissionHandler creates a new permission handler
func NewPermissionHandler(
	permissionService *services.PermissionService,
	tracer *tracing.Tracer,
) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		tracer:            tracer,
	}
}

// GetPermissions retrieves all permissions
func (h *PermissionHandler) GetPermissions(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "PermissionHandler.GetPermissions")
	defer span.End()

	// Get query parameters
	resource := c.Query("resource", "")

	var permissions []models.PermissionResponse
	var err error

	// Get permissions by resource if provided, otherwise get all
	if resource != "" {
		h.tracer.SetAttributes(ctx,
			attribute.String("resource", resource),
		)

		permissions, err = h.permissionService.GetPermissionsByResource(ctx, resource)
	} else {
		permissions, err = h.permissionService.GetAllPermissions(ctx)
	}

	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("resource", resource).
			Msg("Failed to get permissions")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get permissions",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    permissions,
	})
}

// GetPermission retrieves a permission by ID
func (h *PermissionHandler) GetPermission(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "PermissionHandler.GetPermission")
	defer span.End()

	// Get permission ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Permission ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("permission_id", id),
	)

	// Get permission
	permission, err := h.permissionService.GetPermissionByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("permission_id", id).
			Msg("Failed to get permission")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Permission not found",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    permission,
	})
}

// CreatePermission creates a new permission
func (h *PermissionHandler) CreatePermission(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "PermissionHandler.CreatePermission")
	defer span.End()

	// Parse request body
	var request models.PermissionCreateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("permission_name", request.Name),
		attribute.String("resource", request.Resource),
		attribute.String("action", request.Action),
	)

	// Validate request
	if request.Name == "" || request.Resource == "" || request.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Permission name, resource, and action are required",
		})
	}

	// Create permission
	permission, err := h.permissionService.CreatePermission(ctx, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("permission_name", request.Name).
			Str("resource", request.Resource).
			Str("action", request.Action).
			Msg("Failed to create permission")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create permission",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("permission_name", request.Name).
		Str("permission_id", permission.ID.String()).
		Msg("Permission created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    permission,
	})
}

// UpdatePermission updates a permission
func (h *PermissionHandler) UpdatePermission(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "PermissionHandler.UpdatePermission")
	defer span.End()

	// Get permission ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Permission ID is required",
		})
	}

	// Parse request body
	var request models.PermissionUpdateRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("permission_id", id),
	)

	// Update permission
	permission, err := h.permissionService.UpdatePermission(ctx, id, request)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("permission_id", id).
			Msg("Failed to update permission")

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update permission",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("permission_id", id).
		Msg("Permission updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    permission,
	})
}

// DeletePermission deletes a permission
func (h *PermissionHandler) DeletePermission(c *fiber.Ctx) error {
	ctx, span := h.tracer.StartSpan(c.Context(), "PermissionHandler.DeletePermission")
	defer span.End()

	// Get permission ID from path
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Permission ID is required",
		})
	}

	h.tracer.SetAttributes(ctx,
		attribute.String("permission_id", id),
	)

	// Get the permission first for logging
	permission, err := h.permissionService.GetPermissionByID(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("permission_id", id).
			Msg("Permission not found for deletion")

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Permission not found",
			"error":   err.Error(),
		})
	}

	// Delete permission
	err = h.permissionService.DeletePermission(ctx, id)
	if err != nil {
		h.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("permission_id", id).
			Msg("Failed to delete permission")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete permission",
			"error":   err.Error(),
		})
	}

	// Log activity
	adminID, _ := c.Locals("userID").(string)
	log.Info().
		Str("admin_id", adminID).
		Str("permission_id", id).
		Str("permission_name", permission.Name).
		Msg("Permission deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Permission deleted successfully",
	})
}
