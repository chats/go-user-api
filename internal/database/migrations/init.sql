-- Create tables
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(resource, action)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

-- Insert default roles
INSERT INTO roles (name, description) 
VALUES 
    ('admin', 'Administrator with full access'),
    ('supervisor', 'Supervisor with management permissions'),
    ('editor', 'Editor with content modification permissions'),
    ('viewer', 'Viewer with read-only permissions')
ON CONFLICT (name) DO NOTHING;

-- Insert default permissions
INSERT INTO permissions (name, resource, action, description)
VALUES
    ('user:read', 'user', 'read', 'View user information'),
    ('user:write', 'user', 'write', 'Create or modify users'),
    ('user:delete', 'user', 'delete', 'Delete users'),
    ('role:read', 'role', 'read', 'View role information'),
    ('role:write', 'role', 'write', 'Create or modify roles'),
    ('role:delete', 'role', 'delete', 'Delete roles'),
    ('permission:read', 'permission', 'read', 'View permission information'),
    ('permission:write', 'permission', 'write', 'Create or modify permissions'),
    ('permission:delete', 'permission', 'delete', 'Delete permissions')
ON CONFLICT (resource, action) DO NOTHING;

-- Assign permissions to roles
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'admin'),
    id
FROM permissions
ON CONFLICT DO NOTHING;

-- Supervisor gets read and write permissions, but not delete
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'supervisor'),
    id
FROM permissions
WHERE action != 'delete'
ON CONFLICT DO NOTHING;

-- Editor gets read permission for all resources and write permission for content
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'editor'),
    id
FROM permissions
WHERE action = 'read' OR (action = 'write' AND resource IN ('user'))
ON CONFLICT DO NOTHING;

-- Viewer gets only read permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE name = 'viewer'),
    id
FROM permissions
WHERE action = 'read'
ON CONFLICT DO NOTHING;

-- Create default admin user (password is 'adminpassword')
INSERT INTO users (username, email, password, first_name, last_name) 
VALUES ('admin', 'admin@example.com', '$2a$10$WY0AEJESgw6QU.NqvyCP3.DBaIwjUTXUUOejAUAt1ipDHu5qW37XC', 'Admin', 'User')
ON CONFLICT (username) DO NOTHING;

-- Assign admin role to admin user
INSERT INTO user_roles (user_id, role_id)
SELECT 
    (SELECT id FROM users WHERE username = 'admin'),
    (SELECT id FROM roles WHERE name = 'admin')
ON CONFLICT DO NOTHING;