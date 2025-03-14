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

// PostgresDB represents the PostgreSQL database connection
type PostgresDB struct {
	*sqlx.DB
	cfg *config.Config
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	return &PostgresDB{
		cfg: cfg,
	}, nil
}

// Connect establishes a connection to the database
func (db *PostgresDB) Connect(ctx context.Context) error {
	sqlxDB, err := sqlx.ConnectContext(ctx, "postgres", db.cfg.GetDBConnString())
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	// Set connection pool settings
	sqlxDB.SetMaxOpenConns(50)
	sqlxDB.SetMaxIdleConns(10)

	db.DB = sqlxDB
	return nil
}

// Migrate applies database migrations
func (db *PostgresDB) Migrate() error {
	log.Info().Msg("Applying PostgreSQL database migrations...")

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

	log.Info().Msg("PostgreSQL database migrations applied successfully")
	return nil
}

// Close closes the database connection
func (db *PostgresDB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// GetImplementation returns the actual database implementation
func (db *PostgresDB) GetImplementation() interface{} {
	return db.DB
}
