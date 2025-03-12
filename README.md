# go-user-api

A microservice for user authentication and authorization with JWT, roles, and permissions management.

## Features

- JWT-based authentication
- Role-based access control (RBAC)
- Permission management
- RESTful HTTP API
- gRPC API for user profile
- RabbitMQ for job queue (email sending)
- Distributed logging with Kafka
- Distributed tracing with Jaeger
- Redis caching for database queries
- PostgreSQL database
- Swagger documentation
- Docker and Docker Compose support

## Requirements

- Go 1.19 or higher
- PostgreSQL
- Redis
- RabbitMQ
- Kafka
- Jaeger (optional, for tracing)

## Getting Started

### Using Docker Compose

The easiest way to run the project is with Docker Compose:

```bash
# Clone the repository
git clone https://github.com/chats/go-user-api.git
cd go-user-api

# Start all services
docker-compose up -d
```

This will start the API service along with PostgreSQL, Redis, RabbitMQ, Kafka, and Jaeger.

### Manual Setup

If you prefer to run the project manually:

1. Install Go 1.19 or higher
2. Clone the repository
3. Install dependencies
4. Set up PostgreSQL, Redis, RabbitMQ, and Kafka
5. Configure environment variables
6. Run the application

```bash
# Clone the repository
git clone https://github.com/chats/go-user-api.git
cd go-user-api

# Install dependencies
go mod download

# Set up environment variables (copy .env.example to .env and modify)
cp .env.example .env

# Run the application
go run cmd/server/main.go
```

## API Documentation

The API documentation is available at `/swagger/` when the server is running.

## API Endpoints

### Authentication

- `POST /api/v1/auth/login` - Login with username and password
- `POST /api/v1/auth/change-password` - Change password (authenticated)
- `POST /api/v1/auth/reset-password` - Reset password (admin only)

### Users

- `GET /api/v1/users` - Get all users (requires user:read permission)
- `POST /api/v1/users` - Create a user (requires user:write permission)
- `GET /api/v1/users/me` - Get current user profile
- `GET /api/v1/users/:id` - Get a user by ID (requires user:read permission)
- `PUT /api/v1/users/:id` - Update a user (requires user:write permission)
- `DELETE /api/v1/users/:id` - Delete a user (requires user:delete permission)
- `GET /api/v1/users/:id/permissions` - Get user permissions (requires user:read permission)

### Roles

- `GET /api/v1/roles` - Get all roles (requires role:read permission)
- `POST /api/v1/roles` - Create a role (requires role:write permission)
- `GET /api/v1/roles/:id` - Get a role by ID (requires role:read permission)
- `PUT /api/v1/roles/:id` - Update a role (requires role:write permission)
- `DELETE /api/v1/roles/:id` - Delete a role (requires role:delete permission)
- `GET /api/v1/roles/:id/permissions` - Get role permissions (requires role:read permission)

### Permissions

- `GET /api/v1/permissions` - Get all permissions (requires permission:read permission)
- `POST /api/v1/permissions` - Create a permission (requires permission:write permission)
- `GET /api/v1/permissions/:id` - Get a permission by ID (requires permission:read permission)
- `PUT /api/v1/permissions/:id` - Update a permission (requires permission:write permission)
- `DELETE /api/v1/permissions/:id` - Delete a permission (requires permission:delete permission)

## gRPC API

The service also provides a gRPC API for user profile and permission checking:

- `GetUser` - Get user profile by ID
- `GetUserPermissions` - Get user permissions
- `ValidateToken` - Validate JWT token
- `HasPermission` - Check if a user has a specific permission

## Development

### Generating Protocol Buffers

```bash
# Install Protocol Buffer compiler
# https://github.com/protocolbuffers/protobuf/releases

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# On MacOS using Homebrew
brew install protobuf
# On Linux (Debian based) using apt
apt install -y protobuf-compiler
# On Windows using Winget
winget install protobuf 

# Generate code
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/grpc/proto/user.proto
```

### Generating Swagger Documentation

```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger docs
swag init -g cmd/server/main.go -o docs
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.