# 7.5 Projections et Reconstruction d'État

## Résumé

Les **projections** transforment un flux d'événements en vues matérialisées optimisées pour des cas d'usage spécifiques. Elles sont au cœur de CQRS et Event Sourcing.

### Points clés

- Une projection = une vue construite à partir d'événements
- Plusieurs projections peuvent consommer les mêmes événements
- Reproductibles : on peut les reconstruire à tout moment
- Chaque projection est optimisée pour un usage spécifique

## Types de projections

### Projection simple (Single-stream)

Une entité = une projection.

```
Événements P001          Projection P001
─────────────────       ─────────────────
PolicyCreated    ──────► {id: P001, status: DRAFT}
PolicyActivated  ──────► {id: P001, status: ACTIVE}
PolicyModified   ──────► {id: P001, status: ACTIVE, premium: 900}
```

### Projection agrégée (Multi-stream)

Combine plusieurs flux d'événements.

```
Événements Policies      ┐
─────────────────────    │
PolicyCreated (x100)     │         Stats Projection
PolicyCancelled (x10)    ├────────► {total: 100,
                         │           active: 90,
Événements Claims        │           cancelled: 10}
─────────────────────    │
ClaimOpened (x50)        ┘
```

### Projection dénormalisée

Pré-calcule les jointures pour des requêtes rapides.

```
PolicyCreated + CustomerData
        ↓
┌─────────────────────────────────────────┐
│ {                                       │
│   policy_id: "P001",                    │
│   customer_name: "Jean Dupont",  ← Pré-jointure
│   customer_email: "jean@email.com",    │
│   product: "AUTO",                      │
│   premium: 850                          │
│ }                                       │
└─────────────────────────────────────────┘
```

## Implémentation

### Structure de base

```python
class Projection:
    """Interface de base pour les projections."""

    async def apply(self, event: Event):
        """Applique un événement à la projection."""
        handler = getattr(self, f"on_{event.type}", None)
        if handler:
            await handler(event)

    def get_state(self) -> Dict:
        """Retourne l'état actuel de la projection."""
        raise NotImplementedError
```

### Projection liste de polices

```python
class PolicyListProjection(Projection):
    def __init__(self):
        self.policies = {}

    async def on_PolicyCreated(self, event):
        self.policies[event.aggregate_id] = {
            "id": event.aggregate_id,
            "customer_id": event.data["customer_id"],
            "product": event.data["product"],
            "premium": event.data["premium"],
            "status": "DRAFT",
            "created_at": event.timestamp
        }

    async def on_PolicyActivated(self, event):
        if event.aggregate_id in self.policies:
            policy = self.policies[event.aggregate_id]
            policy["status"] = "ACTIVE"
            policy["start_date"] = event.data["start_date"]
            policy["end_date"] = event.data["end_date"]

    async def on_PolicyCancelled(self, event):
        if event.aggregate_id in self.policies:
            self.policies[event.aggregate_id]["status"] = "CANCELLED"

    def get_state(self):
        return list(self.policies.values())

    def get_by_customer(self, customer_id):
        return [p for p in self.policies.values()
                if p["customer_id"] == customer_id]
```

### Projection statistiques

```python
class PolicyStatsProjection(Projection):
    def __init__(self):
        self.stats = {
            "total": 0,
            "by_status": {},
            "by_product": {},
            "total_premium": 0
        }

    async def on_PolicyCreated(self, event):
        self.stats["total"] += 1
        self.stats["total_premium"] += event.data.get("premium", 0)

        product = event.data.get("product", "UNKNOWN")
        self.stats["by_product"][product] = \
            self.stats["by_product"].get(product, 0) + 1

        self.stats["by_status"]["DRAFT"] = \
            self.stats["by_status"].get("DRAFT", 0) + 1

    async def on_PolicyActivated(self, event):
        self.stats["by_status"]["DRAFT"] = \
            max(0, self.stats["by_status"].get("DRAFT", 0) - 1)
        self.stats["by_status"]["ACTIVE"] = \
            self.stats["by_status"].get("ACTIVE", 0) + 1

    def get_state(self):
        return self.stats
```

## Reconstruction (Rebuild)

### Pourquoi reconstruire ?

- Nouvelle projection créée
- Bug corrigé dans une projection
- Données corrompues
- Migration de schéma

### Processus

```
1. Arrêter la projection
2. Vider l'état actuel
3. Rejouer tous les événements depuis le début
4. Redémarrer la projection
```

### Implémentation

```python
class ProjectionManager:
    def __init__(self, event_store, projection):
        self.event_store = event_store
        self.projection = projection
        self.last_position = 0

    async def rebuild(self):
        """Reconstruit la projection depuis le début."""
        # Vider l'état
        self.projection.reset()
        self.last_position = 0

        # Rejouer tous les événements
        events = await self.event_store.get_all_events()
        for event in events:
            await self.projection.apply(event)
            self.last_position = event.position

        return self.last_position

    async def catch_up(self):
        """Rattrape les nouveaux événements."""
        events = await self.event_store.get_events_after(self.last_position)
        for event in events:
            await self.projection.apply(event)
            self.last_position = event.position
```

## Patterns avancés

### Projection avec fenêtre temporelle

```python
class MonthlyStatsProjection(Projection):
    def __init__(self):
        self.monthly_stats = {}  # {month: stats}

    async def on_PolicyCreated(self, event):
        month = event.timestamp[:7]  # "2024-01"
        if month not in self.monthly_stats:
            self.monthly_stats[month] = {"count": 0, "premium": 0}

        self.monthly_stats[month]["count"] += 1
        self.monthly_stats[month]["premium"] += event.data.get("premium", 0)

    def get_month(self, month):
        return self.monthly_stats.get(month, {"count": 0, "premium": 0})
```

### Projection composite

```python
class CustomerDashboardProjection(Projection):
    """Vue composite pour le dashboard client."""

    def __init__(self):
        self.customers = {}

    async def on_CustomerCreated(self, event):
        self.customers[event.aggregate_id] = {
            "id": event.aggregate_id,
            "name": event.data["name"],
            "policies": [],
            "claims": [],
            "total_premium": 0
        }

    async def on_PolicyCreated(self, event):
        customer_id = event.data["customer_id"]
        if customer_id in self.customers:
            self.customers[customer_id]["policies"].append({
                "id": event.aggregate_id,
                "product": event.data["product"],
                "premium": event.data["premium"]
            })
            self.customers[customer_id]["total_premium"] += \
                event.data.get("premium", 0)

    async def on_ClaimOpened(self, event):
        # Trouver le client via la police
        policy_id = event.data["policy_id"]
        # ... ajouter le claim au dashboard
```

## Bonnes pratiques

### 1. Idempotence

```python
async def on_PolicyCreated(self, event):
    # Vérifier si déjà traité
    if event.aggregate_id in self.policies:
        return  # Ignorer le doublon

    self.policies[event.aggregate_id] = {...}
```

### 2. Versioning des projections

```python
class PolicyListProjectionV2(Projection):
    VERSION = 2  # Incrémenter quand le schéma change

    def get_schema_version(self):
        return self.VERSION
```

### 3. Monitoring

```python
class MonitoredProjection(Projection):
    def __init__(self, inner):
        self.inner = inner
        self.events_processed = 0
        self.last_event_time = None

    async def apply(self, event):
        start = time.time()
        await self.inner.apply(event)
        duration = time.time() - start

        self.events_processed += 1
        self.last_event_time = event.timestamp

        metrics.record("projection.latency", duration)
        metrics.increment("projection.events_processed")
```

## Stockage des projections

| Type | Usage | Exemple |
|------|-------|---------|
| In-memory | Dev/Test, petit volume | Dict Python |
| SQL | Requêtes complexes | PostgreSQL |
| Document | Flexibilité schéma | MongoDB |
| Cache | Haute performance | Redis |
| Search | Full-text search | Elasticsearch |

## Points d'attention

1. **Cohérence éventuelle** : Les projections sont asynchrones
2. **Ordre des événements** : Important pour certaines projections
3. **Performance rebuild** : Peut être long avec beaucoup d'événements
4. **Stockage** : Chaque projection a ses propres besoins
