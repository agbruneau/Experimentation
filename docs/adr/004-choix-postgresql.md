# ADR 004: Choix de PostgreSQL comme base de données

## Statut

Accepté

## Contexte

Le service Bancaire nécessite une base de données pour persister les comptes et transactions. Options considérées:

- PostgreSQL
- MySQL/MariaDB
- MongoDB
- CockroachDB
- SQLite

## Décision

Nous avons choisi **PostgreSQL 16**.

## Justification

### Avantages

1. **Robustesse**: ACID complet, transactions fiables
2. **SQL standard**: Portabilité, compétences transférables
3. **Performance**: Excellent pour les workloads mixtes
4. **Extensions**: jsonb, arrays, full-text search
5. **Écosystème**: pgx (Go), migrations, outils d'administration
6. **Gratuité**: Open source, pas de vendor lock-in
7. **Documentation**: Excellente documentation officielle

### Inconvénients

1. **Scalabilité horizontale**: Moins native que les bases NoSQL
2. **Sharding**: Requiert des outils tiers (Citus)
3. **Schema rigide**: Moins flexible que MongoDB

### Alternatives rejetées

| Alternative | Raison du rejet |
|-------------|-----------------|
| MySQL | Moins de fonctionnalités (JSON, arrays) |
| MongoDB | Overkill pour le cas d'usage, pas de transactions multi-documents natives historiquement |
| CockroachDB | Plus complexe, pas nécessaire pour un lab |
| SQLite | Pas adapté au multi-conteneur |

## Conséquences

- Schema par domaine (ex: `bancaire.comptes`)
- Migrations SQL versionnées
- Utilisation de pgx/v5 comme driver Go
- Connection pooling via pgxpool

## Schéma de données Bancaire

```sql
CREATE SCHEMA IF NOT EXISTS bancaire;

CREATE TABLE bancaire.comptes (
    id VARCHAR(36) PRIMARY KEY,
    client_id VARCHAR(36) NOT NULL,
    type_compte VARCHAR(20) NOT NULL,
    devise VARCHAR(3) DEFAULT 'EUR',
    solde DECIMAL(18, 2) DEFAULT 0.00,
    statut VARCHAR(20) DEFAULT 'ACTIF',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE bancaire.transactions (
    id VARCHAR(36) PRIMARY KEY,
    compte_id VARCHAR(36) REFERENCES bancaire.comptes(id),
    type_transaction VARCHAR(20) NOT NULL,
    montant DECIMAL(18, 2) NOT NULL,
    devise VARCHAR(3) DEFAULT 'EUR',
    reference VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE bancaire.processed_events (
    event_id VARCHAR(36) PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW()
);
```

## Indexes

```sql
CREATE INDEX idx_comptes_client ON bancaire.comptes(client_id);
CREATE INDEX idx_transactions_compte ON bancaire.transactions(compte_id);
CREATE INDEX idx_transactions_created ON bancaire.transactions(created_at);
```
