package routes

import (
	"github.com/chats/go-user-api/api/http/handlers"
	"github.com/chats/go-user-api/api/http/middleware"
	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up all HTTP routes for the application
func SetupRoutes(
	app *fiber.App,
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	roleHandler *handlers.RoleHandler,
	permissionHandler *handlers.PermissionHandler,
	authService *services.AuthService,
) {
	// Health check
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Public routes
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	// Protected routes
	protected := api.Group("", middleware.JWTAuthMiddleware(cfg))

	// Auth routes
	protectedAuth := protected.Group("/auth")
	protectedAuth.Post("/change-password", authHandler.ChangePassword)
	protectedAuth.Post("/reset-password", middleware.AdminOnlyMiddleware(), authHandler.ResetPassword)

	// User routes
	users := protected.Group("/users")
	users.Get("/", middleware.ResourceReadAccessMiddleware(authService, "user"), userHandler.GetUsers)
	users.Post("/", middleware.ResourceWriteAccessMiddleware(authService, "user"), userHandler.CreateUser)
	users.Get("/me", userHandler.GetMe)
	users.Get("/:id", middleware.ResourceReadAccessMiddleware(authService, "user"), userHandler.GetUser)
	users.Put("/:id", middleware.ResourceWriteAccessMiddleware(authService, "user"), userHandler.UpdateUser)
	users.Delete("/:id", middleware.ResourceDeleteAccessMiddleware(authService, "user"), userHandler.DeleteUser)
	users.Get("/:id/permissions", middleware.ResourceReadAccessMiddleware(authService, "user"), userHandler.GetUserPermissions)

	// Role routes
	roles := protected.Group("/roles")
	roles.Get("/", middleware.ResourceReadAccessMiddleware(authService, "role"), roleHandler.GetRoles)
	roles.Post("/", middleware.ResourceWriteAccessMiddleware(authService, "role"), roleHandler.CreateRole)
	roles.Get("/:id", middleware.ResourceReadAccessMiddleware(authService, "role"), roleHandler.GetRole)
	roles.Put("/:id", middleware.ResourceWriteAccessMiddleware(authService, "role"), roleHandler.UpdateRole)
	roles.Delete("/:id", middleware.ResourceDeleteAccessMiddleware(authService, "role"), roleHandler.DeleteRole)
	roles.Get("/:id/permissions", middleware.ResourceReadAccessMiddleware(authService, "role"), roleHandler.GetRolePermissions)

	// Permission routes
	permissions := protected.Group("/permissions")
	permissions.Get("/", middleware.ResourceReadAccessMiddleware(authService, "permission"), permissionHandler.GetPermissions)
	permissions.Post("/", middleware.ResourceWriteAccessMiddleware(authService, "permission"), permissionHandler.CreatePermission)
	permissions.Get("/:id", middleware.ResourceReadAccessMiddleware(authService, "permission"), permissionHandler.GetPermission)
	permissions.Put("/:id", middleware.ResourceWriteAccessMiddleware(authService, "permission"), permissionHandler.UpdatePermission)
	permissions.Delete("/:id", middleware.ResourceDeleteAccessMiddleware(authService, "permission"), permissionHandler.DeletePermission)
}
