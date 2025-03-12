package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chats/go-user-api/api/grpc/pb"
	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/messaging/kafka"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/chats/go-user-api/internal/utils"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserGRPCServer implements the UserService gRPC service
type UserGRPCServer struct {
	pb.UnimplementedUserServiceServer
	userService   *services.UserService
	authService   *services.AuthService
	kafkaProducer *kafka.Producer
	tracer        *tracing.Tracer
	config        *config.Config
}

// NewUserGRPCServer creates a new user gRPC server
func NewUserGRPCServer(
	userService *services.UserService,
	authService *services.AuthService,
	kafkaProducer *kafka.Producer,
	tracer *tracing.Tracer,
	config *config.Config,
) *UserGRPCServer {
	return &UserGRPCServer{
		userService:   userService,
		authService:   authService,
		kafkaProducer: kafkaProducer,
		tracer:        tracer,
		config:        config,
	}
}

// GetUser retrieves a user profile by ID
func (s *UserGRPCServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserProfile, error) {
	ctx, span := s.tracer.StartSpan(ctx, "UserGRPCServer.GetUser")
	defer span.End()

	s.tracer.SetAttributes(ctx,
		attribute.String("user_id", req.UserId),
	)

	// Get user
	user, err := s.userService.GetUserByID(ctx, req.UserId)
	if err != nil {
		s.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", req.UserId).
			Msg("gRPC: Failed to get user")

		return nil, status.Errorf(codes.NotFound, "User not found: %v", err)
	}

	// Convert to protobuf message
	createdAt := &timestamp.Timestamp{
		Seconds: user.CreatedAt.Unix(),
		Nanos:   int32(user.CreatedAt.Nanosecond()),
	}

	updatedAt := &timestamp.Timestamp{
		Seconds: user.UpdatedAt.Unix(),
		Nanos:   int32(user.UpdatedAt.Nanosecond()),
	}

	// Convert roles
	roles := make([]*pb.Role, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = &pb.Role{
			Id:          role.ID.String(),
			Name:        role.Name,
			Description: role.Description,
		}
	}

	// Log activity
	s.kafkaProducer.LogActivity(ctx, "", "grpc", "get_user", map[string]interface{}{
		"target_user_id": req.UserId,
	})

	return &pb.UserProfile{
		Id:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Roles:     roles,
	}, nil
}

// GetUserPermissions retrieves all permissions for a user
func (s *UserGRPCServer) GetUserPermissions(ctx context.Context, req *pb.GetUserRequest) (*pb.UserPermissionsResponse, error) {
	ctx, span := s.tracer.StartSpan(ctx, "UserGRPCServer.GetUserPermissions")
	defer span.End()

	s.tracer.SetAttributes(ctx,
		attribute.String("user_id", req.UserId),
	)

	// Check if user exists
	_, err := s.userService.GetUserByID(ctx, req.UserId)
	if err != nil {
		s.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", req.UserId).
			Msg("gRPC: User not found for permissions lookup")

		return nil, status.Errorf(codes.NotFound, "User not found: %v", err)
	}

	// Get user permissions
	permissions, err := s.userService.GetUserPermissions(ctx, req.UserId)
	if err != nil {
		s.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", req.UserId).
			Msg("gRPC: Failed to get user permissions")

		return nil, status.Errorf(codes.Internal, "Failed to get user permissions: %v", err)
	}

	// Convert to protobuf message
	protoPermissions := make([]*pb.Permission, len(permissions))
	for i, perm := range permissions {
		protoPermissions[i] = &pb.Permission{
			Id:          perm.ID.String(),
			Name:        perm.Name,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Description: perm.Description,
		}
	}

	// Log activity
	s.kafkaProducer.LogActivity(ctx, "", "grpc", "get_user_permissions", map[string]interface{}{
		"target_user_id": req.UserId,
	})

	return &pb.UserPermissionsResponse{
		Permissions: protoPermissions,
	}, nil
}

// ValidateToken validates a JWT token and returns user info
func (s *UserGRPCServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.TokenValidationResponse, error) {
	ctx, span := s.tracer.StartSpan(ctx, "UserGRPCServer.ValidateToken")
	defer span.End()

	// Parse and verify the token
	claims, err := utils.ParseJWT(req.Token, s.config)
	if err != nil {
		s.tracer.RecordError(ctx, err)

		log.Warn().Err(err).
			Msg("gRPC: Invalid token")

		return &pb.TokenValidationResponse{
			IsValid: false,
			Error: &pb.Error{
				Code:    "invalid_token",
				Message: err.Error(),
			},
		}, nil
	}

	// Get expiration time
	expTime := time.Unix(claims.ExpiresAt.Unix(), 0)
	expProto := &timestamp.Timestamp{
		Seconds: expTime.Unix(),
		Nanos:   int32(expTime.Nanosecond()),
	}

	s.tracer.SetAttributes(ctx,
		attribute.String("user_id", claims.UserID),
		attribute.String("username", claims.Username),
	)

	// Log activity
	s.kafkaProducer.LogActivity(ctx, claims.UserID, "grpc", "validate_token", nil)

	// Return validation result
	return &pb.TokenValidationResponse{
		IsValid:   true,
		UserId:    claims.UserID,
		Username:  claims.Username,
		Roles:     claims.Roles,
		ExpiresAt: expProto,
		Error:     nil,
	}, nil
}

// HasPermission checks if a user has a specific permission
func (s *UserGRPCServer) HasPermission(ctx context.Context, req *pb.HasPermissionRequest) (*pb.HasPermissionResponse, error) {
	ctx, span := s.tracer.StartSpan(ctx, "UserGRPCServer.HasPermission")
	defer span.End()

	s.tracer.SetAttributes(ctx,
		attribute.String("user_id", req.UserId),
		attribute.String("resource", req.Resource),
		attribute.String("action", req.Action),
	)

	// Check permission
	hasPermission, err := s.authService.CheckPermission(ctx, req.UserId, req.Resource, req.Action)
	if err != nil {
		s.tracer.RecordError(ctx, err)

		log.Error().Err(err).
			Str("user_id", req.UserId).
			Str("resource", req.Resource).
			Str("action", req.Action).
			Msg("gRPC: Failed to check permission")

		// If it's a "not found" error
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return &pb.HasPermissionResponse{
				HasPermission: false,
				Error: &pb.Error{
					Code:    "user_not_found",
					Message: fmt.Sprintf("User not found: %v", err.Error()),
				},
			}, nil
		}

		return &pb.HasPermissionResponse{
			HasPermission: false,
			Error: &pb.Error{
				Code:    "internal_error",
				Message: fmt.Sprintf("Failed to check permission: %v", err.Error()),
			},
		}, nil
	}

	// Log activity
	s.kafkaProducer.LogActivity(ctx, req.UserId, "grpc", "check_permission", map[string]interface{}{
		"resource":       req.Resource,
		"action":         req.Action,
		"has_permission": hasPermission,
	})

	return &pb.HasPermissionResponse{
		HasPermission: hasPermission,
		Error:         nil,
	}, nil
}
