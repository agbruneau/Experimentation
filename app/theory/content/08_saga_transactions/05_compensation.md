# 8.5 Compensation et Rollback

## Résumé

Dans les transactions distribuées, le rollback classique n'existe pas. La **compensation** est une action qui annule sémantiquement les effets d'une opération précédente, sans pour autant l'effacer de l'historique.

### Points clés

- La compensation n'est PAS un rollback
- L'historique est préservé (audit)
- Chaque action doit avoir sa compensation définie
- La compensation doit être idempotente

## Différence Rollback vs Compensation

### Rollback (transaction classique)

```
BEGIN TRANSACTION
  INSERT INTO policies VALUES (...)   ← Écrit
  ERROR!
ROLLBACK                              ← Effacé comme si rien ne s'était passé
```

L'état de la DB revient exactement à avant.

### Compensation (transaction distribuée)

```
T1: INSERT INTO policies VALUES (...)  ← Policy créée
T2: INSERT INTO invoices VALUES (...)  ← Invoice créée
T3: ERROR!

COMPENSATION:
C2: UPDATE invoices SET status = 'CANCELLED'   ← Annulation sémantique
C1: UPDATE policies SET status = 'CANCELLED'   ← Annulation sémantique
```

L'historique montre : créé → annulé.

## Concevoir des compensations

### Règles de base

| Règle | Description |
|-------|-------------|
| **Inverse sémantique** | La compensation inverse l'effet métier |
| **Idempotence** | Peut être appelée plusieurs fois sans effet |
| **Autonomie** | Ne dépend pas d'état externe |
| **Traçabilité** | Laisse une trace dans l'historique |

### Exemples de compensations

| Action | Compensation |
|--------|--------------|
| Créer une police | Annuler la police |
| Réserver un paiement | Annuler la réservation |
| Envoyer un email | Envoyer un email de correction |
| Débiter un compte | Créditer le compte |
| Créer une facture | Créer un avoir |

## Implémentation

### Structure de compensation

```python
@dataclass
class CompensatableStep:
    name: str
    action: Callable
    compensate: Optional[Callable]
    is_compensatable: bool = True

    def has_compensation(self) -> bool:
        return self.compensate is not None and self.is_compensatable
```

### Actions et leurs compensations

```python
# Action: Créer une police
async def create_policy(ctx):
    policy = await policy_service.create({
        "customer_id": ctx["customer_id"],
        "product": ctx["product"]
    })
    ctx["policy_id"] = policy.id
    return {"policy_id": policy.id}

# Compensation: Annuler la police
async def cancel_policy(ctx):
    policy_id = ctx.get("policy_id")
    if policy_id:
        # Vérifier si pas déjà annulée (idempotence)
        policy = await policy_service.get(policy_id)
        if policy and policy.status != "CANCELLED":
            await policy_service.cancel(policy_id, reason="Saga compensation")
```

### Compensation avec état contextuel

```python
async def reserve_payment(ctx):
    """Réserve le paiement (préautorisation)."""
    reservation = await payment_service.reserve(
        amount=ctx["premium"],
        customer_id=ctx["customer_id"]
    )
    ctx["payment_reservation_id"] = reservation.id
    return {"reservation_id": reservation.id}

async def release_payment_reservation(ctx):
    """Libère la réservation (compensation)."""
    reservation_id = ctx.get("payment_reservation_id")
    if reservation_id:
        reservation = await payment_service.get_reservation(reservation_id)
        if reservation and reservation.status == "RESERVED":
            await payment_service.release(reservation_id)
```

## Patterns de compensation

### 1. Annulation directe

```python
# Action
await db.insert("policies", policy_data)

# Compensation
await db.update("policies", policy_id, {"status": "CANCELLED"})
```

### 2. Opération inverse

```python
# Action
await account_service.debit(account_id, amount)

# Compensation
await account_service.credit(account_id, amount, reason="Compensation")
```

### 3. Action compensatoire métier

```python
# Action: Facture émise
await billing_service.create_invoice(policy_id, amount)

# Compensation: Avoir émis
await billing_service.create_credit_note(invoice_id, amount, reason="Subscription cancelled")
```

### 4. Notification de compensation

```python
# Action: Email de bienvenue
await notification_service.send_welcome_email(customer_id)

# Compensation: Email d'annulation
await notification_service.send_cancellation_email(
    customer_id,
    reason="Your subscription has been cancelled"
)
```

## Compensation et idempotence

```python
async def compensate_create_policy(ctx):
    """Compensation idempotente."""
    policy_id = ctx.get("policy_id")

    if not policy_id:
        # Rien à compenser
        return

    # Récupérer l'état actuel
    policy = await policy_service.get(policy_id)

    if not policy:
        # Déjà supprimée ou n'existe pas
        return

    if policy.status == "CANCELLED":
        # Déjà compensée
        return

    # Effectuer la compensation
    await policy_service.cancel(policy_id)
```

## Compensation partielle

Parfois, seule une partie peut être compensée :

```python
async def compensate_payment(ctx):
    """Compensation partielle si paiement déjà capturé."""
    reservation_id = ctx.get("payment_reservation_id")
    reservation = await payment_service.get(reservation_id)

    if reservation.status == "RESERVED":
        # Peut annuler complètement
        await payment_service.release(reservation_id)

    elif reservation.status == "CAPTURED":
        # Doit faire un remboursement
        await payment_service.refund(reservation_id)

    # Si REFUNDED ou CANCELLED, rien à faire
```

## Compensation ordonnée

L'ordre de compensation est généralement l'inverse de l'exécution :

```python
# Exécution: 1 → 2 → 3 → 4 → ERREUR

# Compensation: 4 → 3 → 2 → 1
# (en pratique, souvent 3 → 2 → 1 car 4 n'a pas réussi)

async def compensate_saga(completed_steps, ctx):
    for step in reversed(completed_steps):
        if step.has_compensation():
            await step.compensate(ctx)
```

## Bonnes pratiques

### 1. Toujours définir la compensation

```python
# Pour chaque add_step avec action, penser à compensate
saga.add_step(
    action=create_policy,
    compensate=cancel_policy  # NE PAS OUBLIER !
)
```

### 2. Tester les compensations

```python
@pytest.mark.asyncio
async def test_policy_compensation():
    # Créer
    result = await create_policy({"customer_id": "C001"})

    # Compenser
    await cancel_policy({"policy_id": result["policy_id"]})

    # Vérifier
    policy = await policy_service.get(result["policy_id"])
    assert policy.status == "CANCELLED"

    # Test idempotence
    await cancel_policy({"policy_id": result["policy_id"]})
    # Pas d'erreur
```

### 3. Logger les compensations

```python
async def compensate_with_logging(step, ctx):
    logger.info(f"Starting compensation for {step.name}", extra={
        "saga_id": ctx.get("saga_id"),
        "step": step.name
    })

    try:
        await step.compensate(ctx)
        logger.info(f"Compensation succeeded for {step.name}")
    except Exception as e:
        logger.error(f"Compensation failed for {step.name}: {e}")
        raise
```
