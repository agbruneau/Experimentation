# Introduction au Chaos Engineering

## Résumé

Le **Chaos Engineering** est la discipline qui consiste à expérimenter sur un système distribué pour renforcer la confiance dans sa capacité à résister aux conditions turbulentes en production.

## Points clés

- **Tester en conditions réelles** : Injecter des pannes contrôlées
- **Découvrir les faiblesses** avant qu'elles ne surviennent en production
- **Valider les mécanismes de résilience** (circuit breaker, retry, fallback)
- **Améliorer continuellement** la robustesse du système

## Principes du Chaos Engineering

### 1. Définir l'état stable
Identifier les métriques qui définissent un fonctionnement normal.

```
État stable du Quote Engine:
- Latence p99 < 200ms
- Taux d'erreur < 0.1%
- Débit > 100 req/s
```

### 2. Formuler une hypothèse
```
"Si le tarificateur externe est indisponible,
 le système doit continuer à fonctionner avec le fallback,
 et la latence ne doit pas dépasser 500ms."
```

### 3. Introduire des variables de chaos
```
Types de chaos:
├── Latence : Ajouter 5s de délai
├── Erreurs : Retourner 500 pour 50% des requêtes
├── Panne totale : Service complètement indisponible
└── Partition réseau : Isoler un service
```

### 4. Observer et analyser
```
Métriques à observer:
├── Latence (p50, p95, p99)
├── Taux d'erreur
├── Taux de fallback utilisé
├── État des circuit breakers
└── Expérience utilisateur
```

## Types d'expériences

### Injection de latence
```python
# Simule un service lent
async def chaos_latency(request):
    if random() < CHAOS_PROBABILITY:
        await sleep(CHAOS_LATENCY_SECONDS)
    return await original_handler(request)
```

### Injection d'erreurs
```python
# Simule des erreurs aléatoires
async def chaos_error(request):
    if random() < CHAOS_ERROR_RATE:
        raise ServiceUnavailable("Chaos injection")
    return await original_handler(request)
```

### Kill service
```bash
# Arrête un service pendant 30 secondes
docker stop quote-engine
sleep 30
docker start quote-engine
```

## Scénarios de chaos pour l'assurance

### Scénario 1 : Panne tarificateur
```
Hypothèse: Le système utilise le fallback sans impact utilisateur

Expérience:
1. Désactiver le tarificateur externe
2. Observer le comportement du Quote Engine
3. Vérifier que les devis sont créés (avec tarif estimatif)
4. Mesurer la latence

Résultat attendu:
- Circuit breaker s'ouvre après 5 échecs
- Fallback activé (tarif par défaut)
- Latence < 500ms
- Taux de succès > 99%
```

### Scénario 2 : Base de données lente
```
Hypothèse: Les timeouts protègent le système

Expérience:
1. Ajouter 5s de latence sur la DB
2. Observer les timeouts
3. Vérifier les retries
4. Mesurer l'impact utilisateur

Résultat attendu:
- Timeout après 3s
- Retry avec backoff
- Erreur gracieuse si tous retries échouent
```

### Scénario 3 : Pic de charge
```
Hypothèse: Les bulkheads isolent correctement

Expérience:
1. Envoyer 10x le trafic normal sur Claims
2. Observer si Quote Engine est impacté
3. Vérifier le rejet des requêtes excédentaires

Résultat attendu:
- Claims: rejets après saturation bulkhead
- Quote: fonctionnement normal
- Pas de dégradation généralisée
```

## Matrice de chaos

| Composant | Latence | Erreurs | Panne | Partition |
|-----------|---------|---------|-------|-----------|
| Quote Engine | ✓ | ✓ | ✓ | ✓ |
| Policy Admin | ✓ | ✓ | ✓ | ✓ |
| Claims | ✓ | ✓ | ✓ | ✓ |
| External Rating | ✓ | ✓ | ✓ | - |
| Database | ✓ | ✓ | ✓ | - |
| Message Broker | - | ✓ | ✓ | ✓ |

## Implémentation simple

```python
class ChaosMonkey:
    def __init__(self, config):
        self.enabled = config.get("enabled", False)
        self.latency_probability = config.get("latency_prob", 0)
        self.latency_seconds = config.get("latency_sec", 0)
        self.error_probability = config.get("error_prob", 0)

    async def maybe_inject_chaos(self):
        if not self.enabled:
            return

        # Injection de latence
        if random() < self.latency_probability:
            await asyncio.sleep(self.latency_seconds)

        # Injection d'erreur
        if random() < self.error_probability:
            raise ChaosException("Chaos injection!")

    def wrap(self, func):
        async def wrapper(*args, **kwargs):
            await self.maybe_inject_chaos()
            return await func(*args, **kwargs)
        return wrapper
```

## Bonnes pratiques

### Commencer petit
1. Commencer en environnement de test
2. Puis staging avec faible probabilité
3. Enfin production avec précautions

### Automatiser
```yaml
# chaos-experiment.yaml
experiment:
  name: "tarificateur-panne"
  hypothesis: "Le fallback maintient le service"
  steady_state:
    - metric: latency_p99
      max: 500ms
    - metric: error_rate
      max: 1%
  actions:
    - type: kill_service
      target: external-rating
      duration: 60s
  rollback:
    automatic: true
    on_steady_state_violation: true
```

### Documenter les résultats
```markdown
## Résultat Expérience #42

**Date**: 2024-03-15
**Hypothèse**: Circuit breaker protège contre la panne du tarificateur

**Observations**:
- Circuit breaker ouvert après 5 échecs (15s)
- Fallback activé avec succès
- Latence p99: 450ms (attendu: <500ms) ✓
- Aucune erreur visible utilisateur ✓

**Actions correctives**:
- RAS - Comportement conforme

**Status**: VALIDÉ ✓
```

## Ce qu'il faut éviter

1. **Chaos en production sans préparation** : Toujours commencer en test
2. **Pas de rollback automatique** : Prévoir l'arrêt d'urgence
3. **Tester sans monitoring** : Impossible de mesurer l'impact
4. **Ignorer les résultats** : Chaque découverte doit être traitée
5. **Chaos permanent** : Les expériences doivent être ponctuelles et contrôlées
