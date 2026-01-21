# 8.2 Saga Pattern - Orchestration

## Résumé

Le pattern **Saga Orchestration** utilise un orchestrateur central pour coordonner une séquence d'étapes locales. En cas d'échec, l'orchestrateur déclenche les compensations dans l'ordre inverse.

### Points clés

- Un orchestrateur central gère le flux
- Chaque étape a une action et une compensation
- L'ordre des compensations est l'inverse des actions
- Visibilité complète du workflow

## Principe

```
┌─────────────────────────────────────────────────────────────┐
│                    SAGA ORCHESTRATOR                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐        │
│    │ T1  │──►│ T2  │──►│ T3  │──►│ T4  │──►│ T5  │        │
│    │Quote│   │Policy│  │Invoice│ │ Doc  │   │Notif│        │
│    └──┬──┘   └──┬──┘   └──┬──┘   └──┬──┘   └─────┘        │
│       │         │         │         │                       │
│    ┌──▼──┐   ┌──▼──┐   ┌──▼──┐   ┌──▼──┐                  │
│    │ C1  │◄──│ C2  │◄──│ C3  │◄──│ C4  │    Compensation  │
│    └─────┘   └─────┘   └─────┘   └─────┘                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Implémentation

### Structure d'une Saga

```python
class SagaOrchestrator:
    def __init__(self):
        self.steps = []

    def add_step(self, action, compensate=None):
        """Ajoute une étape avec sa compensation."""
        self.steps.append({
            "action": action,
            "compensate": compensate
        })

    async def execute(self, context):
        """Exécute la saga complète."""
        completed_steps = []

        try:
            for step in self.steps:
                result = await step["action"](context)
                context.update(result)
                completed_steps.append(step)

            return {"status": "COMPLETED", "context": context}

        except Exception as e:
            # Compensation en ordre inverse
            for step in reversed(completed_steps):
                if step["compensate"]:
                    await step["compensate"](context)

            return {"status": "COMPENSATED", "error": str(e)}
```

### Saga de souscription

```python
class SubscriptionSaga(SagaOrchestrator):
    def __init__(self, services):
        super().__init__()
        self.services = services

        # Étape 1: Valider le devis
        self.add_step(
            action=self._validate_quote,
            compensate=None  # Validation = pas de compensation
        )

        # Étape 2: Créer la police
        self.add_step(
            action=self._create_policy,
            compensate=self._cancel_policy
        )

        # Étape 3: Créer la facture
        self.add_step(
            action=self._create_invoice,
            compensate=self._cancel_invoice
        )

        # Étape 4: Générer les documents
        self.add_step(
            action=self._generate_documents,
            compensate=self._delete_documents
        )

        # Étape 5: Notifications (pas de compensation)
        self.add_step(
            action=self._send_notifications,
            compensate=None
        )

    async def _validate_quote(self, ctx):
        result = await self.services["quote"].validate(ctx["quote_id"])
        return {"validated": True}

    async def _create_policy(self, ctx):
        policy = await self.services["policy"].create({
            "customer_id": ctx["customer_id"],
            "product": ctx["product"]
        })
        return {"policy_id": policy.id}

    async def _cancel_policy(self, ctx):
        await self.services["policy"].cancel(ctx["policy_id"])

    async def _create_invoice(self, ctx):
        invoice = await self.services["billing"].create_invoice(ctx["policy_id"])
        return {"invoice_id": invoice.id}

    async def _cancel_invoice(self, ctx):
        await self.services["billing"].cancel_invoice(ctx["invoice_id"])

    # ... autres méthodes
```

## Flux d'exécution

### Scénario succès

```
1. validate_quote()      → ctx = {validated: true}
2. create_policy()       → ctx = {policy_id: "POL-001"}
3. create_invoice()      → ctx = {invoice_id: "INV-001"}
4. generate_documents()  → ctx = {doc_ids: ["D1", "D2"]}
5. send_notifications()  → ctx = {notified: true}

Résultat: COMPLETED
```

### Scénario échec avec compensation

```
1. validate_quote()      ✓
2. create_policy()       ✓ (policy_id: "POL-001")
3. create_invoice()      ✗ ERREUR!

Compensation (ordre inverse):
3. cancel_invoice()      - (rien à annuler)
2. cancel_policy()       ✓ Annule POL-001
1. (pas de compensation)

Résultat: COMPENSATED
```

## Gestion des états

### État de la Saga

```python
class SagaStatus(Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    COMPENSATING = "compensating"
    COMPENSATED = "compensated"
```

### Persistance pour reprise

```python
@dataclass
class SagaExecution:
    saga_id: str
    status: SagaStatus
    current_step: int
    steps_completed: List[str]
    steps_compensated: List[str]
    context: Dict
    error: Optional[str]
    started_at: datetime
    completed_at: Optional[datetime]
```

## Avantages

| Avantage | Description |
|----------|-------------|
| **Visibilité** | L'orchestrateur connaît l'état complet |
| **Contrôle** | Facile à implémenter les règles métier |
| **Debug** | Trace complète des étapes |
| **Résilience** | Reprise après panne possible |

## Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Couplage** | L'orchestrateur connaît tous les services |
| **SPOF** | Point unique de défaillance potentiel |
| **Synchrone** | Attente de chaque étape |

## Bonnes pratiques

### 1. Idempotence des compensations

```python
async def cancel_policy(self, ctx):
    policy = await self.db.get_policy(ctx["policy_id"])
    if policy and policy.status != "CANCELLED":
        await self.db.update_policy(ctx["policy_id"], {"status": "CANCELLED"})
    # Si déjà annulée, ne rien faire
```

### 2. Timeout par étape

```python
async def execute_step(self, step, ctx, timeout=30):
    try:
        return await asyncio.wait_for(
            step["action"](ctx),
            timeout=timeout
        )
    except asyncio.TimeoutError:
        raise SagaStepTimeout(step["name"])
```

### 3. Retry avant compensation

```python
async def execute_with_retry(self, step, ctx, max_retries=3):
    for attempt in range(max_retries):
        try:
            return await step["action"](ctx)
        except RetryableError:
            if attempt == max_retries - 1:
                raise
            await asyncio.sleep(2 ** attempt)
```

## Sandbox : Scénario EVT-04

Dans le scénario EVT-04, vous allez :
1. Définir les étapes d'une saga de souscription
2. Exécuter la saga avec succès
3. Simuler une panne à l'étape billing
4. Observer la compensation automatique
5. Analyser les logs du workflow
