package database

import (
	"context"
	"fmt"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB represents the MongoDB database connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	cfg      *config.Config
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg *config.Config) (*MongoDB, error) {
	return &MongoDB{
		cfg: cfg,
	}, nil
}

// Connect establishes a connection to the database
func (db *MongoDB) Connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(db.cfg.GetMongoDBConnString())

	// Set a timeout for the connection
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db.Client = client
	db.Database = client.Database(db.cfg.MongoDBName)

	log.Info().Msg("Connected to MongoDB successfully")
	return nil
}

// Migrate creates initial collections and indexes for MongoDB
func (db *MongoDB) Migrate() error {
	log.Info().Msg("Setting up MongoDB collections and indexes...")

	ctx := context.Background()

	// Ensure collections exist by accessing them
	collections := []string{
		"users",
		"roles",
		"permissions",
		"user_roles",
		"role_permissions",
	}

	for _, collName := range collections {
		err := db.createCollectionIfNotExists(ctx, collName)
		if err != nil {
			return fmt.Errorf("failed to create collection %s: %w", collName, err)
		}
	}

	// Create indexes
	// Index for users collection
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := db.Database.Collection("users").Indexes().CreateMany(ctx, userIndexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes for users collection: %w", err)
	}

	// Index for roles collection
	roleIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = db.Database.Collection("roles").Indexes().CreateMany(ctx, roleIndexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes for roles collection: %w", err)
	}

	// Index for permissions collection
	permissionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = db.Database.Collection("permissions").Indexes().CreateMany(ctx, permissionIndexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes for permissions collection: %w", err)
	}

	// Index for user_roles collection
	userRolesIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "role_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = db.Database.Collection("user_roles").Indexes().CreateMany(ctx, userRolesIndexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes for user_roles collection: %w", err)
	}

	// Index for role_permissions collection
	rolePermissionsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "role_id", Value: 1}, {Key: "permission_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = db.Database.Collection("role_permissions").Indexes().CreateMany(ctx, rolePermissionsIndexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes for role_permissions collection: %w", err)
	}

	// Insert default roles and permissions if needed
	err = db.seedDefaultData(ctx)
	if err != nil {
		return fmt.Errorf("failed to seed default data: %w", err)
	}

	log.Info().Msg("MongoDB setup completed successfully")
	return nil
}

func (db *MongoDB) createCollectionIfNotExists(ctx context.Context, name string) error {
	collections, err := db.Database.ListCollectionNames(ctx, bson.M{"name": name})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// If collection doesn't exist, create it
	if len(collections) == 0 {
		err = db.Database.CreateCollection(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to create collection %s: %w", name, err)
		}
		log.Info().Str("collection", name).Msg("Created collection")
	}

	return nil
}

func (db *MongoDB) seedDefaultData(ctx context.Context) error {
	// Check if admin role exists
	adminRoleCount, err := db.Database.Collection("roles").CountDocuments(ctx, bson.M{"name": "admin"})
	if err != nil {
		return fmt.Errorf("failed to count admin role: %w", err)
	}

	// If no admin role, seed default roles
	if adminRoleCount == 0 {
		log.Info().Msg("Seeding default roles...")

		defaultRoles := []interface{}{
			bson.M{
				"_id":         generateObjectID(),
				"name":        "admin",
				"description": "Administrator with full access",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "supervisor",
				"description": "Supervisor with management permissions",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "editor",
				"description": "Editor with content modification permissions",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "viewer",
				"description": "Viewer with read-only permissions",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
		}

		_, err = db.Database.Collection("roles").InsertMany(ctx, defaultRoles)
		if err != nil {
			return fmt.Errorf("failed to seed default roles: %w", err)
		}
	}

	// Check if user:read permission exists
	userReadPermCount, err := db.Database.Collection("permissions").CountDocuments(ctx, bson.M{"name": "user:read"})
	if err != nil {
		return fmt.Errorf("failed to count user:read permission: %w", err)
	}

	// If no user:read permission, seed default permissions
	if userReadPermCount == 0 {
		log.Info().Msg("Seeding default permissions...")

		defaultPermissions := []interface{}{
			bson.M{
				"_id":         generateObjectID(),
				"name":        "user:read",
				"resource":    "user",
				"action":      "read",
				"description": "View user information",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "user:write",
				"resource":    "user",
				"action":      "write",
				"description": "Create or modify users",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "user:delete",
				"resource":    "user",
				"action":      "delete",
				"description": "Delete users",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "role:read",
				"resource":    "role",
				"action":      "read",
				"description": "View role information",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "role:write",
				"resource":    "role",
				"action":      "write",
				"description": "Create or modify roles",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "role:delete",
				"resource":    "role",
				"action":      "delete",
				"description": "Delete roles",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "permission:read",
				"resource":    "permission",
				"action":      "read",
				"description": "View permission information",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "permission:write",
				"resource":    "permission",
				"action":      "write",
				"description": "Create or modify permissions",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
			bson.M{
				"_id":         generateObjectID(),
				"name":        "permission:delete",
				"resource":    "permission",
				"action":      "delete",
				"description": "Delete permissions",
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			},
		}

		_, err = db.Database.Collection("permissions").InsertMany(ctx, defaultPermissions)
		if err != nil {
			return fmt.Errorf("failed to seed default permissions: %w", err)
		}
	}

	// Check if admin user exists
	adminUserCount, err := db.Database.Collection("users").CountDocuments(ctx, bson.M{"username": "admin"})
	if err != nil {
		return fmt.Errorf("failed to count admin user: %w", err)
	}

	// If no admin user, seed default admin user
	if adminUserCount == 0 {
		log.Info().Msg("Seeding default admin user...")

		// Default password is 'adminpassword'
		adminUser := bson.M{
			"_id":        generateObjectID(),
			"username":   "admin",
			"email":      "admin@example.com",
			"password":   "$2a$10$FPS/DKJWlcHvU1fJuDEYDO0IXNoXQw./hCBlh90AogplwklD7PylC",
			"first_name": "Admin",
			"last_name":  "User",
			"is_active":  true,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		}

		_, err = db.Database.Collection("users").InsertOne(ctx, adminUser)
		if err != nil {
			return fmt.Errorf("failed to seed default admin user: %w", err)
		}

		// Assign admin role to admin user
		// First get the admin user ID
		var adminUserDoc bson.M
		err = db.Database.Collection("users").FindOne(ctx, bson.M{"username": "admin"}).Decode(&adminUserDoc)
		if err != nil {
			return fmt.Errorf("failed to find admin user: %w", err)
		}

		// Then get the admin role ID
		var adminRoleDoc bson.M
		err = db.Database.Collection("roles").FindOne(ctx, bson.M{"name": "admin"}).Decode(&adminRoleDoc)
		if err != nil {
			return fmt.Errorf("failed to find admin role: %w", err)
		}

		// Assign admin role to admin user
		userRole := bson.M{
			"user_id":    adminUserDoc["_id"],
			"role_id":    adminRoleDoc["_id"],
			"created_at": time.Now(),
		}

		_, err = db.Database.Collection("user_roles").InsertOne(ctx, userRole)
		if err != nil {
			return fmt.Errorf("failed to assign admin role to admin user: %w", err)
		}

		// Assign all permissions to admin role
		// First get all permissions
		permissionsCursor, err := db.Database.Collection("permissions").Find(ctx, bson.M{})
		if err != nil {
			return fmt.Errorf("failed to find permissions: %w", err)
		}
		defer permissionsCursor.Close(ctx)

		// Assign each permission to admin role
		var rolePermissions []interface{}
		for permissionsCursor.Next(ctx) {
			var permDoc bson.M
			if err := permissionsCursor.Decode(&permDoc); err != nil {
				return fmt.Errorf("failed to decode permission: %w", err)
			}

			rolePermission := bson.M{
				"role_id":       adminRoleDoc["_id"],
				"permission_id": permDoc["_id"],
				"created_at":    time.Now(),
			}

			rolePermissions = append(rolePermissions, rolePermission)
		}

		if len(rolePermissions) > 0 {
			_, err = db.Database.Collection("role_permissions").InsertMany(ctx, rolePermissions)
			if err != nil {
				return fmt.Errorf("failed to assign permissions to admin role: %w", err)
			}
		}
	}

	return nil
}

func generateObjectID() string {
	return time.Now().Format(time.RFC3339Nano)
}

// Close closes the database connection
func (db *MongoDB) Close() error {
	if db.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return db.Client.Disconnect(ctx)
	}
	return nil
}

// GetImplementation returns the actual database implementation
func (db *MongoDB) GetImplementation() interface{} {
	return db
}

// GetCollection returns a MongoDB collection
func (db *MongoDB) GetCollection(name string) *mongo.Collection {
	return db.Database.Collection(name)
}
