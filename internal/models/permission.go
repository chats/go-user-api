package models

import (
	"time"

	"github.com/google/uuid"
)

// Permission represents a permission in the system
type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PermissionCreateRequest represents a request to create a permission
type PermissionCreateRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description"`
	Resource    string `json:"resource" validate:"required,min=1"`
	Action      string `json:"action" validate:"required,min=1"`
}

// PermissionUpdateRequest represents a request to update a permission
type PermissionUpdateRequest struct {
	Name        string `json:"name" validate:"omitempty,min=3,max=100"`
	Description string `json:"description"`
	Resource    string `json:"resource" validate:"omitempty,min=1"`
	Action      string `json:"action" validate:"omitempty,min=1"`
}

// PermissionResponse represents a permission response format
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts Permission to PermissionResponse
func (p *Permission) ToResponse() PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Resource:    p.Resource,
		Action:      p.Action,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
