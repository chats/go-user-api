package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/chats/go-user-api/api/grpc/pb"
	grpcserver "github.com/chats/go-user-api/api/grpc/server"
	"github.com/chats/go-user-api/api/http/handlers"
	"github.com/chats/go-user-api/api/http/routes"
	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/cache"
	"github.com/chats/go-user-api/internal/database"
	"github.com/chats/go-user-api/internal/logger"
	"github.com/chats/go-user-api/internal/repositories"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const (
	serviceConnectRetries = 3
	serviceRetryInterval  = 2 * time.Second
	gracefulTimeout       = 15 * time.Second
)

func main() {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logger
	logger.InitLogger()

	log.Info().Msg("Starting service...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to database with retries
	var db *database.DB
	for i := 0; i < serviceConnectRetries; i++ {
		db, err = database.NewDB(cfg)
		if err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", i+1).Int("maxAttempts", serviceConnectRetries).
			Msg("Failed to connect to database, retrying...")
		time.Sleep(serviceRetryInterval)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database after multiple attempts")
	}
	defer db.Close()

	// Apply database migrations
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to apply database migrations")
	}

	// Initialize Redis cache with retries
	var redisClient *cache.RedisClient
	for i := 0; i < serviceConnectRetries; i++ {
		redisClient, err = cache.NewRedisClient(cfg)
		if err == nil {
			break
		}
		log.Warn().Err(err).Int("attempt", i+1).Int("maxAttempts", serviceConnectRetries).
			Msg("Failed to connect to Redis, retrying...")
		time.Sleep(serviceRetryInterval)
	}
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to Redis after multiple attempts, continuing without caching")
	}
	defer func() {
		if redisClient != nil {
			redisClient.Close()
		}
	}()

	// Initialize tracer
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracer, continuing without tracing")
	}
	defer func() {
		if tracer != nil {
			tracer.Close()
		}
	}()

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db, redisClient)
	roleRepo := repositories.NewRoleRepository(db, redisClient)
	permissionRepo := repositories.NewPermissionRepository(db, redisClient)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg)
	userService := services.NewUserService(userRepo, roleRepo)
	roleService := services.NewRoleService(roleRepo, permissionRepo)
	permissionService := services.NewPermissionService(permissionRepo)

	// Initialize HTTP handlers
	authHandler := handlers.NewAuthHandler(authService, userService, tracer)
	userHandler := handlers.NewUserHandler(userService, tracer)
	roleHandler := handlers.NewRoleHandler(roleService, tracer)
	permissionHandler := handlers.NewPermissionHandler(permissionService, tracer)

	// Initialize gRPC server
	userGRPCServer := grpcserver.NewUserGRPCServer(userService, authService, tracer, cfg)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "go-user-api",
		DisableStartupMessage: true,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			// Check for specific error types and set appropriate status codes
			if err == fiber.ErrBadRequest {
				code = fiber.StatusBadRequest
			} else if err == fiber.ErrNotFound {
				code = fiber.StatusNotFound
			} else if err == fiber.ErrMethodNotAllowed {
				code = fiber.StatusMethodNotAllowed
			}

			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		},
	})

	// Set up middleware
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
	}))
	app.Use(recover.New())
	app.Use(requestid.New())

	// CORS configuration with specific origins
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CorsAllowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		ExposeHeaders:    "Content-Length, Content-Type",
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Set up routes
	routes.SetupRoutes(app, cfg, authHandler, userHandler, roleHandler, permissionHandler, authService)

	// Create an explicit gRPC server variable for proper shutdown
	var grpcServer *grpc.Server

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start HTTP and gRPC servers
	var wg sync.WaitGroup
	wg.Add(2)

	// HTTP server
	httpServer := app
	httpServerCtx, httpServerCtxCancel := context.WithCancel(ctx)
	defer httpServerCtxCancel()

	go func() {
		defer wg.Done()

		log.Info().Str("port", cfg.ServerPort).Msg("Starting HTTP server")

		// Start the HTTP server
		go func() {
			if err := httpServer.Listen(":" + cfg.ServerPort); err != nil {
				log.Error().Err(err).Msg("HTTP server error")
			}
		}()

		// Wait for context cancellation
		<-httpServerCtx.Done()
		log.Info().Msg("HTTP server context canceled")
	}()

	// gRPC server
	go func() {
		defer wg.Done()

		listener, err := net.Listen("tcp", ":"+cfg.GrpcPort)
		if err != nil {
			log.Error().Err(err).Str("port", cfg.GrpcPort).Msg("Failed to listen for gRPC")
			return
		}

		// Set up gRPC server with options
		grpcServer = grpc.NewServer(
			grpc.MaxConcurrentStreams(100),
			grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		)
		pb.RegisterUserServiceServer(grpcServer, userGRPCServer)

		log.Info().Str("port", cfg.GrpcPort).Msg("Starting gRPC server")
		if err := grpcServer.Serve(listener); err != nil {
			log.Error().Err(err).Msg("gRPC server error")
		}
	}()

	// Wait for termination signal
	<-quit
	log.Info().Msg("Received shutdown signal, initiating graceful shutdown...")

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer shutdownCancel()

	// Cancel main context to stop background workers
	cancel()

	// Shutdown HTTP server (with timeout)
	httpShutdownDone := make(chan struct{})
	go func() {
		log.Info().Msg("Shutting down HTTP server...")
		// Handle different versions of Fiber
		shutdownErr := app.Shutdown()
		if shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("HTTP server shutdown error")
		} else {
			log.Info().Msg("HTTP server shutdown complete")
		}
		close(httpShutdownDone)
	}()

	// Gracefully stop gRPC server
	grpcShutdownDone := make(chan struct{})
	go func() {
		if grpcServer != nil {
			log.Info().Msg("Shutting down gRPC server...")

			// Use a separate goroutine to handle the graceful stop timeout
			done := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(done)
			}()

			// Wait for either graceful stop to complete or context to timeout
			select {
			case <-done:
				log.Info().Msg("gRPC server graceful shutdown complete")
			case <-shutdownCtx.Done():
				log.Warn().Msg("gRPC server graceful shutdown timed out, forcing stop")
				grpcServer.Stop()
			}
		}
		close(grpcShutdownDone)
	}()

	// Wait for all components to shut down with timeout handling
	select {
	case <-httpShutdownDone:
		log.Debug().Msg("HTTP server shutdown done")
	case <-shutdownCtx.Done():
		log.Warn().Msg("HTTP server shutdown timed out")
	}

	select {
	case <-grpcShutdownDone:
		log.Debug().Msg("gRPC server shutdown done")
	case <-shutdownCtx.Done():
		log.Warn().Msg("gRPC server shutdown timed out")
	}

	// Final cleanup and exit
	log.Info().Msg("All components shut down, service stopped")
}
