# Distributed Tracing

## Résumé

Le **Distributed Tracing** permet de suivre une requête à travers tous les services qu'elle traverse. Chaque opération est représentée par un **span**, et l'ensemble forme une **trace**.

## Points clés

- **Trace** : Représente une requête complète de bout en bout
- **Span** : Une opération individuelle dans la trace
- **Contexte propagé** : Le trace_id est transmis entre services
- **Visualisation en cascade** : Vue temporelle des appels

## Concepts fondamentaux

### Trace
```
Trace: Création d'un devis complet
├── ID: trace-abc123
├── Début: 14:32:05.000
├── Fin: 14:32:05.450
└── Durée totale: 450ms
```

### Span
```
Span: Appel au Rating API
├── ID: span-xyz789
├── Trace ID: trace-abc123
├── Parent ID: span-def456
├── Service: quote-engine
├── Opération: external_rating.calculate
├── Début: 14:32:05.100
├── Fin: 14:32:05.350
├── Durée: 250ms
└── Tags: {product: "AUTO", customer: "C001"}
```

## Anatomie d'une trace

```
Trace ID: abc123
Time ─────────────────────────────────────────────────────▶

Gateway         [══════════════════════════════════════]  450ms
                        │
Quote Engine            [══════════════════════════]      350ms
                               │           │
Customer Hub                   [════]                      50ms
                                      │
External Rating                       [════════════]      200ms
```

### Représentation JSON
```json
{
  "trace_id": "abc123",
  "spans": [
    {
      "span_id": "span-001",
      "parent_span_id": null,
      "operation": "POST /quotes",
      "service": "gateway",
      "start_time": "14:32:05.000",
      "duration_ms": 450
    },
    {
      "span_id": "span-002",
      "parent_span_id": "span-001",
      "operation": "create_quote",
      "service": "quote-engine",
      "start_time": "14:32:05.050",
      "duration_ms": 350
    },
    {
      "span_id": "span-003",
      "parent_span_id": "span-002",
      "operation": "get_customer",
      "service": "customer-hub",
      "start_time": "14:32:05.100",
      "duration_ms": 50
    },
    {
      "span_id": "span-004",
      "parent_span_id": "span-002",
      "operation": "calculate_rate",
      "service": "external-rating",
      "start_time": "14:32:05.160",
      "duration_ms": 200
    }
  ]
}
```

## Propagation du contexte

```
┌─────────┐     Headers      ┌─────────┐     Headers      ┌─────────┐
│ Gateway │ ─────────────────▶│ Quote   │ ─────────────────▶│ Rating  │
│         │  X-Trace-ID:abc  │ Engine  │  X-Trace-ID:abc  │   API   │
│         │  X-Span-ID:001   │         │  X-Span-ID:002   │         │
└─────────┘                  └─────────┘                  └─────────┘
```

### Headers standards
```
X-Trace-ID: abc123          # Identifiant de la trace
X-Span-ID: span-002         # Span courant
X-Parent-Span-ID: span-001  # Span parent
```

## Pseudo-code

```python
class Tracer:
    def start_span(self, operation_name, parent=None):
        span = Span(
            span_id=generate_id(),
            trace_id=parent.trace_id if parent else generate_id(),
            parent_span_id=parent.span_id if parent else None,
            operation=operation_name,
            start_time=now()
        )
        return span

    def finish_span(self, span):
        span.end_time = now()
        span.duration = span.end_time - span.start_time
        self.export(span)

# Usage
with tracer.span("process_quote") as span:
    span.set_tag("customer_id", "C001")
    customer = get_customer(customer_id)

    with tracer.span("get_rate", parent=span) as child:
        rate = rating_api.calculate(customer)
```

## Cas d'usage assurance

### Trace de souscription complète

```
Souscription d'une police AUTO
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Gateway          [═══════════════════════════════════]  2.1s
                 POST /api/subscriptions

Quote Engine         [═══════════════]                  800ms
                     create_quote
                         │
Customer Hub                 [══]                        50ms
                             get_customer
                                 │
External Rating                  [═════════]            500ms
                                 calculate_rate
                                        │
Policy Admin             [════════════════]             900ms
                         create_policy
                                 │
Billing                          [══════]               400ms
                                 create_invoice
                                        │
Notification                            [═══]           200ms
                                        send_email
```

### Identification des problèmes

**Problème 1 : Service lent**
```
External Rating: 500ms → 70% du temps total de Quote Engine
Action: Optimiser ou mettre en cache
```

**Problème 2 : Appels séquentiels**
```
Policy Admin attend Quote Engine
Billing attend Policy Admin
→ Possibilité de paralléliser ?
```

## Tags et annotations

### Tags (métadonnées statiques)
```json
{
  "service": "quote-engine",
  "version": "2.3.1",
  "environment": "production",
  "customer_id": "C001",
  "product": "AUTO"
}
```

### Logs/Events (événements temporels)
```json
{
  "spans": [{
    "span_id": "span-002",
    "logs": [
      {"timestamp": "14:32:05.100", "event": "cache_miss"},
      {"timestamp": "14:32:05.150", "event": "fallback_used", "reason": "timeout"}
    ]
  }]
}
```

### Erreurs
```json
{
  "span_id": "span-004",
  "status": "ERROR",
  "tags": {
    "error": true,
    "error.type": "TimeoutError",
    "error.message": "Rating API timeout after 5s"
  }
}
```

## Sampling

Pour éviter de tracer 100% des requêtes en production :

| Stratégie | Description | Usage |
|-----------|-------------|-------|
| Probabiliste | 10% des requêtes | Trafic élevé |
| Rate limiting | 100 traces/min max | Contrôle du volume |
| Par erreur | 100% si erreur | Debug |
| Par durée | Si > 1s | Performance |

```python
def should_sample(request):
    # Toujours tracer les erreurs
    if request.has_error:
        return True
    # Toujours tracer les requêtes lentes
    if request.duration > 1000:
        return True
    # Sinon, 10% aléatoire
    return random() < 0.1
```

## Bonnes pratiques

1. **Nommer les spans clairement** : `GET /customers/{id}` pas `span1`
2. **Ajouter du contexte métier** : customer_id, policy_number
3. **Propager le contexte** : Toujours transmettre les headers
4. **Instrumenter les opérations critiques** : API, DB, externes
5. **Sampler intelligemment** : Pas besoin de 100% en prod
6. **Corréler avec les logs** : Même trace_id partout
