package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/chats/go-user-api/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// DB represents the database connection
type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection
func NewDB(cfg *config.Config) (*DB, error) {
	db, err := sqlx.Connect("postgres", cfg.GetDBConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &DB{DB: db}, nil
}

// Migrate applies database migrations
func (db *DB) Migrate() error {
	log.Info().Msg("Applying database migrations...")

	// Read init.sql file
	content, err := ioutil.ReadFile(filepath.Join("internal", "database", "migrations", "init.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute the migration
	_, err = db.ExecContext(context.Background(), string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Info().Msg("Database migrations applied successfully")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
