# EDA-Lab

Simulateur d'architecture événementielle (Event Driven Architecture) pour l'apprentissage des patrons EDA dans un contexte d'écosystème d'entreprise.

## Description

EDA-Lab est une application académique permettant de simuler l'interopérabilité entre différents domaines métier (Bancaire, Assurance Personne, Assurance Dommage) via une architecture événementielle basée sur Apache Kafka.

### Objectifs pédagogiques

- Comprendre les patrons EDA (Pub/Sub, Event Sourcing, CQRS, Saga)
- Expérimenter avec Kafka et Avro/Schema Registry
- Observer les flux d'événements en temps réel
- Mesurer et analyser les performances

## Prérequis

| Outil | Version | Notes |
|-------|---------|-------|
| Docker Desktop | 4.x+ | Avec WSL2 activé (Windows) |
| Go | 1.21+ | Pour les services backend |
| Node.js | 20 LTS | Pour le frontend React |
| Make | 3.8+ | Pour les commandes de build |

## Quick Start

```bash
# Cloner le repository
git clone https://github.com/edalab/eda-lab.git
cd eda-lab

# Démarrer l'infrastructure (Kafka, PostgreSQL, Schema Registry)
make infra-up

# Attendre que l'infrastructure soit prête (~30s)
make test-infra

# Démarrer les services applicatifs
make services-up

# Ouvrir l'interface web
# http://localhost:5173 (dev) ou http://localhost:3000 (prod)
```

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Simulator  │────▶│    Kafka    │────▶│  Bancaire   │
│  (Producer) │     │   (Broker)  │     │ (Consumer)  │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Gateway   │
                    │ (WebSocket) │
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Web UI    │
                    │   (React)   │
                    └─────────────┘
```

## Structure du projet

```
eda-lab/
├── services/           # Microservices Go
│   ├── simulator/      # Générateur d'événements
│   ├── bancaire/       # Service domaine bancaire
│   └── gateway/        # API Gateway + WebSocket
├── pkg/                # Bibliothèques partagées Go
│   ├── config/         # Configuration
│   ├── kafka/          # Client Kafka/Avro
│   ├── database/       # Client PostgreSQL
│   ├── events/         # Types d'événements
│   └── observability/  # Métriques, tracing, logging
├── schemas/            # Schémas Avro
│   └── bancaire/       # Schémas domaine bancaire
├── web-ui/             # Frontend React
├── infra/              # Configuration infrastructure
│   ├── kafka/          # Config Kafka
│   ├── prometheus/     # Config Prometheus
│   └── grafana/        # Dashboards Grafana
├── scripts/            # Scripts utilitaires
├── tests/              # Tests d'intégration et E2E
├── config/             # Fichiers de configuration
└── docs/               # Documentation
```

## Commandes disponibles

```bash
# Infrastructure
make infra-up          # Démarre Kafka, PostgreSQL, Schema Registry
make infra-down        # Arrête l'infrastructure
make infra-logs        # Affiche les logs
make infra-clean       # Supprime les volumes

# Services
make services-up       # Démarre tous les services
make services-down     # Arrête les services

# Tests
make test-unit         # Tests unitaires
make test-integration  # Tests d'intégration
make test-e2e          # Tests end-to-end
make test-infra        # Valide l'infrastructure

# Développement
make dev               # Mode développement (hot reload)
make build             # Build tous les services

# Kafka
make kafka-topics      # Liste les topics
make kafka-create-topic TOPIC=<name>  # Crée un topic

# Utilitaires
make clean             # Nettoie les artefacts
make help              # Affiche l'aide
```

## Documentation

| Document | Description |
|----------|-------------|
| [PDR.MD](PDR.MD) | Product Definition Record - Spécifications complètes |
| [PLAN.MD](PLAN.MD) | Plan d'implémentation détaillé |
| [TODO.MD](TODO.MD) | Liste des tâches |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Documentation technique |
| [docs/patterns/](docs/patterns/) | Guides des patrons EDA |

## Stack technique

- **Backend**: Go 1.21
- **Message Broker**: Apache Kafka (Confluent Platform, mode KRaft)
- **Schema Registry**: Confluent Schema Registry
- **Sérialisation**: Apache Avro
- **Base de données**: PostgreSQL 16
- **Frontend**: React 18 + React Flow + Tailwind CSS
- **Observabilité**: Prometheus, Grafana, Jaeger

## Licence

MIT

---

**EDA-Lab** - Un projet académique pour l'apprentissage des architectures événementielles.
