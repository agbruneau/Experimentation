# 8.4 Outbox Pattern

## Résumé

L'**Outbox Pattern** garantit l'atomicité entre une modification en base de données et la publication d'un événement. L'événement est d'abord écrit dans une table "outbox" dans la même transaction, puis publié de manière asynchrone.

### Points clés

- Événement écrit en base DANS la même transaction que les données
- Publication asynchrone par polling ou CDC
- Garantit qu'aucun événement n'est perdu
- Résout le problème du "dual write"

## Le problème du Dual Write

### Scénario problématique

```python
# DANGER: Dual write non atomique
async def create_policy(data):
    # 1. Écrire en base
    await db.insert_policy(data)

    # 2. Publier l'événement
    await broker.publish("PolicyCreated", data)
    # ↑ Si ça échoue ICI, la base est modifiée mais pas de message !
```

### Scénarios de panne

| Scénario | DB | Message | Problème |
|----------|----|---------|----|
| Panne après INSERT | ✓ | ✗ | Événement perdu |
| Panne après PUBLISH | ✓ | ✓ | OK |
| Panne réseau vers broker | ✓ | ? | Incertain |

## Solution : Outbox Pattern

### Principe

```
┌─────────────────────────────────────────────────────────────┐
│                    MÊME TRANSACTION                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  BEGIN TRANSACTION                                          │
│                                                             │
│  1. INSERT INTO policies (...)                              │
│  2. INSERT INTO outbox (event_type, payload, ...)           │
│                                                             │
│  COMMIT  ← Atomique!                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
            │
            │ Asynchrone
            ▼
┌─────────────────────────────────────────────────────────────┐
│                    OUTBOX PROCESSOR                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. SELECT * FROM outbox WHERE status = 'pending'           │
│  2. Pour chaque entrée:                                     │
│     - Publier vers broker                                   │
│     - UPDATE outbox SET status = 'published'                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Implémentation

### Structure de la table outbox

```sql
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    published_at TIMESTAMP,
    retries INT DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_pending ON outbox(status) WHERE status = 'pending';
```

### Code d'écriture atomique

```python
async def create_policy(data):
    async with db.transaction() as tx:
        # 1. Créer la police
        policy = await tx.execute("""
            INSERT INTO policies (customer_id, product, premium)
            VALUES ($1, $2, $3)
            RETURNING id
        """, data["customer_id"], data["product"], data["premium"])

        policy_id = policy["id"]

        # 2. Ajouter à l'outbox (MÊME TRANSACTION)
        await tx.execute("""
            INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload)
            VALUES ($1, $2, $3, $4)
        """,
            "Policy",
            policy_id,
            "PolicyCreated",
            json.dumps({
                "policy_id": policy_id,
                "customer_id": data["customer_id"],
                "product": data["product"],
                "premium": data["premium"]
            })
        )

        # 3. COMMIT atomique
        return policy_id
```

### Outbox Processor (Polling)

```python
class OutboxProcessor:
    def __init__(self, db, broker):
        self.db = db
        self.broker = broker

    async def process_pending(self):
        """Traite les entrées en attente."""
        entries = await self.db.execute("""
            SELECT * FROM outbox
            WHERE status = 'pending'
            ORDER BY created_at
            LIMIT 100
            FOR UPDATE SKIP LOCKED
        """)

        for entry in entries:
            try:
                # Publier
                await self.broker.publish(entry["event_type"], entry["payload"])

                # Marquer comme publié
                await self.db.execute("""
                    UPDATE outbox
                    SET status = 'published', published_at = NOW()
                    WHERE id = $1
                """, entry["id"])

            except Exception as e:
                # Incrémenter le compteur de retry
                await self.db.execute("""
                    UPDATE outbox
                    SET retries = retries + 1, last_error = $1
                    WHERE id = $2
                """, str(e), entry["id"])

    async def start(self, interval_seconds=1):
        """Démarre le polling."""
        while True:
            await self.process_pending()
            await asyncio.sleep(interval_seconds)
```

## Alternative : CDC (Change Data Capture)

Au lieu du polling, utiliser CDC pour détecter les nouvelles entrées :

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Database  │────►│  Debezium   │────►│   Kafka     │
│   (outbox)  │ CDC │  (CDC tool) │     │   (broker)  │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Avantages du CDC

- Pas de polling (moins de charge DB)
- Latence plus faible
- Ordre garanti

### Inconvénients

- Infrastructure supplémentaire
- Complexité accrue

## Gestion des doublons

Le consommateur DOIT être idempotent car :
- Le processor peut republier après crash
- Le broker peut délivrer plusieurs fois

```python
# Côté consommateur
class PolicyEventHandler:
    async def handle(self, event):
        # Vérifier si déjà traité
        if await self.is_processed(event["id"]):
            return

        await self.process(event)
        await self.mark_processed(event["id"])
```

## Maintenance

### Purge des anciennes entrées

```sql
-- Supprimer les entrées publiées de plus de 7 jours
DELETE FROM outbox
WHERE status = 'published'
AND published_at < NOW() - INTERVAL '7 days';
```

### Monitoring

```sql
-- Alertes si trop d'entrées en attente
SELECT COUNT(*) FROM outbox WHERE status = 'pending';

-- Entrées en échec
SELECT * FROM outbox WHERE retries > 5;
```

## Sandbox : Scénario EVT-06

Dans le scénario EVT-06, vous allez :
1. Créer une table outbox
2. Écrire une transaction atomique (données + outbox)
3. Configurer le polling
4. Observer la publication différée
5. Gérer les doublons côté consommateur
6. Simuler une panne et vérifier la récupération
