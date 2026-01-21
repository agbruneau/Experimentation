# Audit et Conformité

## Résumé

L'**audit** consiste à enregistrer et conserver les actions effectuées sur le système. C'est essentiel pour la sécurité, la conformité réglementaire et l'investigation en cas d'incident.

## Points clés

- **Audit trail** : Historique immuable des actions
- **Qui, Quoi, Quand, Comment** : Informations à capturer
- **Non-répudiation** : Prouver qu'une action a eu lieu
- **Conformité** : RGPD, réglementations sectorielles

## Éléments d'un audit trail

### Structure d'un événement d'audit

```json
{
  "audit_id": "AUD-2024-001234",
  "timestamp": "2024-03-15T14:32:05.123Z",
  "action": "policy.cancel",
  "resource": {
    "type": "policy",
    "id": "POL-2024-001234"
  },
  "actor": {
    "user_id": "agent42",
    "type": "user",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  },
  "context": {
    "request_id": "req-abc123",
    "trace_id": "trace-xyz789",
    "session_id": "sess-def456"
  },
  "changes": {
    "before": {"status": "ACTIVE"},
    "after": {"status": "CANCELLED"}
  },
  "result": "SUCCESS",
  "reason": "Client request - Reason: Sold vehicle"
}
```

### Actions à auditer

| Catégorie | Actions | Priorité |
|-----------|---------|----------|
| Authentification | Login, logout, échecs | Haute |
| Autorisation | Accès refusé, permissions changées | Haute |
| Données sensibles | Lecture, modification, export | Haute |
| Configuration | Paramètres modifiés | Haute |
| Transactions | Création, modification, annulation | Haute |
| Administration | Utilisateurs, rôles | Haute |
| Consultation | Vues, recherches | Moyenne |

## Implémentation

### Service d'audit

```python
class AuditService:
    def __init__(self, storage):
        self.storage = storage

    async def log(
        self,
        action: str,
        resource_type: str,
        resource_id: str,
        actor: dict,
        changes: dict = None,
        result: str = "SUCCESS",
        reason: str = None
    ):
        event = {
            "audit_id": f"AUD-{uuid4().hex[:12].upper()}",
            "timestamp": datetime.utcnow().isoformat(),
            "action": action,
            "resource": {
                "type": resource_type,
                "id": resource_id
            },
            "actor": actor,
            "context": {
                "trace_id": get_current_trace_id(),
                "request_id": get_current_request_id()
            },
            "changes": changes,
            "result": result,
            "reason": reason
        }

        # Stockage immuable
        await self.storage.append(event)

        # Log structuré également
        logger.info("Audit event",
            action=action,
            resource=f"{resource_type}:{resource_id}",
            actor=actor["user_id"],
            result=result
        )

        return event
```

### Décorateur d'audit automatique

```python
def audited(action: str, resource_type: str):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Extraire le contexte
            request = args[0] if args else kwargs.get("request")
            resource_id = kwargs.get("id") or kwargs.get("resource_id")

            actor = {
                "user_id": request.state.user.id,
                "type": "user",
                "ip_address": request.client.host
            }

            try:
                # Capturer l'état avant
                before = await get_resource_state(resource_type, resource_id)

                # Exécuter l'action
                result = await func(*args, **kwargs)

                # Capturer l'état après
                after = await get_resource_state(resource_type, resource_id)

                # Audit succès
                await audit.log(
                    action=action,
                    resource_type=resource_type,
                    resource_id=resource_id,
                    actor=actor,
                    changes={"before": before, "after": after},
                    result="SUCCESS"
                )

                return result

            except Exception as e:
                # Audit échec
                await audit.log(
                    action=action,
                    resource_type=resource_type,
                    resource_id=resource_id,
                    actor=actor,
                    result="FAILURE",
                    reason=str(e)
                )
                raise

        return wrapper
    return decorator

# Usage
@router.delete("/policies/{id}")
@audited(action="policy.cancel", resource_type="policy")
async def cancel_policy(request, id: str, reason: str):
    ...
```

## Stockage des audits

### Exigences

| Exigence | Description |
|----------|-------------|
| Immutabilité | Pas de modification/suppression |
| Horodatage fiable | Timestamp vérifié |
| Intégrité | Détection des altérations |
| Rétention | Conservation selon réglementation |
| Accessibilité | Recherche et export possibles |

### Options de stockage

```
1. Append-only database
   - PostgreSQL avec triggers bloquant UPDATE/DELETE
   - Colonnes: created_at automatique

2. Event Store
   - Stockage natif append-only
   - Idéal pour event sourcing

3. Système de fichiers signé
   - Fichiers journaliers
   - Signature/hash de chaque fichier

4. Blockchain (cas extrêmes)
   - Immutabilité garantie
   - Pour audits très sensibles
```

### Table d'audit PostgreSQL

```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100) NOT NULL,
    actor_id VARCHAR(100) NOT NULL,
    actor_type VARCHAR(50) NOT NULL,
    actor_ip INET,
    changes JSONB,
    result VARCHAR(20) NOT NULL,
    reason TEXT,
    context JSONB
);

-- Index pour recherche
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_resource ON audit_log(resource_type, resource_id);

-- Bloquer les modifications
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log cannot be modified';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER no_audit_update
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();
```

## Cas d'usage assurance

### Audit de souscription

```json
{
  "audit_id": "AUD-2024-123456",
  "timestamp": "2024-03-15T14:32:05Z",
  "action": "policy.create",
  "resource": {
    "type": "policy",
    "id": "POL-2024-001234"
  },
  "actor": {
    "user_id": "agent42",
    "role": "agent",
    "ip_address": "192.168.1.100"
  },
  "changes": {
    "before": null,
    "after": {
      "status": "ACTIVE",
      "customer_id": "C001",
      "product": "AUTO",
      "premium": 456.00
    }
  },
  "result": "SUCCESS",
  "business_context": {
    "quote_id": "Q-2024-789",
    "sales_channel": "DIRECT"
  }
}
```

### Audit de consultation de données sensibles

```json
{
  "audit_id": "AUD-2024-789012",
  "timestamp": "2024-03-15T15:10:00Z",
  "action": "customer.view_sensitive",
  "resource": {
    "type": "customer",
    "id": "C001"
  },
  "actor": {
    "user_id": "claims_handler_12",
    "role": "claims_handler"
  },
  "result": "SUCCESS",
  "data_accessed": ["ssn", "bank_account"],
  "business_justification": "Claim CLM-2024-456 processing"
}
```

## Conformité réglementaire

### RGPD

| Exigence | Audit requis |
|----------|--------------|
| Droit d'accès | Qui a consulté mes données ? |
| Droit de rectification | Qui a modifié mes données ? |
| Droit à l'effacement | Preuve de suppression |
| Portabilité | Export et traçabilité |

### Solvabilité II (Assurance)

- Traçabilité des décisions de souscription
- Audit des calculs de provisions
- Conservation 10 ans minimum

### PCI-DSS

- Audit de tous les accès aux données carte
- Rétention 1 an minimum
- Revue quotidienne des logs

## Bonnes pratiques

1. **Tout auditer** qui concerne les données sensibles
2. **Contexte métier** dans l'audit (pas juste technique)
3. **Stockage séparé** du reste de l'application
4. **Accès en lecture seule** pour les équipes d'investigation
5. **Alertes** sur les patterns suspects
6. **Tests réguliers** de récupération des audits

## Anti-patterns

1. **Audit modifiable** : Perd toute valeur probante
2. **Audit incomplet** : Manque le "pourquoi"
3. **Pas de recherche** : Inutilisable en pratique
4. **Rétention courte** : Non-conforme
5. **Audit dans les logs applicatifs** : Mélange avec le bruit
