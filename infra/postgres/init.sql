-- EDA-Lab PostgreSQL Initialization Script
-- Creates schemas and initial tables for all services

-- Create schemas for each domain
CREATE SCHEMA IF NOT EXISTS bancaire;
CREATE SCHEMA IF NOT EXISTS events;

-- Health check table
CREATE TABLE IF NOT EXISTS bancaire.health_check (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL DEFAULT 'OK',
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO bancaire.health_check (status) VALUES ('initialized');

-- Bancaire domain tables
CREATE TABLE IF NOT EXISTS bancaire.comptes (
    id VARCHAR(36) PRIMARY KEY,
    client_id VARCHAR(36) NOT NULL,
    type_compte VARCHAR(20) NOT NULL,
    devise VARCHAR(3) NOT NULL DEFAULT 'EUR',
    solde DECIMAL(18,2) NOT NULL DEFAULT 0,
    statut VARCHAR(20) NOT NULL DEFAULT 'ACTIF',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bancaire.transactions (
    id VARCHAR(36) PRIMARY KEY,
    compte_id VARCHAR(36) NOT NULL REFERENCES bancaire.comptes(id),
    type VARCHAR(30) NOT NULL,
    montant DECIMAL(18,2) NOT NULL,
    reference VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Event tracking for idempotency
CREATE TABLE IF NOT EXISTS events.processed_events (
    event_id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_comptes_client ON bancaire.comptes(client_id);
CREATE INDEX IF NOT EXISTS idx_transactions_compte ON bancaire.transactions(compte_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON bancaire.transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_processed_events_type ON events.processed_events(event_type);

-- Updated timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to comptes table
DROP TRIGGER IF EXISTS update_comptes_updated_at ON bancaire.comptes;
CREATE TRIGGER update_comptes_updated_at
    BEFORE UPDATE ON bancaire.comptes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions
GRANT ALL PRIVILEGES ON SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON SCHEMA events TO edalab;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA events TO edalab;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA events TO edalab;
