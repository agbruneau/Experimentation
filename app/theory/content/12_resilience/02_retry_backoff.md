# Retry avec Backoff Exponentiel

## Résumé

Le pattern **Retry** permet de réessayer automatiquement une opération qui a échoué. Le **Backoff exponentiel** augmente progressivement le délai entre les tentatives pour éviter de surcharger un service en difficulté.

## Points clés

- **Gérer les échecs temporaires** (réseau, surcharge momentanée)
- **Backoff** : attendre de plus en plus longtemps entre les retries
- **Jitter** : ajouter de l'aléatoire pour éviter les "thundering herd"
- **Savoir quand abandonner** (max retries)

## Le problème

Les erreurs transitoires sont fréquentes :
- Timeout réseau
- Service temporairement surchargé (HTTP 503)
- Base de données momentanément indisponible

Abandonner immédiatement n'est pas optimal.

## Stratégies de Backoff

### Fixed Delay
```
Tentative 1 → attendre 1s
Tentative 2 → attendre 1s
Tentative 3 → attendre 1s
```
Simple mais peut surcharger un service qui se remet.

### Linear Backoff
```
Tentative 1 → attendre 1s
Tentative 2 → attendre 2s
Tentative 3 → attendre 3s
```
Augmentation progressive.

### Exponential Backoff
```
Tentative 1 → attendre 1s
Tentative 2 → attendre 2s
Tentative 3 → attendre 4s
Tentative 4 → attendre 8s
```
Formule : `delay = initial * 2^attempt`

### Exponential avec Jitter
```
Tentative 1 → attendre 1s ± 0.1s
Tentative 2 → attendre 2s ± 0.2s
Tentative 3 → attendre 4s ± 0.4s
```
Évite la synchronisation des retries de plusieurs clients.

## Pseudo-code

```
function retry_with_backoff(operation, max_retries=3):
    for attempt in range(max_retries + 1):
        try:
            return operation()
        catch RetryableException as e:
            if attempt == max_retries:
                throw e

            delay = calculate_delay(attempt)
            sleep(delay)

function calculate_delay(attempt):
    base_delay = INITIAL_DELAY * (2 ^ attempt)
    jitter = base_delay * JITTER_FACTOR * random(-1, 1)
    delay = base_delay + jitter
    return min(delay, MAX_DELAY)
```

## Quand retry vs quand abandonner

### Retryable (réessayer)
- HTTP 408 (Request Timeout)
- HTTP 429 (Too Many Requests)
- HTTP 500 (Internal Server Error)
- HTTP 502 (Bad Gateway)
- HTTP 503 (Service Unavailable)
- HTTP 504 (Gateway Timeout)
- Erreurs réseau

### Non-Retryable (abandonner)
- HTTP 400 (Bad Request)
- HTTP 401 (Unauthorized)
- HTTP 403 (Forbidden)
- HTTP 404 (Not Found)
- Erreurs de validation métier

## Configuration recommandée

| Paramètre | Valeur | Description |
|-----------|--------|-------------|
| max_retries | 3 | Nombre max de tentatives |
| initial_delay | 1s | Délai initial |
| max_delay | 60s | Délai maximum |
| jitter_factor | 0.1 | Variation aléatoire (10%) |

## Cas d'usage assurance

### Appel au tarificateur externe
```python
@retry_with_backoff(max_retries=3)
async def get_external_rate(risk_data):
    response = await http_client.post(
        RATING_API_URL,
        json=risk_data,
        timeout=5.0
    )
    return response.json()
```

### Sauvegarde en base de données
```python
@retry_with_backoff(
    max_retries=5,
    retry_on=(DatabaseConnectionError, TimeoutError)
)
async def save_policy(policy):
    await db.policies.insert(policy)
```

## Thundering Herd Problem

Sans jitter, après une panne :
```
Service revient
    │
    ├── Client A retry → SUCCÈS
    ├── Client B retry → SUCCÈS  } Tous en même temps!
    ├── Client C retry → SUCCÈS  } Service surchargé
    └── Client D retry → ÉCHEC   } Nouvelle panne!
```

Avec jitter :
```
Service revient
    │
    ├── Client A retry (t+0.9s) → SUCCÈS
    ├── Client C retry (t+1.1s) → SUCCÈS
    ├── Client B retry (t+1.3s) → SUCCÈS
    └── Client D retry (t+1.5s) → SUCCÈS
```

## Combinaison avec Circuit Breaker

```
Requête
    │
    ▼
┌─────────────┐
│Circuit Check│
└─────────────┘
    │ CLOSED
    ▼
┌─────────────┐
│   Retry     │ ──échec──▶ Compteur CB++
│   Logic     │
└─────────────┘
    │ succès
    ▼
  Réponse
```

Le retry se fait **à l'intérieur** du circuit breaker. Après N échecs cumulés, le circuit s'ouvre.

## Anti-patterns

1. **Retry infini** : Toujours définir un max_retries
2. **Retry sans backoff** : Surcharge le service déjà en difficulté
3. **Retry sur erreurs non-retryables** : Gaspillage de ressources
4. **Délai trop court** : Le service n'a pas le temps de récupérer
5. **Pas d'idempotence** : Risque de doublons si l'opération n'est pas idempotente
