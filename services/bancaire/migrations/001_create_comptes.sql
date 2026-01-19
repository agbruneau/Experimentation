-- Migration: Create bancaire schema and tables
-- Version: 001
-- Description: Initial schema for Bancaire service

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS bancaire;

-- Comptes table
CREATE TABLE IF NOT EXISTS bancaire.comptes (
    id VARCHAR(50) PRIMARY KEY,
    client_id VARCHAR(50) NOT NULL,
    type_compte VARCHAR(20) NOT NULL CHECK (type_compte IN ('COURANT', 'EPARGNE', 'JOINT')),
    solde DECIMAL(18, 2) NOT NULL DEFAULT 0.00,
    devise VARCHAR(3) NOT NULL DEFAULT 'EUR',
    statut VARCHAR(20) NOT NULL DEFAULT 'ACTIF' CHECK (statut IN ('ACTIF', 'INACTIF', 'FERME')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Transactions table
CREATE TABLE IF NOT EXISTS bancaire.transactions (
    id VARCHAR(50) PRIMARY KEY,
    compte_id VARCHAR(50) NOT NULL REFERENCES bancaire.comptes(id),
    event_id VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('OUVERTURE', 'DEPOT', 'RETRAIT', 'VIREMENT', 'PAIEMENT')),
    montant DECIMAL(18, 2) NOT NULL,
    devise VARCHAR(3) NOT NULL DEFAULT 'EUR',
    solde_apres DECIMAL(18, 2) NOT NULL,
    reference VARCHAR(100),
    description TEXT,
    compte_source_id VARCHAR(50),
    compte_dest_id VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Processed events table for idempotency
CREATE TABLE IF NOT EXISTS bancaire.processed_events (
    event_id VARCHAR(50) PRIMARY KEY,
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_comptes_client ON bancaire.comptes(client_id);
CREATE INDEX IF NOT EXISTS idx_comptes_statut ON bancaire.comptes(statut);
CREATE INDEX IF NOT EXISTS idx_transactions_compte ON bancaire.transactions(compte_id);
CREATE INDEX IF NOT EXISTS idx_transactions_event ON bancaire.transactions(event_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON bancaire.transactions(created_at DESC);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION bancaire.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS comptes_updated_at ON bancaire.comptes;
CREATE TRIGGER comptes_updated_at
    BEFORE UPDATE ON bancaire.comptes
    FOR EACH ROW
    EXECUTE FUNCTION bancaire.update_updated_at();
