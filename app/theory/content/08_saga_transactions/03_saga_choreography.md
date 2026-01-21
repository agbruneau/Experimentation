# 8.3 Saga Pattern - Chorégraphie

## Résumé

La **Saga Chorégraphie** distribue la coordination entre les services. Chaque service écoute des événements et réagit en émettant ses propres événements. Pas d'orchestrateur central.

### Points clés

- Pas de coordinateur central
- Chaque service connaît sa logique locale
- Communication par événements
- Découplage maximal

## Principe

```
┌─────────────────────────────────────────────────────────────┐
│                    SAGA CHORÉGRAPHIE                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Quote        Policy        Billing       Notif           │
│     │             │             │            │              │
│     │ QuoteApproved            │            │              │
│     └────────────►│            │            │              │
│                   │ PolicyCreated           │              │
│                   └────────────►│           │              │
│                                 │ InvoiceCreated           │
│                                 └───────────►│              │
│                                              │ Done         │
│                                                             │
│   En cas d'échec:                                          │
│                   ◄──────────────            │              │
│                   InvoiceFailed              │              │
│     ◄─────────────              PolicyCancelled            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Implémentation

### Chaque service écoute et réagit

```python
# Service Policy
class PolicyService:
    async def on_quote_approved(self, event):
        """Réagit à QuoteApproved."""
        policy = await self.create_policy(event.data)

        await self.broker.publish("PolicyCreated", {
            "policy_id": policy.id,
            "quote_id": event.data["quote_id"],
            "customer_id": event.data["customer_id"]
        })

    async def on_invoice_failed(self, event):
        """Réagit à InvoiceFailed - compensation."""
        await self.cancel_policy(event.data["policy_id"])

        await self.broker.publish("PolicyCancelled", {
            "policy_id": event.data["policy_id"],
            "reason": "Billing failed"
        })
```

```python
# Service Billing
class BillingService:
    async def on_policy_created(self, event):
        """Réagit à PolicyCreated."""
        try:
            invoice = await self.create_invoice(event.data["policy_id"])

            await self.broker.publish("InvoiceCreated", {
                "invoice_id": invoice.id,
                "policy_id": event.data["policy_id"]
            })
        except Exception as e:
            await self.broker.publish("InvoiceFailed", {
                "policy_id": event.data["policy_id"],
                "error": str(e)
            })
```

## Diagramme de séquence

### Succès

```
Quote      Policy     Billing    Notif     EventBus
  │          │          │          │          │
  │──QuoteApproved────────────────────────────►│
  │          │◄────────────────────────────────┤
  │          │                                 │
  │          │──PolicyCreated─────────────────►│
  │          │          │◄─────────────────────┤
  │          │          │                      │
  │          │          │──InvoiceCreated─────►│
  │          │          │          │◄──────────┤
  │          │          │          │           │
  │          │          │          │──NotificationSent─►
```

### Échec avec compensation

```
Quote      Policy     Billing    Notif     EventBus
  │          │          │          │          │
  │──QuoteApproved────────────────────────────►│
  │          │◄────────────────────────────────┤
  │          │                                 │
  │          │──PolicyCreated─────────────────►│
  │          │          │◄─────────────────────┤
  │          │          │                      │
  │          │          │──InvoiceFailed──────►│  ← Échec
  │          │◄────────────────────────────────┤
  │          │                                 │
  │          │──PolicyCancelled───────────────►│  ← Compensation
  │◄───────────────────────────────────────────┤
```

## Comparaison Orchestration vs Chorégraphie

| Aspect | Orchestration | Chorégraphie |
|--------|---------------|--------------|
| **Coordination** | Centralisée | Distribuée |
| **Couplage** | Services → Orchestrateur | Services → Événements |
| **Visibilité** | Facile (un point) | Difficile (distribué) |
| **SPOF** | Orchestrateur | Aucun |
| **Complexité** | Dans l'orchestrateur | Dans chaque service |
| **Scalabilité** | Moyenne | Haute |

## Avantages

| Avantage | Description |
|----------|-------------|
| **Découplage** | Services indépendants |
| **Scalabilité** | Chaque service scale indépendamment |
| **Résilience** | Pas de SPOF |
| **Évolutivité** | Ajouter un service = écouter les événements |

## Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Visibilité** | Difficile de suivre le flux complet |
| **Debug** | Tracer un problème est complexe |
| **Cyclic** | Risque de boucles d'événements |
| **Testing** | Tests d'intégration complexes |

## Quand choisir quoi ?

### Choisir Orchestration

- Workflow métier complexe avec beaucoup de branches
- Besoin de visibilité sur l'état du processus
- Équipe habituée aux processus centralisés
- Compensation complexe dépendant de l'état

### Choisir Chorégraphie

- Services très découplés
- Workflow simple et linéaire
- Haute scalabilité requise
- Équipe mature en event-driven

## Pattern hybride

Combiner les deux pour les cas complexes :

```
┌─────────────────────────────────────────────────────────────┐
│                    APPROCHE HYBRIDE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────┐           │
│  │         ORCHESTRATEUR (workflow complexe)    │           │
│  │  Quote → Policy → Billing                    │           │
│  └─────────────────┬───────────────────────────┘           │
│                    │                                        │
│                    │ PolicyReady (événement)                │
│                    ▼                                        │
│  ┌─────────────────────────────────────────────┐           │
│  │      CHORÉGRAPHIE (extensions découplées)    │           │
│  │  Documents ◄─┬─► Notifications               │           │
│  │              └─► Analytics                   │           │
│  └─────────────────────────────────────────────┘           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Bonnes pratiques

### 1. Événements idempotents

```python
async def on_policy_created(self, event):
    # Vérifier si déjà traité
    if await self.is_processed(event.id):
        return

    await self.process(event)
    await self.mark_processed(event.id)
```

### 2. Corrélation des événements

```python
# Tous les événements d'une même saga partagent un correlation_id
{
    "event_type": "PolicyCreated",
    "correlation_id": "SAGA-001",  # Permet de reconstituer le flux
    "causation_id": "QuoteApproved-001"  # Événement déclencheur
}
```

### 3. Timeout et détection de saga bloquée

```python
# Saga tracker pour détecter les blocages
class SagaTracker:
    async def check_stalled_sagas(self):
        sagas = await self.get_incomplete_sagas(older_than_minutes=30)
        for saga in sagas:
            await self.alert_stalled_saga(saga)
```
