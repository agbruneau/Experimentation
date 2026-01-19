# Architecture EDA-Lab

## Vue d'ensemble

EDA-Lab est une application pédagogique démontrant les patrons d'architecture événementielle (Event-Driven Architecture - EDA) dans un contexte d'écosystème d'entreprise financière.

## Diagramme de contexte (C4 - Level 1)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              UTILISATEUR                                 │
│                          (Étudiant/Développeur)                         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 │ HTTP / WebSocket
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                          │
│                            EDA-LAB SYSTEM                               │
│                                                                          │
│  ┌────────────┐    ┌─────────────┐    ┌────────────────────────────┐   │
│  │  Web UI    │───▶│   Gateway   │───▶│     Services Métier        │   │
│  │  (React)   │    │             │    │  (Simulator, Bancaire...)  │   │
│  └────────────┘    └──────┬──────┘    └────────────────────────────┘   │
│                           │                        │                    │
│                           │                        ▼                    │
│                           │            ┌───────────────────────┐       │
│                           └───────────▶│   Apache Kafka        │       │
│                                        │   (Message Broker)    │       │
│                                        └───────────────────────┘       │
│                                                    │                    │
│                                                    ▼                    │
│                                        ┌───────────────────────┐       │
│                                        │    PostgreSQL         │       │
│                                        │    (Persistence)      │       │
│                                        └───────────────────────┘       │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Diagramme de conteneurs (C4 - Level 2)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                  EDA-LAB                                      │
│                                                                               │
│  ┌─────────────┐                                                             │
│  │   Web UI    │ React + React Flow + Tailwind                               │
│  │   :5173     │ Visualisation en temps réel des flux d'événements           │
│  └──────┬──────┘                                                             │
│         │ HTTP/WS                                                             │
│         ▼                                                                     │
│  ┌─────────────┐                                                             │
│  │   Gateway   │ Go + Chi + Gorilla WebSocket                                │
│  │   :8080     │ API Gateway, Proxy, Hub WebSocket                           │
│  └──────┬──────┘                                                             │
│         │                                                                     │
│    ┌────┴────┬───────────────────┐                                          │
│    │         │                   │                                          │
│    ▼         ▼                   ▼                                          │
│ ┌────────┐ ┌────────┐      ┌──────────────┐                                 │
│ │Simulator│ │Bancaire│      │ Schema       │                                 │
│ │ :8081  │ │ :8082  │      │ Registry     │                                 │
│ │        │ │        │      │ :8081        │                                 │
│ │Producer│ │Consumer│      │              │                                 │
│ └───┬────┘ └───┬────┘      └──────────────┘                                 │
│     │          │                  │                                          │
│     │          │                  │ Schema validation                        │
│     │          │                  │                                          │
│     ▼          ▼                  ▼                                          │
│  ┌─────────────────────────────────────┐                                    │
│  │           Apache Kafka              │                                    │
│  │           :9092 (external)          │                                    │
│  │           :29092 (internal)         │                                    │
│  │                                     │                                    │
│  │  Topics:                            │                                    │
│  │  - bancaire.compte.ouvert           │                                    │
│  │  - bancaire.depot.effectue          │                                    │
│  │  - bancaire.virement.emis           │                                    │
│  │  - system.dlq                       │                                    │
│  └─────────────────────────────────────┘                                    │
│                                                                               │
│  ┌─────────────┐                     ┌─────────────┐                        │
│  │ PostgreSQL  │                     │ Prometheus  │                        │
│  │ :5432       │                     │ :9090       │                        │
│  │             │                     │             │                        │
│  │ Schemas:    │                     │ Métriques   │                        │
│  │ - bancaire  │                     │ services    │                        │
│  └─────────────┘                     └──────┬──────┘                        │
│                                              │                               │
│                                              ▼                               │
│                                       ┌─────────────┐                       │
│                                       │  Grafana    │                       │
│                                       │  :3000      │                       │
│                                       │  Dashboards │                       │
│                                       └─────────────┘                       │
│                                                                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Composants par service

### Service Simulator

```
services/simulator/
├── cmd/simulator/main.go       # Point d'entrée
└── internal/
    ├── generator/
    │   ├── fake_data.go        # Générateur de données fictives françaises
    │   └── event_generator.go  # Générateurs d'événements (CompteOuvert, etc.)
    ├── simulation/
    │   └── manager.go          # Orchestration des simulations
    └── api/
        └── handler.go          # Endpoints REST (start, stop, status, produce)
```

**Responsabilités:**
- Générer des données bancaires réalistes (IBAN français, noms, montants)
- Produire des événements vers Kafka avec sérialisation Avro
- Contrôler le débit de génération (rate limiting)
- Exposer une API REST pour le contrôle de la simulation

### Service Bancaire

```
services/bancaire/
├── cmd/bancaire/main.go        # Point d'entrée
├── migrations/
│   └── 001_create_comptes.sql  # Schéma de base de données
└── internal/
    ├── domain/
    │   └── compte.go           # Modèles de domaine (Compte, Transaction)
    ├── repository/
    │   └── compte_repository.go # Accès PostgreSQL
    ├── handler/
    │   └── event_handler.go    # Handlers d'événements Kafka
    └── api/
        └── handler.go          # API REST pour les queries
```

**Responsabilités:**
- Consommer les événements Kafka (CompteOuvert, DepotEffectue, VirementEmis)
- Persister les données dans PostgreSQL
- Assurer l'idempotence du traitement des événements
- Exposer une API REST pour les requêtes (CQRS read side)

### Service Gateway

```
services/gateway/
├── cmd/gateway/main.go         # Point d'entrée
└── internal/
    ├── proxy/
    │   └── service_proxy.go    # Proxy HTTP vers les services
    ├── websocket/
    │   ├── hub.go              # Hub de distribution WebSocket
    │   └── client.go           # Gestion des clients WebSocket
    └── api/
        └── router.go           # Configuration du routage
```

**Responsabilités:**
- Router les requêtes vers les services appropriés
- Gérer les connexions WebSocket pour le temps réel
- Diffuser les événements Kafka aux clients WebSocket
- Agréger les health checks

## Flux de données

### Flux principal: Création de compte

```
1. Simulation Start
   ┌──────────┐
   │  Web UI  │──POST /simulation/start──▶ Gateway ──▶ Simulator
   └──────────┘

2. Event Production
   ┌──────────┐                              ┌────────────────┐
   │Simulator │──CompteOuvert (Avro)───────▶│     Kafka      │
   └──────────┘                              │bancaire.compte │
                                             │    .ouvert     │
                                             └───────┬────────┘
                                                     │
3. Event Consumption                                 │
                                                     ▼
   ┌──────────┐                              ┌──────────────┐
   │Bancaire  │◀────────Consume──────────────│    Kafka     │
   └────┬─────┘                              └──────────────┘
        │
4. Persistence
        │
        ▼
   ┌──────────┐
   │PostgreSQL│
   │ comptes  │
   └──────────┘

5. Real-time Update
   ┌──────────┐                              ┌──────────────┐
   │ Gateway  │◀────────Consume──────────────│    Kafka     │
   └────┬─────┘                              └──────────────┘
        │
        │ WebSocket broadcast
        ▼
   ┌──────────┐
   │  Web UI  │
   └──────────┘
```

## Schémas Avro

### CompteOuvert

```json
{
  "type": "record",
  "name": "CompteOuvert",
  "namespace": "com.edalab.bancaire.events",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "compte_id", "type": "string"},
    {"name": "client_id", "type": "string"},
    {"name": "type_compte", "type": {"type": "enum", "name": "TypeCompte", "symbols": ["COURANT", "EPARGNE", "JOINT"]}},
    {"name": "devise", "type": "string", "default": "EUR"},
    {"name": "solde_initial", "type": {"type": "bytes", "logicalType": "decimal", "precision": 18, "scale": 2}},
    {"name": "metadata", "type": ["null", {"type": "map", "values": "string"}], "default": null}
  ]
}
```

### Conventions de nommage des topics

```
<domaine>.<entité>.<action>

Exemples:
- bancaire.compte.ouvert
- bancaire.depot.effectue
- bancaire.virement.emis
- bancaire.virement.recu
- system.dlq (Dead Letter Queue)
```

## Observabilité

### Métriques (Prometheus)

| Métrique | Type | Description |
|----------|------|-------------|
| `messages_produced_total` | Counter | Nombre total de messages produits |
| `messages_consumed_total` | Counter | Nombre total de messages consommés |
| `message_latency_seconds` | Histogram | Latence de traitement des messages |
| `processing_errors_total` | Counter | Nombre d'erreurs de traitement |
| `kafka_consumer_lag` | Gauge | Retard de consommation Kafka |

### Dashboards Grafana

1. **Services Overview**: État de santé des services, métriques de base
2. **Kafka Overview**: Débit, lag consommateurs, topics actifs
3. **Performance**: Latences P50/P95/P99, throughput

## Patterns EDA implémentés

### MVP (Itération 1)

| Pattern | Description | Implémentation |
|---------|-------------|----------------|
| **Pub/Sub** | Publication/Souscription découplée | Simulator publie, Bancaire souscrit |
| **Event Notification** | Notification légère d'événement | Messages Avro compacts |
| **Idempotent Consumer** | Traitement idempotent | Table `processed_events` |

### Itérations futures

| Pattern | Description | Planifié |
|---------|-------------|----------|
| Event Sourcing | État dérivé des événements | Itération 2 |
| CQRS | Séparation lecture/écriture | Itération 2 |
| Saga | Orchestration de transactions | Itération 3 |
| Event Replay | Reconstruction d'état | Itération 3 |

## Configuration

### Variables d'environnement

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_SCHEMA_REGISTRY_URL=http://localhost:8081

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=edalab
POSTGRES_USER=edalab
POSTGRES_PASSWORD=edalab_password

# Service Ports
GATEWAY_PORT=8080
SIMULATOR_PORT=8081
BANCAIRE_PORT=8082

# Observability
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000
```

## Sécurité

### Considérations actuelles (MVP)

- Pas d'authentification (environnement académique)
- CORS configuré pour le développement local
- Pas de chiffrement Kafka (PLAINTEXT)

### Recommandations production

- Activer SASL/SSL pour Kafka
- Implémenter OAuth2/JWT pour les APIs
- Configurer mTLS entre services
- Chiffrer les données sensibles

## ADRs (Architecture Decision Records)

| ADR | Décision | Justification |
|-----|----------|---------------|
| 001 | Kafka comme broker | Écosystème riche, Confluent Platform |
| 002 | Avro pour sérialisation | Schema Registry, évolution de schémas |
| 003 | Go pour le backend | Performance, simplicité, goroutines |
| 004 | PostgreSQL | Robustesse, SQL standard |
| 005 | React Flow | Visualisation graphique native |

---

## Références

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/)
- [Event-Driven Architecture Patterns](https://martinfowler.com/articles/201701-event-driven.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
