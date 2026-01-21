# Circuit Breaker Pattern

## Résumé

Le **Circuit Breaker** est un pattern de résilience qui protège votre système contre les pannes en cascade. Comme un disjoncteur électrique, il "coupe le circuit" quand trop d'erreurs surviennent.

## Points clés

- **Trois états** : Fermé (normal), Ouvert (bloqué), Semi-ouvert (test)
- **Protection automatique** contre les services défaillants
- **Fail fast** : évite d'attendre des réponses qui n'arriveront jamais
- **Récupération automatique** quand le service revient

## Le problème

Quand un service externe (ex: tarificateur) tombe en panne :

```
Sans Circuit Breaker:
┌─────────┐    ❌ timeout    ┌─────────┐
│ Gateway │ ─────────────────│ Rating  │ (en panne)
└─────────┘    30s x 1000    └─────────┘
               = 500 min perdues!
```

Chaque requête attend le timeout, consommant des ressources et dégradant l'expérience utilisateur.

## La solution

```
Avec Circuit Breaker:
┌─────────┐    ┌─────────────┐    ┌─────────┐
│ Gateway │───▶│ CB: CLOSED  │───▶│ Rating  │
└─────────┘    └─────────────┘    └─────────┘
                    │
              5 échecs
                    ▼
┌─────────┐    ┌─────────────┐
│ Gateway │───▶│ CB: OPEN    │──✕ (Échec immédiat)
└─────────┘    └─────────────┘
                    │
              30s timeout
                    ▼
┌─────────┐    ┌─────────────┐    ┌─────────┐
│ Gateway │───▶│ CB: HALF_OPEN│───▶│ Rating  │ (test)
└─────────┘    └─────────────┘    └─────────┘
```

## Les trois états

### CLOSED (Fermé)
- État normal, les appels passent
- Compteur d'échecs actif
- Après N échecs → passe en OPEN

### OPEN (Ouvert)
- Tous les appels échouent immédiatement
- Pas de tentative vers le service
- Après un délai → passe en HALF_OPEN

### HALF_OPEN (Semi-ouvert)
- Laisse passer quelques appels tests
- Si succès → retourne en CLOSED
- Si échec → retourne en OPEN

## Pseudo-code

```
class CircuitBreaker:
    state = CLOSED
    failure_count = 0
    last_failure_time = null

    function call(operation):
        if state == OPEN:
            if now() - last_failure_time > reset_timeout:
                state = HALF_OPEN
            else:
                throw CircuitOpenException

        try:
            result = operation()
            on_success()
            return result
        catch exception:
            on_failure()
            throw exception

    function on_success():
        failure_count = 0
        if state == HALF_OPEN:
            state = CLOSED

    function on_failure():
        failure_count++
        last_failure_time = now()
        if failure_count >= threshold:
            state = OPEN
```

## Configuration recommandée

| Paramètre | Valeur typique | Description |
|-----------|----------------|-------------|
| failure_threshold | 5 | Échecs avant ouverture |
| success_threshold | 2 | Succès pour fermer |
| reset_timeout | 30s | Délai avant HALF_OPEN |

## Cas d'usage assurance

### Tarificateur externe
```
Quote Engine ──CB──▶ External Rating API
                        │
                   Si CB ouvert:
                   Utiliser tarif par défaut
```

### Services partenaires
```
Gateway ──CB──▶ Partenaire A
        ──CB──▶ Partenaire B
        ──CB──▶ Partenaire C

Chaque partenaire a son propre Circuit Breaker
```

## Anti-patterns à éviter

1. **Seuil trop bas** : Le circuit s'ouvre pour des erreurs ponctuelles
2. **Timeout trop long** : Les utilisateurs attendent trop
3. **Pas de fallback** : Circuit ouvert = fonctionnalité indisponible
4. **Ignorer les métriques** : Pas de visibilité sur l'état des circuits

## Patterns complémentaires

- **Retry** : Réessayer avant d'ouvrir le circuit
- **Fallback** : Valeur par défaut quand le circuit est ouvert
- **Timeout** : Limite le temps d'attente
- **Bulkhead** : Isole les ressources par service
