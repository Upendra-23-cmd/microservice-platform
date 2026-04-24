-- Migration: 001_create_users_table
-- Direction: DOWN

BEGIN;
DROP TRIGGER  IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE    IF EXISTS users;
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
COMMIT;
