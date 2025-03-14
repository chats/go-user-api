package repositories

import (
	"fmt"

	"github.com/chats/go-user-api/config"
	"github.com/chats/go-user-api/internal/cache"
	"github.com/chats/go-user-api/internal/database"
)

// RepositoryFactory creates repositories based on database type
type RepositoryFactory struct {
	cfg   *config.Config
	db    database.Database
	cache *cache.RedisClient
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(cfg *config.Config, db database.Database, cache *cache.RedisClient) *RepositoryFactory {
	return &RepositoryFactory{
		cfg:   cfg,
		db:    db,
		cache: cache,
	}
}

// CreateUserRepository creates a user repository based on database type
func (f *RepositoryFactory) CreateUserRepository() (UserRepositoryInterface, error) {
	switch f.cfg.DBType {
	case "postgres":
		// We need to cast the database to PostgresDB
		postgresDB, ok := f.db.GetImplementation().(*database.PostgresDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to PostgresDB")
		}
		return NewUserRepository(postgresDB, f.cache), nil
	case "mongodb":
		// We need to cast the database to MongoDB
		mongoDB, ok := f.db.GetImplementation().(*database.MongoDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to MongoDB")
		}
		return NewMongoUserRepository(mongoDB, f.cache), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", f.cfg.DBType)
	}
}

// CreateRoleRepository creates a role repository based on database type
func (f *RepositoryFactory) CreateRoleRepository() (RoleRepositoryInterface, error) {
	switch f.cfg.DBType {
	case "postgres":
		// We need to cast the database to PostgresDB
		postgresDB, ok := f.db.GetImplementation().(*database.PostgresDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to PostgresDB")
		}
		return NewRoleRepository(postgresDB, f.cache), nil
	case "mongodb":
		// We need to cast the database to MongoDB
		mongoDB, ok := f.db.GetImplementation().(*database.MongoDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to MongoDB")
		}
		return NewMongoRoleRepository(mongoDB, f.cache), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", f.cfg.DBType)
	}
}

// CreatePermissionRepository creates a permission repository based on database type
func (f *RepositoryFactory) CreatePermissionRepository() (PermissionRepositoryInterface, error) {
	switch f.cfg.DBType {
	case "postgres":
		// We need to cast the database to PostgresDB
		postgresDB, ok := f.db.GetImplementation().(*database.PostgresDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to PostgresDB")
		}
		return NewPermissionRepository(postgresDB, f.cache), nil
	case "mongodb":
		// We need to cast the database to MongoDB
		mongoDB, ok := f.db.GetImplementation().(*database.MongoDB)
		if !ok {
			return nil, fmt.Errorf("failed to cast database implementation to MongoDB")
		}
		return NewMongoPermissionRepository(mongoDB, f.cache), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", f.cfg.DBType)
	}
}
