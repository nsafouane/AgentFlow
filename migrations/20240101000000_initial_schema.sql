-- +goose Up
-- Initial schema migration for AgentFlow
-- This migration establishes the foundational database structure

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Initial placeholder table to establish migration baseline
CREATE TABLE IF NOT EXISTS migration_baseline (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT NOT NULL DEFAULT 'Initial migration baseline'
);

-- Insert baseline record
INSERT INTO migration_baseline (description) VALUES ('AgentFlow initial schema baseline');

-- +goose Down
-- Rollback initial schema
DROP TABLE IF EXISTS migration_baseline;
DROP EXTENSION IF EXISTS "uuid-ossp";