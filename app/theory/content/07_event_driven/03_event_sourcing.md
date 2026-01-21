# 7.3 Event Sourcing

## Résumé

L'**Event Sourcing** est un pattern où l'état d'une entité est stocké comme une **séquence d'événements** plutôt que comme un état courant. L'état actuel est reconstruit en rejouant tous les événements.

### Points clés

- L'état = somme des événements passés
- Append-only : on ne modifie jamais, on ajoute
- Replay : on peut reconstruire n'importe quel état passé
- Audit trail complet et natif

## Principe fondamental

### Approche traditionnelle (CRUD)

```
┌─────────────────────────────────────────┐
│            TABLE POLICIES               │
├─────────────────────────────────────────┤
│ id   │ status │ premium │ updated_at    │
│ P001 │ ACTIVE │ 900     │ 2024-03-15    │
└─────────────────────────────────────────┘
         Seul l'état actuel est connu
         L'historique est perdu
```

### Approche Event Sourcing

```
┌─────────────────────────────────────────────────────────────┐
│                    EVENT STORE                               │
├─────────────────────────────────────────────────────────────┤
│ event_id │ aggregate_id │ type            │ data            │
│ EVT-001  │ P001         │ PolicyCreated   │ {premium: 850}  │
│ EVT-002  │ P001         │ PolicyActivated │ {start: ...}    │
│ EVT-003  │ P001         │ PolicyModified  │ {premium: 900}  │
└─────────────────────────────────────────────────────────────┘
         Tout l'historique est préservé
         État = replay de tous les événements
```

## Reconstruction de l'état

```python
# Événements stockés pour la police P001
events = [
    PolicyCreated(premium=850, status="DRAFT"),
    PolicyActivated(start_date="2024-01-01"),
    PolicyModified(premium=900),
    PolicyModified(premium=950)
]

# Reconstruction
state = {}
for event in events:
    state = apply(state, event)

# Résultat: {premium: 950, status: "ACTIVE", start_date: "2024-01-01"}
```

### Le Reducer (fonction d'application)

```python
def policy_reducer(state, event):
    new_state = state.copy()

    if event.type == "PolicyCreated":
        return {
            "policy_id": event.data["policy_id"],
            "premium": event.data["premium"],
            "status": "DRAFT"
        }

    elif event.type == "PolicyActivated":
        new_state["status"] = "ACTIVE"
        new_state["start_date"] = event.data["start_date"]

    elif event.type == "PolicyModified":
        new_state.update(event.data)

    elif event.type == "PolicyCancelled":
        new_state["status"] = "CANCELLED"

    return new_state
```

## Avantages

### 1. Audit trail natif

```
Qui a fait quoi, quand ?
└── EVT-001: PolicyCreated by user_123 at 2024-01-01
└── EVT-002: PolicyActivated by user_456 at 2024-01-05
└── EVT-003: PremiumModified by user_123 at 2024-02-15
```

### 2. Voyage dans le temps

```python
# État au 1er janvier
state_jan = rebuild_state(events, to_date="2024-01-01")

# État au 15 février
state_feb = rebuild_state(events, to_date="2024-02-15")

# État actuel
state_now = rebuild_state(events)
```

### 3. Debug et analyse

```python
# Pourquoi la prime est-elle de 950 ?
premium_events = filter(events, type="PolicyModified")
# → On voit l'historique des changements
```

### 4. Correction d'erreurs

```python
# Annuler un événement erroné
events.append(PremiumCorrected(premium=850, reason="Erreur de saisie"))
# L'événement erroné reste dans l'historique (traçabilité)
# Mais l'état final est corrigé
```

## Implémentation

### Structure de l'Event Store

```python
class EventStore:
    def append(self, aggregate_id, event):
        """Ajoute un événement (append-only)."""

    def get_events(self, aggregate_id, from_version=0):
        """Récupère les événements d'un agrégat."""

    def rebuild_state(self, aggregate_id, reducer):
        """Reconstruit l'état en rejouant les événements."""
```

### Exemple complet

```python
# 1. Créer une police
event_store.append("P001", {
    "type": "PolicyCreated",
    "data": {
        "customer_id": "C001",
        "product": "AUTO",
        "premium": 850
    }
})

# 2. Activer la police
event_store.append("P001", {
    "type": "PolicyActivated",
    "data": {
        "start_date": "2024-02-01",
        "end_date": "2025-01-31"
    }
})

# 3. Modifier la prime
event_store.append("P001", {
    "type": "PolicyModified",
    "data": {"premium": 900}
})

# 4. Reconstruire l'état actuel
current_state = event_store.rebuild_state("P001", policy_reducer)
# {customer_id: "C001", product: "AUTO", premium: 900, status: "ACTIVE", ...}

# 5. Reconstruire l'état avant la modification
past_state = event_store.rebuild_state("P001", policy_reducer, to_version=2)
# {customer_id: "C001", product: "AUTO", premium: 850, status: "ACTIVE", ...}
```

## Optimisations

### Snapshots

Reconstruire depuis zéro peut être lent pour les agrégats avec beaucoup d'événements.

```
┌──────────────────────────────────────────────────┐
│                    Avec Snapshot                  │
├──────────────────────────────────────────────────┤
│  Snapshot@v100 ─► [EVT-101] [EVT-102] [EVT-103]  │
│                                                   │
│  Au lieu de rejouer 103 événements,              │
│  on part du snapshot et rejoue 3                 │
└──────────────────────────────────────────────────┘
```

```python
# Créer un snapshot périodiquement
if event_count % 100 == 0:
    state = rebuild_state(aggregate_id)
    save_snapshot(aggregate_id, state, version=event_count)

# Reconstruction optimisée
def rebuild_with_snapshot(aggregate_id):
    snapshot = get_snapshot(aggregate_id)
    events = get_events(aggregate_id, from_version=snapshot.version + 1)
    return apply_events(snapshot.state, events)
```

## Considérations

### Complexité du schéma

Les événements sont immuables, mais le schéma évolue :

```python
# Version 1
{"type": "PolicyCreated", "premium": 850}

# Version 2 (ajout de champ)
{"type": "PolicyCreated", "premium": 850, "currency": "EUR"}

# Solution : Upcasting
def upcast_v1_to_v2(event):
    if event.version == 1:
        event.data["currency"] = "EUR"  # Valeur par défaut
    return event
```

### Quand utiliser

| ✅ Adapté | ❌ Moins adapté |
|----------|-----------------|
| Domaines complexes avec historique important | CRUD simple |
| Besoin d'audit réglementaire | Données éphémères |
| Systèmes financiers | Haute fréquence de lecture |
| Workflow métier complexe | Données volumineuses par entité |

## Sandbox : Scénario EVT-03

Dans le scénario EVT-03, vous allez :
1. Créer une police avec PolicyCreated
2. La modifier plusieurs fois
3. Visualiser le journal d'événements
4. Reconstruire l'état à différents moments
5. Comprendre le "voyage dans le temps"
