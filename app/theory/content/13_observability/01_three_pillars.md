# Les Trois Piliers de l'Observabilité

## Résumé

L'**observabilité** est la capacité à comprendre l'état interne d'un système à partir de ses sorties externes. Elle repose sur trois piliers complémentaires : **Logs**, **Metrics** et **Traces**.

## Points clés

- **Logs** : Événements discrets avec contexte (le "quoi" et "pourquoi")
- **Metrics** : Mesures numériques agrégées (le "combien")
- **Traces** : Suivi des requêtes à travers les services (le "où" et "quand")

## Vue d'ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                    OBSERVABILITÉ                                 │
├────────────────┬────────────────────┬────────────────────────────┤
│                │                    │                            │
│     LOGS       │     METRICS        │     TRACES                 │
│                │                    │                            │
│  Événements    │  Agrégations       │  Flux de requêtes          │
│  discrets      │  numériques        │  distribués                │
│                │                    │                            │
│  "Erreur à     │  "99% des requêtes │  "Cette requête a          │
│   14:32:05     │   < 200ms"         │   traversé A→B→C           │
│   sur Quote"   │                    │   en 450ms"                │
│                │                    │                            │
│  Debug,        │  Dashboard,        │  Diagnostic,               │
│  Audit         │  Alerting          │  Analyse perf              │
│                │                    │                            │
└────────────────┴────────────────────┴────────────────────────────┘
```

## Quand utiliser quoi ?

| Besoin | Pilier | Exemple |
|--------|--------|---------|
| Comprendre une erreur spécifique | Logs | Stack trace, paramètres |
| Voir les tendances | Metrics | Latence moyenne sur 1h |
| Suivre une requête utilisateur | Traces | Chemin à travers les services |
| Alerter sur anomalie | Metrics | Taux d'erreur > 5% |
| Auditer les actions | Logs | Qui a modifié quoi |
| Identifier les goulots | Traces | Service le plus lent |

## Cas d'usage assurance

### Scénario : Création de devis lente

**1. Métriques** (détection)
```
Alerte: latence_p99 > 2s sur /quotes
Dashboard: pic de latence à 14h30
```

**2. Traces** (localisation)
```
Trace de la requête lente:
Gateway        [──]           50ms
Quote Engine   [────]         100ms
External Rating[──────────────] 1800ms  ← Coupable!
```

**3. Logs** (diagnostic)
```
[14:30:15] INFO  quote_engine: Calling external rating
[14:30:15] DEBUG external_rating: Request to api.rating.com
[14:30:17] WARN  external_rating: Slow response: 1800ms
[14:30:17] DEBUG external_rating: Response: {"rate": 450}
```

## Corrélation entre piliers

```
                    Trace ID: abc123
                         │
    ┌────────────────────┼────────────────────┐
    │                    │                    │
    ▼                    ▼                    ▼
┌───────┐          ┌───────────┐        ┌──────────┐
│ LOGS  │          │  TRACES   │        │ METRICS  │
│       │          │           │        │          │
│trace: │          │ Span A    │        │latency=  │
│abc123 │◄────────▶│   │       │        │200ms     │
│       │          │   ▼       │        │          │
│       │          │ Span B    │───────▶│requests= │
│       │          │   │       │        │1         │
└───────┘          │   ▼       │        └──────────┘
                   │ Span C    │
                   └───────────┘
```

Le **Trace ID** est le lien entre les trois piliers :
- Dans les logs : chaque ligne contient le trace_id
- Dans les traces : c'est l'identifiant de la trace
- Dans les métriques : permet de corréler avec les autres données

## Architecture d'observabilité

```
┌─────────────────────────────────────────────────────────────────┐
│                    SOURCES (Applications)                        │
├──────────────────┬──────────────────┬────────────────────────────┤
│                  │                  │                            │
│  Quote Engine    │  Policy Admin    │  Claims                    │
│  ├── Logs        │  ├── Logs        │  ├── Logs                  │
│  ├── Metrics     │  ├── Metrics     │  ├── Metrics               │
│  └── Traces      │  └── Traces      │  └── Traces                │
│                  │                  │                            │
└────────┬─────────┴────────┬─────────┴────────────┬───────────────┘
         │                  │                      │
         ▼                  ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    COLLECTE & STOCKAGE                           │
├────────────────┬────────────────────┬────────────────────────────┤
│                │                    │                            │
│  Log Store     │  Metrics Store     │  Trace Store               │
│  (Elasticsearch)│  (Prometheus)     │  (Jaeger)                  │
│                │                    │                            │
└────────┬───────┴─────────┬──────────┴────────────┬───────────────┘
         │                 │                       │
         ▼                 ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    VISUALISATION & ALERTING                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Grafana / Kibana / Jaeger UI                                   │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Dashboards  │  │   Alertes    │  │  Exploration │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Maturité d'observabilité

| Niveau | Logs | Metrics | Traces | Description |
|--------|------|---------|--------|-------------|
| 0 | printf | Aucune | Aucun | Debugging manuel |
| 1 | Centralisés | Basiques | Aucun | Recherche de logs |
| 2 | Structurés | Dashboard | Request ID | Corrélation basique |
| 3 | Corrélés | Alerting | Distribué | Diagnostic rapide |
| 4 | Intelligents | Prédictif | End-to-end | Proactif |

## Bonnes pratiques

1. **Toujours inclure un ID de corrélation** (trace_id, request_id)
2. **Logs structurés** en JSON, pas de texte libre
3. **Métriques RED** : Rate, Errors, Duration
4. **Traces sur les chemins critiques**
5. **Alerter sur les symptômes**, pas sur les causes
6. **Dashboards par service** ET vue globale
