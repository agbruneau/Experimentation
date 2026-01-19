-- ============================================================================
-- EDA-Lab PostgreSQL Initialization Script
-- ============================================================================

-- Create schema for the Bancaire domain
CREATE SCHEMA IF NOT EXISTS bancaire;

-- Health check table
CREATE TABLE IF NOT EXISTS bancaire.health_check (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL DEFAULT 'OK',
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial health check record
INSERT INTO bancaire.health_check (status) VALUES ('INITIALIZED');

-- ============================================================================
-- Bancaire Domain Tables
-- ============================================================================

-- Comptes table
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

-- Transactions table
CREATE TABLE IF NOT EXISTS bancaire.transactions (
    id VARCHAR(36) PRIMARY KEY,
    compte_id VARCHAR(36) NOT NULL REFERENCES bancaire.comptes(id),
    type VARCHAR(30) NOT NULL,
    montant DECIMAL(18,2) NOT NULL,
    reference VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_comptes_client ON bancaire.comptes(client_id);
CREATE INDEX IF NOT EXISTS idx_comptes_statut ON bancaire.comptes(statut);
CREATE INDEX IF NOT EXISTS idx_transactions_compte ON bancaire.transactions(compte_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON bancaire.transactions(created_at);

-- ============================================================================
-- Event Store (for future Event Sourcing pattern)
-- ============================================================================

CREATE TABLE IF NOT EXISTS bancaire.event_store (
    event_id VARCHAR(36) PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(aggregate_type, aggregate_id, version)
);

CREATE INDEX IF NOT EXISTS idx_event_store_aggregate ON bancaire.event_store(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_store_type ON bancaire.event_store(event_type);
CREATE INDEX IF NOT EXISTS idx_event_store_created ON bancaire.event_store(created_at);

-- ============================================================================
-- Grant permissions
-- ============================================================================
GRANT ALL PRIVILEGES ON SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA bancaire TO edalab;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA bancaire TO edalab;

-- Verification message
DO $$
BEGIN
    RAISE NOTICE 'EDA-Lab database initialized successfully!';
END $$;
