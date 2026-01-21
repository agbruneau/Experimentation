# 7.4 CQRS - Command Query Responsibility Segregation

## Résumé

**CQRS** sépare le modèle utilisé pour les **écritures** (commandes) de celui utilisé pour les **lectures** (requêtes). Chaque modèle est optimisé pour son usage.

### Points clés

- Séparation entre modèle de commande et modèle de requête
- Optimisation indépendante de chaque côté
- Souvent combiné avec Event Sourcing
- Les projections maintiennent le modèle de lecture

## Principe

### Architecture traditionnelle

```
┌────────────────────────────────────────────┐
│              MÊME MODÈLE                    │
│                                            │
│  UI ──► Service ──► Database ──► UI        │
│         (même structure pour tout)         │
└────────────────────────────────────────────┘
```

### Architecture CQRS

```
┌─────────────────────────────────────────────────────────────┐
│                         CQRS                                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  WRITE SIDE                        READ SIDE                │
│  (Commandes)                       (Requêtes)               │
│                                                             │
│  ┌─────────┐                       ┌─────────┐              │
│  │ Command │                       │  Query  │              │
│  │ Handler │                       │ Handler │              │
│  └────┬────┘                       └────┬────┘              │
│       │                                 │                   │
│       ▼                                 ▼                   │
│  ┌─────────┐    Projection        ┌─────────┐              │
│  │ Event   │───────────────────►  │  Read   │              │
│  │ Store   │      (async)         │ Model   │              │
│  └─────────┘                      └─────────┘              │
│                                                             │
│  Optimisé pour                    Optimisé pour            │
│  la cohérence                     la performance           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Commandes (Write Side)

### Caractéristiques

- Expriment une **intention** de modifier l'état
- Nommées à l'impératif (CreatePolicy, CancelClaim)
- Validées avant exécution
- Produisent des événements

### Structure d'une commande

```python
@dataclass
class CreatePolicyCommand:
    command_id: str
    customer_id: str
    product: str
    premium: float
    coverages: List[str]
    correlation_id: str
```

### Handler de commande

```python
class PolicyCommandHandler:
    def handle(self, command: CreatePolicyCommand):
        # 1. Valider
        if not self.validate(command):
            raise ValidationError("Invalid command")

        # 2. Charger l'agrégat (si modification)
        # policy = self.repository.load(command.policy_id)

        # 3. Exécuter la logique métier
        policy_id = self.generate_policy_id()

        # 4. Produire les événements
        event = PolicyCreated(
            policy_id=policy_id,
            customer_id=command.customer_id,
            product=command.product,
            premium=command.premium
        )

        # 5. Persister
        self.event_store.append(policy_id, event)

        return {"policy_id": policy_id}
```

## Requêtes (Read Side)

### Caractéristiques

- N'expriment **aucune intention de modification**
- Nommées comme des questions (GetPolicy, ListPoliciesByCustomer)
- Retournent des données sans effet de bord
- Optimisées pour la performance

### Structure d'une requête

```python
@dataclass
class GetPolicyQuery:
    policy_id: str

@dataclass
class ListPoliciesByCustomerQuery:
    customer_id: str
    status: Optional[str] = None
```

### Handler de requête

```python
class PolicyQueryHandler:
    def handle(self, query: GetPolicyQuery):
        # Lecture directe depuis le read model
        return self.read_model.get_policy(query.policy_id)

    def handle(self, query: ListPoliciesByCustomerQuery):
        return self.read_model.list_by_customer(
            query.customer_id,
            status=query.status
        )
```

## Projections

### Rôle

Les projections transforment les événements en vues optimisées pour la lecture.

```
┌─────────────────────────────────────────────────────────────┐
│                      PROJECTIONS                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Event Stream                     Read Models               │
│                                                             │
│  PolicyCreated ──────────────►  ┌─────────────────────┐    │
│  PolicyActivated ─────────────► │ PolicyListView      │    │
│  PolicyModified ──────────────► │ (toutes les polices)│    │
│                                 └─────────────────────┘    │
│                                                             │
│  PolicyCreated ──────────────►  ┌─────────────────────┐    │
│  PolicyActivated ─────────────► │ PolicyByCustomerView│    │
│                                 │ (polices par client)│    │
│                                 └─────────────────────┘    │
│                                                             │
│  PolicyCreated ──────────────►  ┌─────────────────────┐    │
│  PolicyCancelled ────────────►  │ PolicyStatsView     │    │
│                                 │ (statistiques)      │    │
│                                 └─────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Implémentation d'une projection

```python
class PolicyListProjection:
    def __init__(self):
        self.policies = {}  # Ou base de données dédiée

    def apply(self, event: Event):
        if event.type == "PolicyCreated":
            self.policies[event.aggregate_id] = {
                "id": event.aggregate_id,
                "customer_id": event.data["customer_id"],
                "status": "DRAFT",
                "premium": event.data["premium"]
            }

        elif event.type == "PolicyActivated":
            if event.aggregate_id in self.policies:
                self.policies[event.aggregate_id]["status"] = "ACTIVE"

        elif event.type == "PolicyCancelled":
            if event.aggregate_id in self.policies:
                self.policies[event.aggregate_id]["status"] = "CANCELLED"

    def get_all(self):
        return list(self.policies.values())

    def get_by_status(self, status):
        return [p for p in self.policies.values() if p["status"] == status]
```

## Avantages du CQRS

### 1. Optimisation indépendante

| Write Side | Read Side |
|------------|-----------|
| Modèle normalisé | Modèles dénormalisés |
| Transactions ACID | Requêtes rapides |
| Validation stricte | Cache agressif |
| Une base | Plusieurs bases possibles |

### 2. Scalabilité

```
Write Side: 1 instance (cohérence)
Read Side: N instances (scaling horizontal)
```

### 3. Flexibilité des vues

```
Besoin d'une nouvelle vue ?
→ Créer une nouvelle projection
→ Rejouer les événements
→ Vue prête sans modifier le write side
```

## Cohérence éventuelle

### Le défi

L'écriture et la lecture sont asynchrones :

```
T0: Commande reçue
T1: Événement persisté
T2: Projection mise à jour  ← Délai
T3: Lecture reflète le changement
```

### Solutions

1. **UI optimiste** : Montrer le changement attendu avant confirmation
2. **Polling/SSE** : Notifier quand la vue est à jour
3. **Read-your-writes** : Garantir qu'un utilisateur voit ses propres modifications

## Quand utiliser CQRS

| ✅ Adapté | ❌ Éviter |
|----------|----------|
| Domaine complexe | CRUD simple |
| Besoins de lecture variés | Même vue partout |
| Scaling différencié | Petite application |
| Event Sourcing | Cohérence forte requise |

## Sandbox : Scénario EVT-05

Dans le scénario EVT-05, vous allez :
1. Envoyer des commandes de création/modification de polices
2. Observer la mise à jour asynchrone des projections
3. Exécuter des requêtes sur le modèle de lecture
4. Comparer les performances read vs write
