-- Migration: 001_create_users_table
-- Direction: UP
-- Description: Creates the users table with all required columns, constraints, and indexes.

BEGIN;

-- Enable UUID extension (idempotent)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- USERS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(320) NOT NULL,
    password_hash TEXT         NOT NULL,
    first_name    VARCHAR(100) NOT NULL,
    last_name     VARCHAR(100) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'member'
                               CHECK (role IN ('admin', 'manager', 'member', 'guest')),
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ  -- Soft-delete support
);

-- Unique constraint on active (non-deleted) emails only
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique
    ON users (email)
    WHERE deleted_at IS NULL;

-- Performance indexes
CREATE INDEX IF NOT EXISTS users_role_idx       ON users (role)       WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS users_is_active_idx  ON users (is_active)  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at) WHERE deleted_at IS NULL;

-- Full-text search index
CREATE INDEX IF NOT EXISTS users_search_idx
    ON users USING GIN (
        to_tsvector('english', first_name || ' ' || last_name || ' ' || email)
    )
    WHERE deleted_at IS NULL;

-- Auto-update updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMIT;
