package database

import (
	"context"
)

// Database represents the interface for different database implementations
type Database interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context) error
	// Close closes the database connection
	Close() error
	// Migrate applies database migrations
	Migrate() error
	// GetImplementation returns the actual database implementation
	GetImplementation() interface{}
}
