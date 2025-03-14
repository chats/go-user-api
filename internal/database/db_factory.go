package database

import (
	"context"
	"fmt"

	"github.com/chats/go-user-api/config"
)

// NewDatabase creates a new database connection based on configuration
func NewDatabase(cfg *config.Config) (Database, error) {
	var db Database
	var err error

	// Create database connection based on configuration
	switch cfg.DBType {
	case "postgres":
		db, err = NewPostgresDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL database: %w", err)
		}
	case "mongodb":
		db, err = NewMongoDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create MongoDB database: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}

	// Connect to the database
	err = db.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
