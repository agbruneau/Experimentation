# Métriques et Alerting

## Résumé

Les **métriques** sont des mesures numériques agrégées dans le temps. Elles permettent de surveiller la santé du système, détecter les anomalies et déclencher des alertes.

## Points clés

- **Agrégation** : Compteurs, moyennes, percentiles
- **Dimensions** : Tags pour filtrer (service, endpoint, status)
- **Séries temporelles** : Évolution dans le temps
- **Alerting** : Notifications automatiques sur seuils

## Types de métriques

### Counter (Compteur)
Valeur qui ne fait qu'augmenter.

```
requests_total = 42567
errors_total = 123
```

Usage : Nombre de requêtes, erreurs cumulées

### Gauge (Jauge)
Valeur qui peut monter ou descendre.

```
current_connections = 45
cpu_usage_percent = 67.5
queue_length = 12
```

Usage : Connexions actives, utilisation CPU, taille de queue

### Histogram (Histogramme)
Distribution de valeurs avec buckets.

```
request_duration_ms:
  bucket_50ms: 1200
  bucket_100ms: 3400
  bucket_200ms: 4100
  bucket_500ms: 4300
  bucket_1000ms: 4320
  total: 4320
```

Usage : Latences, tailles de payload

### Summary
Calcul de percentiles côté client.

```
request_duration_ms:
  p50: 45ms
  p90: 120ms
  p99: 450ms
  count: 4320
```

## Métriques RED

Framework standard pour les services :

| Métrique | Description | Exemple |
|----------|-------------|---------|
| **R**ate | Requêtes par seconde | 150 req/s |
| **E**rrors | Taux d'erreurs | 0.5% |
| **D**uration | Temps de réponse | p99 = 200ms |

```
# Rate
http_requests_total{service="quote-engine"} rate[5m]

# Errors
http_errors_total{service="quote-engine"} / http_requests_total

# Duration
http_request_duration_seconds{service="quote-engine"} p99
```

## Métriques USE

Framework pour les ressources :

| Métrique | Description | Exemple |
|----------|-------------|---------|
| **U**tilization | Taux d'utilisation | CPU 75% |
| **S**aturation | File d'attente | Queue: 50 |
| **E**rrors | Erreurs de ressource | I/O errors: 2 |

## Dimensions et tags

```
# Sans tags (limité)
requests_total = 1000

# Avec tags (filtrable)
requests_total{
  service="quote-engine",
  endpoint="/quotes",
  method="POST",
  status="200"
} = 850

requests_total{
  service="quote-engine",
  endpoint="/quotes",
  method="POST",
  status="500"
} = 15
```

### Tags recommandés

| Tag | Valeurs | Description |
|-----|---------|-------------|
| service | quote-engine, policy-admin | Nom du service |
| endpoint | /quotes, /policies | Route |
| method | GET, POST, PUT | Méthode HTTP |
| status | 200, 400, 500 | Code retour |
| environment | prod, staging | Environnement |

## Alerting

### Anatomie d'une alerte

```yaml
alert: HighErrorRate
expr: error_rate > 0.05
for: 5m
labels:
  severity: critical
  service: quote-engine
annotations:
  summary: "Taux d'erreur élevé sur Quote Engine"
  description: "Taux d'erreur: {{ $value }}% (seuil: 5%)"
```

### Bonnes pratiques d'alerting

**Alerter sur les symptômes, pas les causes**
```
# Mauvais : Alerte sur cause
alert: HighCPU
expr: cpu_usage > 90%

# Bon : Alerte sur symptôme
alert: HighLatency
expr: request_duration_p99 > 2s
```

**Multi-fenêtre pour éviter les faux positifs**
```yaml
# Alerte seulement si problème persistant
alert: HighErrorRate
expr: |
  error_rate[5m] > 0.05 AND
  error_rate[15m] > 0.03
```

### Niveaux de sévérité

| Niveau | Critère | Action |
|--------|---------|--------|
| Critical | Impact utilisateur immédiat | Page d'astreinte |
| Warning | Dégradation potentielle | Ticket urgent |
| Info | À surveiller | Dashboard |

## Cas d'usage assurance

### Dashboard Quote Engine

```
┌─────────────────────────────────────────────────────────────────┐
│  QUOTE ENGINE - DASHBOARD                                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ Requests/s   │  │ Error Rate   │  │ p99 Latency  │           │
│  │    156       │  │    0.3%      │  │    187ms     │           │
│  │  ▲ +12%      │  │  ✓ < 1%      │  │  ✓ < 500ms   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                  │
│  Latency Distribution (last 1h)                                 │
│  ████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ p50: 45ms │
│  ████████████████████████████░░░░░░░░░░░░░░░░░░░░░░░ p90: 120ms│
│  ██████████████████████████████████████░░░░░░░░░░░░░ p99: 187ms│
│                                                                  │
│  Requests by Status                                             │
│  ■ 200: 95.2%  ■ 400: 4.0%  ■ 500: 0.8%                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Métriques métier assurance

```
# Devis créés par produit
quotes_created_total{product="AUTO"} = 1234
quotes_created_total{product="HOME"} = 567

# Taux de conversion devis → police
conversion_rate{product="AUTO"} = 0.23  # 23%

# Montant moyen des primes
premium_amount_avg{product="AUTO"} = 456.78

# Sinistres par statut
claims_total{status="OPEN"} = 45
claims_total{status="CLOSED"} = 1234
claims_total{status="REJECTED"} = 89
```

### Alertes métier

```yaml
# Alerte si trop peu de devis
alert: LowQuoteVolume
expr: rate(quotes_created_total[1h]) < 10
for: 30m
labels:
  severity: warning
annotations:
  summary: "Volume de devis anormalement bas"

# Alerte si taux de conversion chute
alert: LowConversionRate
expr: conversion_rate < 0.15
for: 1h
labels:
  severity: warning
annotations:
  summary: "Taux de conversion en baisse"
```

## Anti-patterns

1. **Trop de métriques** : Surcharge le stockage et la visualisation
2. **Cardinalité explosive** : Tags avec trop de valeurs uniques (user_id)
3. **Alertes sur tout** : Alert fatigue, on ignore les vraies alertes
4. **Pas de contexte** : Métriques sans tags impossibles à filtrer
5. **Seuils arbitraires** : Basés sur rien, génèrent des faux positifs
