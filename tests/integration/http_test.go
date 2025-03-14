package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chats/go-user-api/api/http/handlers"
	"github.com/chats/go-user-api/api/http/routes"
	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/mocks"
	"github.com/chats/go-user-api/internal/models"
	"github.com/chats/go-user-api/internal/services"
	"github.com/chats/go-user-api/internal/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupTestApp creates a test Fiber application with mocked services
func setupTestApp() (*fiber.App, *mocks.MockUserRepository, *mocks.MockRoleRepository, *mocks.MockPermissionRepository) {
	// Create mocks
	mockUserRepo := new(mocks.MockUserRepository)
	mockRoleRepo := new(mocks.MockRoleRepository)
	mockPermRepo := new(mocks.MockPermissionRepository)

	// Create config for testing
	cfg := &config.Config{
		JWTSecret:       "your-super-secret-key-here", // This matches what's in .env.example
		JWTExpireMinute: 60,
	}

	// Create mock tracer
	tracer, _ := tracing.NewTracer(cfg)

	// Create services
	authService := services.NewAuthService(mockUserRepo, cfg)
	userService := services.NewUserService(mockUserRepo, mockRoleRepo)
	roleService := services.NewRoleService(mockRoleRepo, mockPermRepo)
	permissionService := services.NewPermissionService(mockPermRepo)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, userService, tracer)
	userHandler := handlers.NewUserHandler(userService, tracer)
	roleHandler := handlers.NewRoleHandler(roleService, tracer)
	permissionHandler := handlers.NewPermissionHandler(permissionService, tracer)

	// Create Fiber app
	app := fiber.New()

	// Set up routes
	routes.SetupRoutes(app, cfg, authHandler, userHandler, roleHandler, permissionHandler, authService)

	return app, mockUserRepo, mockRoleRepo, mockPermRepo
}

// Helper function to make HTTP requests and parse responses
func makeRequest(app *fiber.App, method, path string, body interface{}, token string) (*http.Response, map[string]interface{}, error) {
	// Convert body to JSON
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
	}

	// Create request
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Perform request
	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}

	// Parse response body if it exists
	var respBody map[string]interface{}
	if resp.Body != nil {
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			if err != io.EOF { // Add this check to handle empty response bodies
				return resp, nil, err
			}
			// If we get EOF, return an empty map instead of nil
			respBody = map[string]interface{}{}
		}
	}

	return resp, respBody, nil
}

func TestAuthLogin_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user
	userID := uuid.New()
	hashedPassword := "$2a$10$FPS/DKJWlcHvU1fJuDEYDO0IXNoXQw./hCBlh90AogplwklD7PylC" // bcrypt hash for "adminpassword"
	user := &models.User{
		ID:        userID,
		Username:  "admin",
		Email:     "admin@example.com",
		Password:  hashedPassword,
		FirstName: "Admin",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles: []models.Role{
			{ID: uuid.New(), Name: "admin"},
		},
	}

	// Setup mock behavior
	mockUserRepo.On("GetByUsername", mock.Anything, "admin").Return(user, nil)

	// Test successful login
	loginReq := map[string]interface{}{
		"username": "admin",
		"password": "adminpassword",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/auth/login", loginReq, "")
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])
	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["access_token"])
	assert.Equal(t, "bearer", data["token_type"])
	assert.Greater(t, data["expires_in"].(float64), float64(0))

	userResp := data["user"].(map[string]interface{})
	assert.Equal(t, userID.String(), userResp["id"])
	assert.Equal(t, "admin", userResp["username"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestGetUserByID_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user and token
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles: []models.Role{
			{ID: uuid.New(), Name: "admin"},
		},
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "user", "read").Return(true, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	// Test successful user retrieval
	resp, body, err := makeRequest(app, "GET", fmt.Sprintf("/api/v1/users/%s", userID), nil, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.Equal(t, userID.String(), data["id"])
	assert.Equal(t, "testuser", data["username"])
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "Test", data["first_name"])
	assert.Equal(t, "User", data["last_name"])
	assert.True(t, data["is_active"].(bool))

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestCreateUser_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "user", "write").Return(true, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, "newuser").Return(nil, errors.New("user not found"))
	mockUserRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).Return(nil)

	// For getting the created user
	newUserID := uuid.New()
	mockUserRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&models.User{
		ID:        newUserID,
		Username:  "newuser",
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles:     []models.Role{},
	}, nil)

	// Test successful user creation
	createUserReq := map[string]interface{}{
		"username":   "newuser",
		"email":      "new@example.com",
		"password":   "password123",
		"first_name": "New",
		"last_name":  "User",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/users", createUserReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "newuser", data["username"])
	assert.Equal(t, "new@example.com", data["email"])
	assert.Equal(t, "New", data["first_name"])
	assert.Equal(t, "User", data["last_name"])
	assert.True(t, data["is_active"].(bool))

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestListRoles_Integration(t *testing.T) {
	app, mockUserRepo, mockRoleRepo, _ := setupTestApp()

	// Test roles
	roles := []*models.Role{
		{
			ID:          uuid.New(),
			Name:        "admin",
			Description: "Administrator role",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "user",
			Description: "Regular user role",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "role", "read").Return(true, nil)
	mockRoleRepo.On("GetAll", mock.Anything).Return(roles, nil)

	// Test successful roles retrieval
	resp, body, err := makeRequest(app, "GET", "/api/v1/roles", nil, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].([]interface{})
	assert.Len(t, data, 2)

	role1 := data[0].(map[string]interface{})
	assert.Equal(t, "admin", role1["name"])
	assert.Equal(t, "Administrator role", role1["description"])

	role2 := data[1].(map[string]interface{})
	assert.Equal(t, "user", role2["name"])
	assert.Equal(t, "Regular user role", role2["description"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
	mockRoleRepo.AssertExpectations(t)
}

func TestGetPermissionsByResource_Integration(t *testing.T) {
	app, mockUserRepo, _, mockPermRepo := setupTestApp()

	// Test permissions
	perms := []*models.Permission{
		{
			ID:          uuid.New(),
			Name:        "user:read",
			Description: "Read user",
			Resource:    "user",
			Action:      "read",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "user:write",
			Description: "Write user",
			Resource:    "user",
			Action:      "write",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "permission", "read").Return(true, nil)
	mockPermRepo.On("GetByResource", mock.Anything, "user").Return(perms, nil)

	// Test successful permissions retrieval
	resp, body, err := makeRequest(app, "GET", "/api/v1/permissions?resource=user", nil, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].([]interface{})
	assert.Len(t, data, 2)

	perm1 := data[0].(map[string]interface{})
	assert.Equal(t, "user:read", perm1["name"])
	assert.Equal(t, "Read user", perm1["description"])
	assert.Equal(t, "user", perm1["resource"])
	assert.Equal(t, "read", perm1["action"])

	perm2 := data[1].(map[string]interface{})
	assert.Equal(t, "user:write", perm2["name"])
	assert.Equal(t, "Write user", perm2["description"])
	assert.Equal(t, "user", perm2["resource"])
	assert.Equal(t, "write", perm2["action"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
	mockPermRepo.AssertExpectations(t)
}

func TestUnauthorizedAccess_Integration(t *testing.T) {
	app, _, _, _ := setupTestApp()

	// Test unauthorized access (no token)
	resp, body, err := makeRequest(app, "GET", "/api/v1/users", nil, "")
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.False(t, body["success"].(bool))
	assert.Contains(t, body["message"].(string), "Missing authorization header")
}

func TestForbiddenAccess_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "user", "write").Return(false, nil)

	// Test forbidden access (no permission)
	createUserReq := map[string]interface{}{
		"username":   "newuser",
		"email":      "new@example.com",
		"password":   "password123",
		"first_name": "New",
		"last_name":  "User",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/users", createUserReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.False(t, body["success"].(bool))
	assert.Contains(t, body["message"].(string), "Access denied")

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestUserGetMe_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		Username:  "currentuser",
		Email:     "current@example.com",
		FirstName: "Current",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles: []models.Role{
			{ID: uuid.New(), Name: "user"},
		},
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("GetByID", mock.Anything, mock.Anything).Return(user, nil)

	// For getting user permissions
	permissions := []models.Permission{
		{ID: uuid.New(), Name: "user:read", Resource: "user", Action: "read"},
	}
	mockUserRepo.On("GetUserPermissions", mock.Anything, mock.Anything).Return(permissions, nil)

	// Test successful "get me" operation
	resp, body, err := makeRequest(app, "GET", "/api/v1/users/me", nil, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["user"])
	assert.NotNil(t, data["permissions"])

	userData := data["user"].(map[string]interface{})
	assert.Equal(t, userID.String(), userData["id"])
	assert.Equal(t, "currentuser", userData["username"])

	permissionsData := data["permissions"].([]interface{})
	assert.Len(t, permissionsData, 1)
	assert.Equal(t, "user:read", permissionsData[0].(map[string]interface{})["name"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestCreateRole_Integration(t *testing.T) {
	app, mockUserRepo, mockRoleRepo, _ := setupTestApp()

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "role", "write").Return(true, nil)
	mockRoleRepo.On("GetByName", mock.Anything, "newrole").Return(nil, errors.New("role not found"))
	mockRoleRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).Return(nil)

	// For getting the created role
	roleID := uuid.New()
	mockRoleRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&models.Role{
		ID:          roleID,
		Name:        "newrole",
		Description: "New role description",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Permissions: []models.Permission{},
	}, nil)

	// Test successful role creation
	createRoleReq := map[string]interface{}{
		"name":        "newrole",
		"description": "New role description",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/roles", createRoleReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.Equal(t, roleID.String(), data["id"])
	assert.Equal(t, "newrole", data["name"])
	assert.Equal(t, "New role description", data["description"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
	mockRoleRepo.AssertExpectations(t)
}

func TestCreatePermission_Integration(t *testing.T) {
	app, mockUserRepo, _, mockPermRepo := setupTestApp()

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "permission", "write").Return(true, nil)
	mockPermRepo.On("GetByResourceAction", mock.Anything, "report", "read").Return(nil, errors.New("permission not found"))
	mockPermRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).Return(nil)

	// For getting the created permission
	permID := uuid.New()
	mockPermRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&models.Permission{
		ID:          permID,
		Name:        "report:read",
		Description: "Permission to read reports",
		Resource:    "report",
		Action:      "read",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil).Maybe()

	// Test successful permission creation
	createPermReq := map[string]interface{}{
		"name":        "report:read",
		"description": "Permission to read reports",
		"resource":    "report",
		"action":      "read",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/permissions", createPermReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "report:read", data["name"])
	assert.Equal(t, "Permission to read reports", data["description"])
	assert.Equal(t, "report", data["resource"])
	assert.Equal(t, "read", data["action"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
	mockPermRepo.AssertExpectations(t)
}

func TestUpdateUser_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles:     []models.Role{},
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "user", "write").Return(true, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	mockUserRepo.On("ExecuteTx", mock.Anything, mock.AnythingOfType("func(repositories.TxRepositoryInterface) error")).Return(nil)

	// Updated user
	updatedUser := *user
	updatedUser.FirstName = "Updated"
	updatedUser.LastName = "Name"
	mockUserRepo.On("GetByID", mock.Anything, userID).Return(&updatedUser, nil)

	// Test successful user update
	updateUserReq := map[string]interface{}{
		"first_name": "Updated",
		"last_name":  "Name",
	}

	resp, body, err := makeRequest(app, "PUT", fmt.Sprintf("/api/v1/users/%s", userID), updateUserReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.Equal(t, userID.String(), data["id"])
	assert.Equal(t, "testuser", data["username"])
	assert.Equal(t, "Updated", data["first_name"])
	assert.Equal(t, "Name", data["last_name"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestChangePassword_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user
	userID := uuid.New()
	hashedPassword := "$2a$10$FPS/DKJWlcHvU1fJuDEYDO0IXNoXQw./hCBlh90AogplwklD7PylC" // bcrypt hash for "adminpassword"
	user := &models.User{
		ID:        userID,
		Username:  "admin",
		Email:     "admin@example.com",
		Password:  hashedPassword,
		FirstName: "Admin",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("GetByID", mock.Anything, mock.Anything).Return(user, nil)
	mockUserRepo.On("UpdatePassword", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Test successful password change
	passwordChangeReq := map[string]interface{}{
		"current_password": "adminpassword",
		"new_password":     "newpassword123",
	}

	resp, body, err := makeRequest(app, "POST", "/api/v1/auth/change-password", passwordChangeReq, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.Equal(t, "Password changed successfully", body["message"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}

func TestDeleteUser_Integration(t *testing.T) {
	app, mockUserRepo, _, _ := setupTestApp()

	// Create test user
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Setup mock behaviors
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOlsiYWRtaW4iXSwiZXhwIjoxOTkxNjYwODAwfQ.tY_j1nkcr9wPT4BqiCsTlG8lSb7IsSBu_pK63bnCjFc"
	mockUserRepo.On("HasPermission", mock.Anything, mock.Anything, "user", "delete").Return(true, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	mockUserRepo.On("Delete", mock.Anything, userID).Return(nil)

	// Test successful user deletion
	resp, body, err := makeRequest(app, "DELETE", fmt.Sprintf("/api/v1/users/%s", userID), nil, token)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, body["success"].(bool))
	assert.Equal(t, "User deleted successfully", body["message"])

	// Verify mock expectations
	mockUserRepo.AssertExpectations(t)
}
