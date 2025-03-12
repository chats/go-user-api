package middleware

import (
	"github.com/chats/go-user-api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// HasPermissionMiddleware creates a middleware that checks if user has the required permission
func HasPermissionMiddleware(authService *services.AuthService, resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context
		userID, ok := c.Locals("userID").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "User ID not found in token",
			})
		}

		// Check if user has the required permission
		hasPermission, err := authService.CheckPermission(c.Context(), userID, resource, action)
		if err != nil {
			log.Error().Err(err).
				Str("user_id", userID).
				Str("resource", resource).
				Str("action", action).
				Msg("Failed to check permission")

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to check permission",
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Access denied: insufficient permissions",
			})
		}

		return c.Next()
	}
}

// ResourceWriteAccessMiddleware creates a middleware that checks if user has write access to a resource
func ResourceWriteAccessMiddleware(authService *services.AuthService, resource string) fiber.Handler {
	return HasPermissionMiddleware(authService, resource, "write")
}

// ResourceReadAccessMiddleware creates a middleware that checks if user has read access to a resource
func ResourceReadAccessMiddleware(authService *services.AuthService, resource string) fiber.Handler {
	return HasPermissionMiddleware(authService, resource, "read")
}

// ResourceDeleteAccessMiddleware creates a middleware that checks if user has delete access to a resource
func ResourceDeleteAccessMiddleware(authService *services.AuthService, resource string) fiber.Handler {
	return HasPermissionMiddleware(authService, resource, "delete")
}
