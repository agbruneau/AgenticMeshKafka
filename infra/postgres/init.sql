-- EDA-Lab PostgreSQL initialization script

-- Create bancaire schema
CREATE SCHEMA IF NOT EXISTS bancaire;

-- Create health check table for testing
CREATE TABLE IF NOT EXISTS bancaire.health_check (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert a test row
INSERT INTO bancaire.health_check (status) VALUES ('healthy');

-- Grant permissions
GRANT ALL PRIVILEGES ON SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA bancaire TO edalab;
