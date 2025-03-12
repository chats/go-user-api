package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user role in the system
type Role struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty" db:"-"`
}

// RoleCreateRequest represents a request to create a role
type RoleCreateRequest struct {
	Name          string   `json:"name" validate:"required,min=3,max=50"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

// RoleUpdateRequest represents a request to update a role
type RoleUpdateRequest struct {
	Name          string   `json:"name" validate:"omitempty,min=3,max=50"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

// RoleResponse represents a role response format
type RoleResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// ToResponse converts Role to RoleResponse
func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Permissions: r.Permissions,
	}
}
