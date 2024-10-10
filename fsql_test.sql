-- Create a test role and database
CREATE ROLE test_user WITH LOGIN PASSWORD 'test_password';
CREATE DATABASE test_db WITH OWNER test_user;

-- Connect to the test_db as test_user
\c test_db test_user

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create necessary tables
CREATE TABLE ai_model (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key TEXT NOT NULL,
    name TEXT,
    description TEXT,
    type TEXT NOT NULL,
    provider TEXT NOT NULL,
    settings JSONB,
    default_negative_prompt TEXT
);
