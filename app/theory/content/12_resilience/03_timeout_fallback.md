# Timeout et Fallback

## Timeout

### Résumé

Le **Timeout** définit une durée maximale d'attente pour une opération. C'est la première ligne de défense contre les services lents ou non-répondants.

### Points clés

- **Fail fast** : Ne pas bloquer indéfiniment
- **Libérer les ressources** : Threads, connexions, mémoire
- **Améliorer l'UX** : Réponse rapide même si dégradée

### Types de timeout

| Type | Description | Exemple |
|------|-------------|---------|
| Connection Timeout | Temps pour établir la connexion | 5s |
| Read Timeout | Temps pour recevoir la réponse | 30s |
| Total Timeout | Durée totale de l'opération | 60s |

### Configuration par type de service

```
Services critiques (paiement):
  timeout = 10s
  Pas de fallback possible

Services enrichissement (recommandations):
  timeout = 2s
  Fallback: valeurs par défaut

Services batch (reporting):
  timeout = 300s
  Pas d'impact utilisateur
```

### Pseudo-code

```
async function with_timeout(operation, timeout_seconds):
    try:
        return await wait_for(operation(), timeout=timeout_seconds)
    catch TimeoutError:
        throw OperationTimeoutException(
            f"Opération timeout après {timeout_seconds}s"
        )
```

---

## Fallback

### Résumé

Le **Fallback** fournit une valeur ou un comportement alternatif quand l'opération principale échoue. C'est le "Plan B" automatique.

### Points clés

- **Dégradation gracieuse** : Le service reste partiellement fonctionnel
- **Transparence** : L'utilisateur peut ne pas voir la différence
- **Plusieurs niveaux** : Cache → Fallback function → Default value

### Stratégies de Fallback

#### 1. Valeur par défaut
```
Si tarificateur externe KO:
  → Utiliser tarif standard prédéfini
```

#### 2. Cache (stale data)
```
Si service KO:
  → Retourner la dernière valeur mise en cache
  → Marquer comme "données potentiellement obsolètes"
```

#### 3. Service alternatif
```
Si Tarificateur A KO:
  → Appeler Tarificateur B (backup)
```

#### 4. Fonctionnalité dégradée
```
Si Service de recommandations KO:
  → Afficher les produits les plus populaires
```

### Pseudo-code

```
async function with_fallback(operation, fallback):
    try:
        return await operation()
    catch Exception:
        if callable(fallback):
            return fallback()
        return fallback
```

### Exemple combiné : Timeout + Fallback

```python
DEFAULT_RATES = {"AUTO": 500, "HOME": 300}
CACHE = {}

@with_timeout(seconds=5.0)
@with_fallback(fallback_function=get_cached_rate)
async def get_external_rate(product, risk_data):
    rate = await external_rating_api.calculate(product, risk_data)
    CACHE[product] = rate  # Mise en cache
    return rate

def get_cached_rate(product, risk_data):
    if product in CACHE:
        return CACHE[product]
    return DEFAULT_RATES.get(product, 400)
```

---

## Combinaison des patterns de résilience

### Ordre d'application recommandé

```
Requête
    │
    ▼
┌───────────────┐
│   Timeout     │ → Limite le temps d'attente
└───────────────┘
    │
    ▼
┌───────────────┐
│    Retry      │ → Réessaie N fois
└───────────────┘
    │
    ▼
┌───────────────┐
│Circuit Breaker│ → Coupe si trop d'échecs
└───────────────┘
    │
    ▼ (échec)
┌───────────────┐
│   Fallback    │ → Valeur alternative
└───────────────┘
    │
    ▼
  Réponse
```

### Pseudo-code combiné

```python
@circuit_breaker(failure_threshold=5)
@with_timeout(seconds=5.0)
@retry_with_backoff(max_retries=3)
@with_fallback(fallback_value={"rate": 500})
async def get_rate(risk_data):
    return await external_api.calculate(risk_data)
```

---

## Cas d'usage assurance

### Devis avec tarificateur externe

```
Client demande un devis
        │
        ▼
┌─────────────────────────────────────────┐
│           Quote Engine                   │
│                                          │
│  1. Calculer avec Rating API externe    │
│     timeout=5s, retry=3, fallback=cache │
│                                          │
│  2. Si fallback utilisé:                │
│     marquer devis comme "estimatif"     │
│                                          │
│  3. Retourner le devis                  │
└─────────────────────────────────────────┘
        │
        ▼
  Devis retourné
  (précis ou estimatif)
```

### Vue 360° client

```
API Composition
      │
      ├──▶ Customer Hub    [timeout=2s, fallback=partiel]
      ├──▶ Policy Admin    [timeout=3s, fallback=partiel]
      ├──▶ Claims          [timeout=3s, fallback=vide]
      └──▶ Billing         [timeout=2s, fallback=vide]
      │
      ▼
  Vue 360° (complète ou partielle)
```

---

## Métriques importantes

| Métrique | Description | Seuil d'alerte |
|----------|-------------|----------------|
| timeout_rate | % de requêtes en timeout | > 5% |
| fallback_rate | % de requêtes utilisant fallback | > 10% |
| circuit_open_duration | Durée circuit ouvert | > 5 min |
| p99_latency | Latence 99e percentile | > timeout/2 |

---

## Bonnes pratiques

1. **Définir des timeouts explicites** partout (jamais de timeout infini)
2. **Adapter les timeouts** au type d'opération
3. **Toujours avoir un fallback** pour les services non-critiques
4. **Logger les fallbacks** pour monitoring
5. **Informer l'utilisateur** si les données sont dégradées
6. **Tester les fallbacks** régulièrement (chaos engineering)
