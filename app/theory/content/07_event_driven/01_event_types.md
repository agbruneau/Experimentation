# 7.1 Événements Métier vs Techniques

## Résumé

Dans une architecture événementielle, il est crucial de distinguer les **événements métier** (domain events) des **événements techniques** (infrastructure events). Cette distinction impacte la conception, le nommage et le traitement.

### Points clés

- Les événements métier représentent des faits significatifs du domaine
- Les événements techniques concernent l'infrastructure
- Le nommage doit être clair et au passé
- La granularité dépend du contexte

## Événements Métier (Domain Events)

### Définition

Un événement métier représente **quelque chose de significatif qui s'est produit** dans le domaine d'activité. C'est un fait immuable, exprimé au passé.

### Caractéristiques

| Aspect | Description |
|--------|-------------|
| **Temps** | Passé (ex: PolicyCreated, ClaimSubmitted) |
| **Langage** | Ubiquitaire (termes métier) |
| **Immutabilité** | Ne peut pas être modifié, seulement annulé |
| **Autonomie** | Compréhensible sans contexte technique |

### Exemples en Assurance

```
CYCLE DE VIE D'UNE POLICE
├── QuoteRequested
├── QuoteCalculated
├── PolicyCreated
├── PolicyActivated
├── PolicyModified
├── PolicyRenewed
└── PolicyCancelled

CYCLE DE VIE D'UN SINISTRE
├── ClaimOpened
├── ClaimDocumentsReceived
├── ClaimAssessed
├── ClaimApproved / ClaimRejected
├── ClaimPaid
└── ClaimClosed

FACTURATION
├── InvoiceGenerated
├── PaymentReceived
├── PaymentFailed
└── InvoiceOverdue
```

### Structure recommandée

```json
{
  "event_type": "PolicyCreated",
  "event_id": "EVT-001",
  "timestamp": "2024-01-15T10:30:00Z",
  "aggregate_id": "POL-2024-001",
  "aggregate_type": "Policy",
  "version": 1,
  "data": {
    "policy_number": "POL-2024-001",
    "customer_id": "C001",
    "product": "AUTO",
    "premium": 850.00
  },
  "metadata": {
    "user_id": "USR-123",
    "correlation_id": "REQ-456",
    "causation_id": "CMD-789"
  }
}
```

## Événements Techniques (Infrastructure Events)

### Définition

Les événements techniques concernent le fonctionnement de l'infrastructure et sont généralement invisibles du métier.

### Exemples

```
INFRASTRUCTURE
├── ServiceStarted
├── ServiceStopped
├── HealthCheckFailed
├── CircuitBreakerOpened
└── CircuitBreakerClosed

MESSAGING
├── MessagePublished
├── MessageConsumed
├── MessageRetried
├── MessageMovedToDLQ
└── QueuePurged

BASE DE DONNÉES
├── ConnectionPoolExhausted
├── SlowQueryDetected
├── ReplicationLagDetected
└── BackupCompleted
```

### Quand les utiliser

- Monitoring et alerting
- Debugging et troubleshooting
- Audit technique
- Auto-healing

## Comparaison

| Aspect | Événement Métier | Événement Technique |
|--------|------------------|---------------------|
| **Audience** | Équipe métier + tech | Équipe tech uniquement |
| **Durée de vie** | Longue (années) | Courte (jours/semaines) |
| **Stockage** | Event Store permanent | Logs/Métriques |
| **Schéma** | Versionné, stable | Flexible |
| **Exemples** | PolicyCreated | CircuitBreakerTripped |

## Bonnes pratiques de nommage

### Pour les événements métier

| ✅ Bon | ❌ Éviter |
|--------|----------|
| PolicyCreated | CreatePolicy |
| ClaimSubmitted | ClaimSubmit |
| PaymentReceived | Payment |
| CustomerAddressChanged | UpdateAddress |

### Règles

1. **Passé composé** : L'événement est un fait accompli
2. **Sujet + Action** : ClaimAssessed, InvoiceGenerated
3. **Spécifique** : CustomerEmailUpdated plutôt que CustomerUpdated
4. **Sans ambiguïté** : PolicyCancelled, pas PolicyStopped

## Granularité des événements

### Événement fin (fine-grained)

```
CustomerFirstNameChanged
CustomerLastNameChanged
CustomerEmailChanged
CustomerPhoneChanged
```

**Avantages** : Précision, traçabilité fine
**Inconvénients** : Volume, complexité

### Événement agrégé (coarse-grained)

```
CustomerProfileUpdated {
  changes: ["firstName", "email"]
}
```

**Avantages** : Simplicité, moins de messages
**Inconvénients** : Moins de détails, handlers plus complexes

### Recommandation

- **Événements de cycle de vie** : Granularité fine (PolicyCreated, PolicyActivated)
- **Modifications de données** : Dépend du besoin de traçabilité
- **Notifications** : Granularité moyenne
