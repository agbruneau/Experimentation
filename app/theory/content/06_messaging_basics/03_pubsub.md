# 6.3 Publish/Subscribe (Topics)

## Résumé

Le pattern **Publish/Subscribe** (Pub/Sub) permet à un producteur de diffuser un message à **plusieurs consommateurs** simultanément. Contrairement aux queues, tous les abonnés reçoivent chaque message.

### Points clés

- Un message est diffusé à tous les abonnés du topic
- Les producteurs ne connaissent pas les consommateurs
- Découplage fort entre producteurs et consommateurs
- Idéal pour les notifications et événements métier

## Fonctionnement

```
                    ┌─────────────────┐
                 ┌─►│  Consommateur 1 │
                 │  └─────────────────┘
┌────────────┐   │  ┌─────────────────┐
│ Producteur │──►│─►│  Consommateur 2 │
└────────────┘   │  └─────────────────┘
       │         │  ┌─────────────────┐
       │         └─►│  Consommateur 3 │
       ▼            └─────────────────┘
  topic.events
```

### Différences avec les Queues

| Aspect | Queue | Topic (Pub/Sub) |
|--------|-------|-----------------|
| Consommation | Un seul | Tous les abonnés |
| Objectif | Distribuer le travail | Diffuser l'information |
| Couplage | Producteur → Consommateur | Producteur → Topic |
| Persistance | Jusqu'à consommation | Variable selon config |

## Types d'abonnements

### Abonnement simple

Chaque abonné reçoit tous les messages du topic.

```
Topic: policies.events
  │
  ├──► Billing (reçoit tout)
  ├──► Notifications (reçoit tout)
  └──► Audit (reçoit tout)
```

### Abonnement avec filtre

Les abonnés peuvent filtrer les messages selon des critères.

```
Topic: policies.events
  │
  ├──► Billing (filtre: type=CREATED or type=RENEWED)
  ├──► Notifications (filtre: type=CREATED)
  └──► Audit (pas de filtre)
```

### Consumer Groups

Combinaison de Pub/Sub et Competing Consumers :

```
Topic: policies.events
  │
  ├──► Group: billing-service
  │      ├── Instance 1 (partage les messages)
  │      └── Instance 2
  │
  └──► Group: notification-service
         ├── Instance 1 (partage les messages)
         └── Instance 2
```

Chaque groupe reçoit tous les messages, mais au sein d'un groupe, chaque message n'est traité que par une instance.

## Événements métier en Assurance

### Exemple : Création de police

```
PolicyAdmin publie: PolicyCreated
  │
  ├──► Billing
  │      └─ Crée la première facture
  │
  ├──► Notifications
  │      └─ Envoie email de bienvenue
  │
  ├──► Documents
  │      └─ Génère les conditions générales
  │
  └──► Audit
         └─ Enregistre la trace
```

### Structure d'événement recommandée

```json
{
  "event_type": "PolicyCreated",
  "event_id": "evt-123456",
  "timestamp": "2024-01-15T10:30:00Z",
  "source": "policy-admin",
  "correlation_id": "req-789",
  "data": {
    "policy_number": "POL-2024-001",
    "customer_id": "C001",
    "product": "AUTO",
    "premium": 850.00
  }
}
```

## Patterns de publication

### Event Notification

Message léger indiquant qu'un événement s'est produit :

```json
{
  "event_type": "PolicyCreated",
  "policy_id": "POL-2024-001",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Les consommateurs doivent appeler l'API pour obtenir les détails.

**Avantages** : Messages petits, données toujours fraîches
**Inconvénients** : Charge sur l'API source, couplage temporel

### Event-Carried State Transfer

Message contenant toutes les données nécessaires :

```json
{
  "event_type": "PolicyCreated",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "policy_number": "POL-2024-001",
    "customer_id": "C001",
    "customer_name": "Jean Dupont",
    "product": "AUTO",
    "premium": 850.00,
    "start_date": "2024-02-01",
    "coverages": ["RC", "VOL", "BRIS_GLACE"]
  }
}
```

**Avantages** : Autonomie des consommateurs, pas de callbacks
**Inconvénients** : Messages plus gros, données potentiellement obsolètes

## Implémentation pseudo-code

```python
# Publication d'un événement
async def create_policy(policy_data):
    # 1. Créer la police en base
    policy = await db.insert_policy(policy_data)

    # 2. Publier l'événement
    await broker.publish(
        "policies.events",
        {
            "event_type": "PolicyCreated",
            "event_id": generate_id(),
            "timestamp": now(),
            "data": {
                "policy_number": policy.number,
                "customer_id": policy.customer_id,
                "product": policy.product,
                "premium": policy.premium
            }
        }
    )

    return policy

# Abonnement (Billing)
async def billing_handler(event):
    if event["event_type"] == "PolicyCreated":
        await create_initial_invoice(
            event["data"]["policy_number"],
            event["data"]["premium"]
        )

await broker.subscribe("policies.events", billing_handler)

# Abonnement (Notifications)
async def notification_handler(event):
    if event["event_type"] == "PolicyCreated":
        await send_welcome_email(event["data"]["customer_id"])

await broker.subscribe("policies.events", notification_handler)
```

## Bonnes pratiques

| Pratique | Description |
|----------|-------------|
| Nommage cohérent | `domaine.entité.action` (ex: `policies.created`) |
| Versionning | Inclure la version du schéma |
| Idempotence | Les handlers doivent être idempotents |
| Schema registry | Documenter et valider les schémas |
| Ordre non garanti | Ne pas dépendre de l'ordre entre topics |

## Sandbox : Suite EVT-01

Dans le scénario EVT-01, vous allez publier un événement `PolicyCreated` et observer comment plusieurs services (Billing, Notifications, Audit) le reçoivent simultanément.
