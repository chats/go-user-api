package middleware

import (
	"fmt"
	"strings"

	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// JWTAuthMiddleware creates a middleware that validates JWT tokens
func JWTAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Missing authorization header",
			})
		}

		// Extract token
		tokenString, err := utils.ExtractBearerToken(authHeader)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}

		// Parse and verify token
		claims, err := utils.ParseJWT(tokenString, cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}

		// Store user information in context
		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("roles", claims.Roles)

		// Generate request ID if not exists
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}
		c.Locals("requestID", requestID)

		log.Debug().
			Str("user_id", claims.UserID).
			Str("username", claims.Username).
			Strs("roles", claims.Roles).
			Str("request_id", requestID).
			Str("path", c.Path()).
			Str("method", c.Method()).
			Msg("User authenticated")

		return c.Next()
	}
}

// HasRoleMiddleware creates a middleware that checks if user has at least one of the required roles
func HasRoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get roles from context
		rolesInterface := c.Locals("roles")
		if rolesInterface == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "No roles found in token",
			})
		}

		roles, ok := rolesInterface.([]string)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to parse roles",
			})
		}

		// Check if user has any of the allowed roles
		for _, allowedRole := range allowedRoles {
			for _, userRole := range roles {
				if userRole == allowedRole {
					return c.Next()
				}
			}
		}

		// User does not have any of the required roles
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Access denied: requires one of these roles: %s", strings.Join(allowedRoles, ", ")),
		})
	}
}

// AdminOnlyMiddleware creates a middleware that restricts access to admin users only
func AdminOnlyMiddleware() fiber.Handler {
	return HasRoleMiddleware("admin")
}
