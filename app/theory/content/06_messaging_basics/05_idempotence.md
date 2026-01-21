# 6.5 Idempotence et Déduplication

## Résumé

L'**idempotence** garantit qu'une opération peut être exécutée plusieurs fois avec le même résultat. C'est une propriété essentielle pour les systèmes de messaging utilisant at-least-once, où les doublons sont possibles.

### Points clés

- Une opération idempotente donne le même résultat si exécutée 1 ou N fois
- Nécessaire avec la garantie at-least-once
- Peut être implémentée de plusieurs façons
- La déduplication est une technique complémentaire

## Comprendre l'idempotence

### Opérations naturellement idempotentes

```python
# Idempotent : SET (écraser une valeur)
user.email = "new@email.com"  # Même résultat si exécuté 2 fois

# Idempotent : DELETE avec condition
DELETE FROM invoices WHERE id = 123  # OK si déjà supprimé

# Idempotent : PUT (remplacer une ressource)
PUT /api/policies/POL-001 { status: "ACTIVE" }
```

### Opérations NON idempotentes

```python
# NON idempotent : INCREMENT
balance += 100  # Double le montant si exécuté 2 fois !

# NON idempotent : INSERT
INSERT INTO invoices (amount, ...) VALUES (500, ...)  # Doublon !

# NON idempotent : POST (créer nouvelle ressource)
POST /api/payments { amount: 100 }  # Double paiement !
```

## Techniques d'idempotence

### 1. Clé d'idempotence

Utiliser un identifiant unique pour chaque requête :

```python
async def process_payment(event):
    idempotency_key = event["event_id"]

    # Vérifier si déjà traité
    if await db.exists("processed_events", idempotency_key):
        return  # Déjà traité, ignorer

    # Traiter
    await create_payment(event["data"])

    # Marquer comme traité (atomiquement si possible)
    await db.insert("processed_events", {
        "key": idempotency_key,
        "processed_at": now()
    })
```

### 2. Upsert (Insert or Update)

```python
# Au lieu de INSERT qui crée des doublons
INSERT INTO invoices (policy_id, month, amount) VALUES (...)

# Utiliser UPSERT
INSERT INTO invoices (policy_id, month, amount)
VALUES ('POL-001', '2024-01', 100)
ON CONFLICT (policy_id, month)
DO UPDATE SET amount = 100, updated_at = now()
```

### 3. Versioning optimiste

```python
async def update_policy_status(event):
    policy_id = event["data"]["policy_id"]
    new_status = event["data"]["status"]
    expected_version = event["data"]["version"]

    result = await db.execute("""
        UPDATE policies
        SET status = $1, version = version + 1
        WHERE id = $2 AND version = $3
    """, new_status, policy_id, expected_version)

    if result.rows_affected == 0:
        # Version mismatch = déjà traité ou conflit
        log.info(f"Skipping duplicate or conflicting update")
```

### 4. Transformation en opération idempotente

```python
# NON idempotent : incrémenter le solde
balance += payment_amount  # Problème si doublons

# IDEMPOTENT : définir le nouveau solde
# L'événement contient le solde final, pas le delta
async def update_balance(event):
    await db.execute("""
        UPDATE accounts
        SET balance = $1
        WHERE id = $2 AND last_update < $3
    """,
        event["data"]["new_balance"],
        event["data"]["account_id"],
        event["timestamp"]
    )
```

## Déduplication

### Table de déduplication

```sql
CREATE TABLE processed_messages (
    message_id VARCHAR(50) PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW(),
    result TEXT
);

-- Index pour nettoyage
CREATE INDEX idx_processed_at ON processed_messages(processed_at);
```

### Avec TTL (Time To Live)

```python
class MessageDeduplicator:
    def __init__(self, ttl_hours=24):
        self.ttl = ttl_hours * 3600
        self.cache = {}  # En prod : Redis avec SETEX

    async def is_duplicate(self, message_id):
        return message_id in self.cache

    async def mark_processed(self, message_id):
        self.cache[message_id] = {
            "processed_at": time.time()
        }
        # En prod: SETEX avec TTL

    async def cleanup_old_entries(self):
        cutoff = time.time() - self.ttl
        self.cache = {
            k: v for k, v in self.cache.items()
            if v["processed_at"] > cutoff
        }
```

## Patterns complets

### Pattern Consumer Idempotent

```python
class IdempotentConsumer:
    def __init__(self, deduplicator, handler):
        self.deduplicator = deduplicator
        self.handler = handler

    async def process(self, message):
        message_id = message.id

        # 1. Vérifier si doublon
        if await self.deduplicator.is_duplicate(message_id):
            log.info(f"Duplicate message {message_id}, skipping")
            return

        # 2. Traiter
        try:
            result = await self.handler(message.payload)

            # 3. Marquer comme traité (idéalement atomique avec le traitement)
            await self.deduplicator.mark_processed(message_id)

            return result
        except Exception as e:
            # Ne pas marquer comme traité si échec
            raise
```

### Pattern avec transaction atomique

```python
async def process_policy_event(event):
    async with db.transaction() as tx:
        # Vérifier ET verrouiller en une opération
        existing = await tx.execute("""
            INSERT INTO processed_events (event_id, status)
            VALUES ($1, 'processing')
            ON CONFLICT (event_id) DO NOTHING
            RETURNING event_id
        """, event["event_id"])

        if not existing:
            # Déjà traité ou en cours
            return

        try:
            # Traitement métier
            await create_policy_from_event(event, tx)

            # Marquer comme terminé
            await tx.execute("""
                UPDATE processed_events
                SET status = 'completed', completed_at = now()
                WHERE event_id = $1
            """, event["event_id"])

        except Exception:
            await tx.rollback()
            raise
```

## Cas d'usage Assurance

### Facturation mensuelle

```python
async def create_monthly_invoice(event):
    policy_id = event["data"]["policy_id"]
    month = event["data"]["billing_month"]  # Ex: "2024-01"
    amount = event["data"]["amount"]

    # Clé naturelle d'idempotence : policy + mois
    await db.execute("""
        INSERT INTO invoices (policy_id, billing_month, amount, status)
        VALUES ($1, $2, $3, 'PENDING')
        ON CONFLICT (policy_id, billing_month)
        DO NOTHING  -- Ignorer si déjà créée
    """, policy_id, month, amount)
```

### Notification client

```python
async def send_notification(event):
    notification_id = event["event_id"]

    # Utiliser l'ID d'événement comme clé d'idempotence
    sent = await notification_service.send_if_not_sent(
        notification_id,
        event["data"]["customer_email"],
        event["data"]["template"],
        event["data"]["context"]
    )

    if not sent:
        log.info(f"Notification {notification_id} already sent")
```

## Checklist idempotence

| Vérification | ✓ |
|--------------|---|
| Chaque message a un ID unique | |
| Les handlers vérifient les doublons | |
| Les opérations d'écriture sont idempotentes | |
| La table de déduplication a un TTL | |
| Les tests incluent des scénarios de doublons | |

## Points d'attention

1. **La fenêtre de déduplication** doit couvrir le temps max de retry
2. **La table de déduplication** peut devenir volumineuse - prévoir le nettoyage
3. **Les opérations atomiques** sont préférables mais pas toujours possibles
4. **Tester avec des doublons intentionnels** pour valider l'implémentation
