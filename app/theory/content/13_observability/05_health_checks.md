# Health Checks et Readiness Probes

## Résumé

Les **Health Checks** permettent de vérifier automatiquement l'état de santé d'un service. Ils sont essentiels pour le load balancing, l'orchestration et la résilience du système.

## Points clés

- **Liveness** : Le service est-il en vie ? (redémarrer si non)
- **Readiness** : Le service peut-il traiter des requêtes ? (retirer du LB si non)
- **Startup** : Le service a-t-il fini de démarrer ?
- **Deep vs Shallow** : Vérification superficielle vs dépendances

## Types de Health Checks

### Liveness Probe
"Le processus est-il en vie ?"

```
GET /health/live
→ 200 OK

Si échec répété → Redémarrer le service
```

Usage : Détecter les processus bloqués (deadlock, memory leak)

### Readiness Probe
"Le service peut-il traiter des requêtes ?"

```
GET /health/ready
→ 200 OK (toutes les dépendances OK)
→ 503 Service Unavailable (DB indisponible)

Si échec → Retirer du load balancer
```

Usage : Ne pas envoyer de trafic à un service qui ne peut pas répondre

### Startup Probe
"Le service a-t-il fini de démarrer ?"

```
GET /health/startup
→ 503 (en cours de démarrage)
→ 200 OK (prêt)

Tant que startup probe échoue:
- Liveness et Readiness ne sont pas vérifiés
- Le service n'est pas exposé
```

Usage : Services avec démarrage long (chargement cache, warm-up)

## Shallow vs Deep Health Check

### Shallow (superficiel)
```
GET /health
→ 200 OK

Vérifie seulement que le processus répond.
Rapide, pas de dépendance.
```

### Deep (approfondi)
```
GET /health/ready
{
  "status": "healthy",
  "checks": {
    "database": {"status": "healthy", "latency_ms": 5},
    "redis": {"status": "healthy", "latency_ms": 2},
    "external_api": {"status": "degraded", "latency_ms": 1500}
  }
}
```

Vérifie toutes les dépendances.
Plus lent, plus informatif.

## Implémentation

### Structure de réponse recommandée

```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2024-03-15T14:32:05Z",
  "version": "2.3.1",
  "uptime_seconds": 86400,
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 5,
      "message": "Connected to primary"
    },
    "cache": {
      "status": "healthy",
      "latency_ms": 2,
      "hit_rate": 0.95
    },
    "external_rating": {
      "status": "degraded",
      "latency_ms": 1500,
      "message": "Slow response, using fallback"
    }
  }
}
```

### Pseudo-code

```python
async def health_check():
    checks = {}
    overall_status = "healthy"

    # Check Database
    try:
        start = time.time()
        await db.execute("SELECT 1")
        latency = (time.time() - start) * 1000
        checks["database"] = {
            "status": "healthy",
            "latency_ms": latency
        }
    except Exception as e:
        checks["database"] = {
            "status": "unhealthy",
            "error": str(e)
        }
        overall_status = "unhealthy"

    # Check External API (non-bloquant)
    try:
        start = time.time()
        await external_api.ping()
        latency = (time.time() - start) * 1000
        status = "healthy" if latency < 1000 else "degraded"
        checks["external_api"] = {
            "status": status,
            "latency_ms": latency
        }
        if status == "degraded":
            overall_status = "degraded"
    except Exception as e:
        checks["external_api"] = {
            "status": "degraded",
            "error": str(e),
            "message": "Using fallback"
        }
        if overall_status == "healthy":
            overall_status = "degraded"

    return {
        "status": overall_status,
        "checks": checks
    }
```

## Configuration

### Intervalles recommandés

| Probe | Intervalle | Timeout | Échecs avant action |
|-------|------------|---------|---------------------|
| Liveness | 10s | 5s | 3 |
| Readiness | 5s | 3s | 1 |
| Startup | 5s | 5s | 30 (long timeout) |

### Exemple configuration

```yaml
# Liveness: redémarre si bloqué
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

# Readiness: retire du LB si pas prêt
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 1

# Startup: attend que le service démarre
startupProbe:
  httpGet:
    path: /health/startup
    port: 8080
  periodSeconds: 5
  failureThreshold: 30  # 30 × 5s = 150s max
```

## Cas d'usage assurance

### Quote Engine Health Check

```json
{
  "status": "healthy",
  "service": "quote-engine",
  "version": "2.3.1",
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 3,
      "pool_used": 5,
      "pool_max": 20
    },
    "external_rating": {
      "status": "healthy",
      "latency_ms": 150,
      "circuit_breaker": "CLOSED"
    },
    "customer_hub": {
      "status": "healthy",
      "latency_ms": 25
    },
    "cache": {
      "status": "healthy",
      "hit_rate": 0.87,
      "memory_mb": 256
    }
  }
}
```

### Gestion de la dégradation

```python
def determine_status(checks):
    """
    Détermine le statut global basé sur les checks.

    Règles:
    - unhealthy si dépendance critique KO (database)
    - degraded si dépendance non-critique KO (external_api)
    - healthy sinon
    """
    critical = ["database"]
    non_critical = ["external_rating", "cache"]

    for name, check in checks.items():
        if check["status"] == "unhealthy":
            if name in critical:
                return "unhealthy"
            return "degraded"

    return "healthy"
```

## Bonnes pratiques

### 1. Séparer liveness et readiness
```
/health/live → Juste vérifier que le process répond
/health/ready → Vérifier les dépendances
```

### 2. Timeout court pour les checks
```
# Mauvais : timeout de 30s
# Si la DB est lente, le check bloque tout

# Bon : timeout de 3s
# Échec rapide si la DB ne répond pas
```

### 3. Ne pas inclure les dépendances externes dans liveness
```python
# Mauvais : Liveness vérifie l'API externe
# Si l'API externe est down, tous les pods redémarrent en boucle!

# Bon : Liveness ne vérifie que le process local
```

### 4. Cacher les résultats des checks
```python
# Éviter de surcharger les dépendances avec des checks
cache = {}
CACHE_TTL = 5  # secondes

async def cached_health_check():
    if "result" in cache and cache["time"] > time.time() - CACHE_TTL:
        return cache["result"]

    result = await do_health_check()
    cache["result"] = result
    cache["time"] = time.time()
    return result
```

### 5. Exposer des métriques sur les checks
```python
# Métriques pour chaque check
health_check_duration_seconds{check="database"}
health_check_status{check="database", status="healthy|degraded|unhealthy"}
```

## Anti-patterns

1. **Liveness qui vérifie tout** : Redémarrages en cascade
2. **Timeout trop long** : Détection lente des problèmes
3. **Pas de startup probe** : Trafic envoyé avant que le service soit prêt
4. **Check qui modifie l'état** : Health check ne doit être que lecture
5. **Pas de cache** : Surcharge les dépendances
