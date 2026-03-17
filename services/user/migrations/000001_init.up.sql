-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for username lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Index for display_name search (partial, for future full-text search)
CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(display_name);
