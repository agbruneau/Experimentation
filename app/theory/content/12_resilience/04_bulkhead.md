# Bulkhead Pattern

## Résumé

Le pattern **Bulkhead** (cloison étanche) isole les ressources entre différentes parties du système pour éviter qu'une panne dans un composant n'affecte les autres. Inspiré des cloisons des navires qui limitent les dégâts en cas de brèche.

## Points clés

- **Isolation des ressources** : Chaque service a ses propres ressources
- **Limiter l'impact des pannes** : Un service lent n'affecte pas les autres
- **Garantir la disponibilité** : Les services critiques restent accessibles

## Le problème

Sans isolation, un service lent consomme toutes les ressources :

```
Pool de connexions partagé (100 max)
         │
         ├──▶ Quote Engine (rapide) : utilise 10 connexions
         ├──▶ Claims (normal) : utilise 20 connexions
         └──▶ Rating API (LENT!) : utilise 70 connexions... bloqué!
                                    ↓
                         Toutes les connexions utilisées!
                         Quote et Claims aussi bloqués!
```

## La solution

```
Bulkhead - Pools séparés

┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│  Pool Quote (30) │   │  Pool Claims (30)│   │  Pool Rating (40)│
│     ████░░░░░    │   │     ████████░░   │   │     ████████████ │
│     (10 utilisées)│   │     (20 utilisées)│   │     (40 saturé!) │
└──────────────────┘   └──────────────────┘   └──────────────────┘
         ↓                      ↓                      ↓
   Quote OK ✓              Claims OK ✓           Rating KO ✗
                                            (seulement Rating affecté)
```

## Types de Bulkhead

### Thread Pool Isolation
Chaque service a son propre pool de threads.

```
┌─────────────────────────────────┐
│         Thread Pools            │
├─────────────┬─────────┬─────────┤
│ QuotePool   │ClaimsPool│RatingPool│
│  [10 threads]│[10 threads]│[5 threads] │
└─────────────┴─────────┴─────────┘
```

### Semaphore Isolation
Limite le nombre d'appels concurrents par compteur.

```python
quote_semaphore = Semaphore(10)
rating_semaphore = Semaphore(5)

async def call_quote():
    async with quote_semaphore:
        return await quote_service.call()
```

### Connection Pool Isolation
Pools de connexions dédiés par service externe.

```
Database Connections:
├── App DB Pool: 50 connexions
├── Reporting DB Pool: 20 connexions
└── Audit DB Pool: 10 connexions
```

## Pseudo-code

```python
class Bulkhead:
    def __init__(self, name, max_concurrent):
        self.name = name
        self.semaphore = Semaphore(max_concurrent)
        self.current = 0

    async def execute(self, operation):
        if not self.semaphore.try_acquire():
            raise BulkheadFullException(
                f"Bulkhead {self.name} is full"
            )
        try:
            self.current += 1
            return await operation()
        finally:
            self.current -= 1
            self.semaphore.release()
```

## Configuration recommandée

| Service | Type | Limite | Raison |
|---------|------|--------|--------|
| Quote Engine | Semaphore | 50 | Service critique, haute disponibilité |
| Claims | Semaphore | 30 | Volume modéré |
| External Rating | Thread Pool | 10 | Service externe lent |
| Reporting | Thread Pool | 5 | Non-critique, peut attendre |

## Cas d'usage assurance

### Gateway avec Bulkheads

```
                    API Gateway
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐     ┌─────────┐     ┌─────────┐
   │Bulkhead │     │Bulkhead │     │Bulkhead │
   │ Quotes  │     │ Policies│     │ Claims  │
   │ max=100 │     │ max=100 │     │ max=50  │
   └────┬────┘     └────┬────┘     └────┬────┘
        │               │               │
        ▼               ▼               ▼
   Quote Engine    Policy Admin    Claims Mgmt
```

### Composition avec isolation

```
Vue 360° Client
      │
      ├──▶ [Bulkhead A] Customer Hub    max=20
      ├──▶ [Bulkhead B] Policy Admin    max=20
      ├──▶ [Bulkhead C] Claims          max=10
      └──▶ [Bulkhead D] External APIs   max=5
```

Si External APIs est lent, seul le bulkhead D sature.
Les autres composants continuent de fonctionner.

## Combinaison avec Circuit Breaker

```
Requête
    │
    ▼
┌──────────────┐
│   Bulkhead   │ → Limite les appels concurrents
└──────────────┘
    │
    ▼
┌──────────────┐
│Circuit Breaker│ → Coupe si le service est KO
└──────────────┘
    │
    ▼
   Service
```

**Bulkhead** protège contre les **services lents**.
**Circuit Breaker** protège contre les **services défaillants**.

## Dimensionnement

### Formule de base

```
Max Concurrent = (Requêtes/seconde × Latence moyenne) × Marge sécurité

Exemple:
- 100 req/s attendues
- 200ms de latence moyenne
- Marge de sécurité: 2x

Max = (100 × 0.2) × 2 = 40 connexions concurrentes
```

### Surveillance

| Métrique | Seuil d'alerte | Action |
|----------|----------------|--------|
| bulkhead_usage | > 80% | Augmenter la limite ou investiguer |
| bulkhead_rejections | > 0 | Service en difficulté |
| wait_time | > 100ms | Queue trop longue |

## Anti-patterns

1. **Limite trop haute** : N'isole pas vraiment
2. **Limite trop basse** : Rejette des requêtes légitimes
3. **Pas de monitoring** : Impossible de savoir si ça fonctionne
4. **Bulkhead unique** : Revient à ne pas avoir de bulkhead

## Bonnes pratiques

1. **Séparer par criticité** : Services critiques avec plus de ressources
2. **Séparer les externes** : Les services tiers dans leur propre bulkhead
3. **Monitorer l'utilisation** : Ajuster dynamiquement si nécessaire
4. **Prévoir le fallback** : Que faire quand le bulkhead est plein ?
5. **Tester en charge** : Valider les limites avec des tests de performance
