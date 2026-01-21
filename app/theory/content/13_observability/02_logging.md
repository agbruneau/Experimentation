# Logging Structuré

## Résumé

Le **logging structuré** consiste à émettre des logs dans un format parsable (JSON) plutôt qu'en texte libre. Cela permet la recherche, l'agrégation et l'analyse automatique des logs.

## Points clés

- **Format JSON** : Facilite le parsing et l'indexation
- **Champs standardisés** : timestamp, level, message, service, trace_id
- **Contexte riche** : Attributs métier pour le filtrage
- **Corrélation** : Lien avec traces et métriques via trace_id

## Log non-structuré vs structuré

### Non-structuré (mauvais)
```
2024-03-15 14:32:05 ERROR Quote creation failed for customer C001: timeout
```

Problèmes :
- Difficile à parser
- Format variable
- Pas de filtrage possible

### Structuré (bon)
```json
{
  "timestamp": "2024-03-15T14:32:05.123Z",
  "level": "ERROR",
  "service": "quote-engine",
  "message": "Quote creation failed",
  "trace_id": "abc123def456",
  "span_id": "789xyz",
  "customer_id": "C001",
  "error_type": "TimeoutError",
  "error_message": "External rating API timeout after 5s",
  "duration_ms": 5023
}
```

Avantages :
- Recherche par champ : `customer_id:C001 AND level:ERROR`
- Agrégation : nombre d'erreurs par type
- Corrélation avec les traces

## Niveaux de log

| Niveau | Usage | Exemple |
|--------|-------|---------|
| DEBUG | Détails techniques (dev) | "Parsing request body" |
| INFO | Événements normaux | "Quote Q001 created" |
| WARNING | Situations anormales mais gérées | "Fallback used for rating" |
| ERROR | Erreurs qui impactent une requête | "Quote creation failed" |
| CRITICAL | Erreurs système critiques | "Database connection lost" |

## Structure recommandée

### Champs obligatoires
```json
{
  "timestamp": "ISO8601",
  "level": "INFO|WARNING|ERROR|...",
  "service": "nom-du-service",
  "message": "description courte"
}
```

### Champs de corrélation
```json
{
  "trace_id": "identifiant-trace",
  "span_id": "identifiant-span",
  "request_id": "identifiant-requete"
}
```

### Champs contextuels (exemples assurance)
```json
{
  "customer_id": "C001",
  "policy_number": "POL-2024-001",
  "claim_id": "CLM-001",
  "user_id": "agent42",
  "action": "quote.create"
}
```

## Pseudo-code

```python
class StructuredLogger:
    def __init__(self, service_name):
        self.service = service_name

    def log(self, level, message, **context):
        entry = {
            "timestamp": datetime.now().isoformat(),
            "level": level,
            "service": self.service,
            "message": message,
            "trace_id": get_current_trace_id(),
            **context
        }
        print(json.dumps(entry))

# Usage
logger = StructuredLogger("quote-engine")
logger.log("INFO", "Quote created",
    customer_id="C001",
    quote_id="Q001",
    premium=450.00
)
```

## Bonnes pratiques

### 1. Messages concis et actionables
```json
// Mauvais
{"message": "Une erreur s'est produite quelque part"}

// Bon
{"message": "Quote creation failed", "error": "Rating API timeout"}
```

### 2. Données sensibles masquées
```json
// Mauvais
{"customer_email": "jean.dupont@email.com"}

// Bon
{"customer_email": "j***@***.com"}
```

### 3. Contexte métier pertinent
```json
{
  "message": "Claim status changed",
  "claim_id": "CLM-001",
  "old_status": "OPEN",
  "new_status": "APPROVED",
  "approver": "underwriter42"
}
```

### 4. Erreurs avec stack trace
```json
{
  "level": "ERROR",
  "message": "Database query failed",
  "error_type": "ConnectionError",
  "error_message": "Connection refused",
  "stack_trace": "at db.query() line 42\nat policy.save() line 15..."
}
```

## Cas d'usage assurance

### Log de création de devis
```json
{
  "timestamp": "2024-03-15T14:32:05.123Z",
  "level": "INFO",
  "service": "quote-engine",
  "message": "Quote created successfully",
  "trace_id": "trace-abc123",
  "customer_id": "C001",
  "quote_id": "Q-2024-0042",
  "product": "AUTO",
  "premium": 450.00,
  "duration_ms": 234
}
```

### Log d'erreur de tarification
```json
{
  "timestamp": "2024-03-15T14:32:10.456Z",
  "level": "WARNING",
  "service": "quote-engine",
  "message": "External rating failed, using fallback",
  "trace_id": "trace-abc123",
  "customer_id": "C001",
  "error_type": "TimeoutError",
  "fallback_rate": 500.00,
  "external_api": "rating.partner.com"
}
```

### Log d'audit de modification de police
```json
{
  "timestamp": "2024-03-15T14:35:00.789Z",
  "level": "INFO",
  "service": "policy-admin",
  "message": "Policy modified",
  "trace_id": "trace-def456",
  "policy_number": "POL-2024-001",
  "modified_by": "agent42",
  "changes": {
    "coverage": {"old": "RC", "new": "RC+VOL"},
    "premium": {"old": 400, "new": 550}
  },
  "audit_event": true
}
```

## Recherche et analyse

### Requêtes typiques
```
# Toutes les erreurs du Quote Engine
service:quote-engine AND level:ERROR

# Erreurs pour un client spécifique
customer_id:C001 AND level:ERROR

# Toutes les actions d'un agent
modified_by:agent42 AND audit_event:true

# Suivre une requête
trace_id:trace-abc123

# Erreurs de timeout aujourd'hui
error_type:TimeoutError AND timestamp:[now-1d TO now]
```

## Anti-patterns

1. **Log texte libre** : Impossible à parser
2. **Pas de trace_id** : Impossible de corréler
3. **Trop de logs DEBUG en prod** : Surcharge le stockage
4. **Données sensibles en clair** : Problème RGPD
5. **Messages génériques** : "Error occurred" n'aide pas
