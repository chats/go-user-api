package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"net"

	"github.com/chats/go-user-api/api/grpc/pb"
	"github.com/chats/go-user-api/api/grpc/server"
	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/mocks"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// setupGRPCServer sets up a test gRPC server with mocked services
func setupGRPCServer() (*bufconn.Listener, *mocks.MockUserRepository, *mocks.MockRoleRepository, *mocks.MockPermissionRepository) {
	// Create mocks
	mockUserRepo := new(mocks.MockUserRepository)
	mockRoleRepo := new(mocks.MockRoleRepository)
	mockPermRepo := new(mocks.MockPermissionRepository)

	// Create config for testing
	cfg := &config.Config{
		JWTSecret:       "test-secret-key",
		JWTExpireMinute: 60,
	}

	// Create mock tracer
	tracer, _ := tracing.NewTracer(cfg)

	// Create services
	authService := services.NewAuthService(mockUserRepo, cfg)
	userService := services.NewUserService(mockUserRepo, mockRoleRepo)

	// Create gRPC server
	grpcServer := server.NewUserGRPCServer(userService, authService, tracer, cfg)

	// Set up gRPC server with bufconn listener
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, grpcServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	return lis, mockUserRepo, mockRoleRepo, mockPermRepo
}

// bufDialer is a helper function to allow connecting to the bufconn listener
func bufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, url string) (net.Conn, error) {
		return lis.Dial()
	}
}

// setupGRPCClient sets up a gRPC client for testing
func setupGRPCClient(lis *bufconn.Listener) (pb.UserServiceClient, *grpc.ClientConn, error) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer(lis)), grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewUserServiceClient(conn)
	return client, conn, nil
}

func TestGRPC_GetUser(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user
	userId := uuid.New()
	user := &models.User{
		ID:        userId,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles: []models.Role{
			{ID: uuid.New(), Name: "admin", Description: "Admin role"},
		},
	}

	// Setup mock behavior
	mockUserRepo.On("GetByID", mock.Anything, userId).Return(user, nil)

	// Create request
	req := &pb.GetUserRequest{
		UserId: userId.String(),
	}

	// Call GetUser
	resp, err := client.GetUser(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, userId.String(), resp.Id)
	assert.Equal(t, user.Username, resp.Username)
	assert.Equal(t, user.Email, resp.Email)
	assert.Equal(t, user.FirstName, resp.FirstName)
	assert.Equal(t, user.LastName, resp.LastName)
	assert.Equal(t, user.IsActive, resp.IsActive)
	assert.NotNil(t, resp.CreatedAt)
	assert.NotNil(t, resp.UpdatedAt)
	assert.Len(t, resp.Roles, 1)
	assert.Equal(t, "admin", resp.Roles[0].Name)
	assert.Equal(t, "Admin role", resp.Roles[0].Description)

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGRPC_GetUserNotFound(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user ID (not found)
	userId := uuid.New()

	// Setup mock behavior
	mockUserRepo.On("GetByID", mock.Anything, userId).Return(nil, errors.New("user not found"))

	// Create request
	req := &pb.GetUserRequest{
		UserId: userId.String(),
	}

	// Call GetUser
	resp, err := client.GetUser(context.Background(), req)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, resp)

	// Check error type
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Contains(t, st.Message(), "User not found")

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGRPC_GetUserPermissions(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user ID
	userId := uuid.New()

	// Test user
	user := &models.User{
		ID:       userId,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	// Test permissions
	permissions := []models.Permission{
		{
			ID:          uuid.New(),
			Name:        "user:read",
			Description: "Read user information",
			Resource:    "user",
			Action:      "read",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "user:write",
			Description: "Write user information",
			Resource:    "user",
			Action:      "write",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Setup mock behaviors
	mockUserRepo.On("GetByID", mock.Anything, userId).Return(user, nil)
	mockUserRepo.On("GetUserPermissions", mock.Anything, userId).Return(permissions, nil)

	// Create request
	req := &pb.GetUserRequest{
		UserId: userId.String(),
	}

	// Call GetUserPermissions
	resp, err := client.GetUserPermissions(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.Len(t, resp.Permissions, 2)

	perm1 := resp.Permissions[0]
	assert.Equal(t, permissions[0].ID.String(), perm1.Id)
	assert.Equal(t, "user:read", perm1.Name)
	assert.Equal(t, "Read user information", perm1.Description)
	assert.Equal(t, "user", perm1.Resource)
	assert.Equal(t, "read", perm1.Action)

	perm2 := resp.Permissions[1]
	assert.Equal(t, permissions[1].ID.String(), perm2.Id)
	assert.Equal(t, "user:write", perm2.Name)
	assert.Equal(t, "Write user information", perm2.Description)
	assert.Equal(t, "user", perm2.Resource)
	assert.Equal(t, "write", perm2.Action)

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGRPC_ValidateToken(t *testing.T) {
	// Set up server and client
	lis, _, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Create a valid token for testing
	// This token is signed with the test secret key "test-secret-key"
	// Expires in 2033 (far in the future)
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"

	// Create request
	req := &pb.ValidateTokenRequest{
		Token: validToken,
	}

	// Call ValidateToken
	resp, err := client.ValidateToken(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.True(t, resp.IsValid)
	assert.Equal(t, "admin", resp.UserId)
	assert.Equal(t, "admin", resp.Username)
	assert.Contains(t, resp.Roles, "admin")
	assert.NotNil(t, resp.ExpiresAt)
	assert.Nil(t, resp.Error)
}

func TestGRPC_ValidateInvalidToken(t *testing.T) {
	// Set up server and client
	lis, _, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Create an invalid token for testing
	invalidToken := "invalid.token.here"

	// Create request
	req := &pb.ValidateTokenRequest{
		Token: invalidToken,
	}

	// Call ValidateToken
	resp, err := client.ValidateToken(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.False(t, resp.IsValid)
	assert.Empty(t, resp.UserId)
	assert.Empty(t, resp.Username)
	assert.Empty(t, resp.Roles)
	assert.Nil(t, resp.ExpiresAt)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "invalid_token", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "failed to parse token")
}

func TestGRPC_HasPermission(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user ID
	userId := uuid.New()

	// Setup mock behavior
	mockUserRepo.On("HasPermission", mock.Anything, userId, "user", "read").Return(true, nil)

	// Create request
	req := &pb.HasPermissionRequest{
		UserId:   userId.String(),
		Resource: "user",
		Action:   "read",
	}

	// Call HasPermission
	resp, err := client.HasPermission(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.True(t, resp.HasPermission)
	assert.Nil(t, resp.Error)

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGRPC_HasNoPermission(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user ID
	userId := uuid.New()

	// Setup mock behavior
	mockUserRepo.On("HasPermission", mock.Anything, userId, "user", "delete").Return(false, nil)

	// Create request
	req := &pb.HasPermissionRequest{
		UserId:   userId.String(),
		Resource: "user",
		Action:   "delete",
	}

	// Call HasPermission
	resp, err := client.HasPermission(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.False(t, resp.HasPermission)
	assert.Nil(t, resp.Error)

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGRPC_HasPermissionError(t *testing.T) {
	// Set up server and client
	lis, mockUserRepo, _, _ := setupGRPCServer()
	client, conn, err := setupGRPCClient(lis)
	require.NoError(t, err)
	defer conn.Close()

	// Test user ID
	userId := uuid.New()

	// Setup mock behavior
	mockUserRepo.On("HasPermission", mock.Anything, userId, "user", "write").
		Return(false, errors.New("user not found"))

	// Create request
	req := &pb.HasPermissionRequest{
		UserId:   userId.String(),
		Resource: "user",
		Action:   "write",
	}

	// Call HasPermission
	resp, err := client.HasPermission(context.Background(), req)
	require.NoError(t, err)

	// Assertions
	assert.NotNil(t, resp)
	assert.False(t, resp.HasPermission)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "user_not_found", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "User not found")

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}
