# Plan de Réalisation - kafka-eda-lab

## Vue d'ensemble

Ce document détaille le plan de réalisation du projet `kafka-eda-lab` décomposé en instructions exécutables par Claude Code. Chaque instruction est conçue pour être :
- **Autonome** : Testable indépendamment
- **Incrémentale** : S'appuie sur les étapes précédentes
- **Vérifiable** : Critères de validation clairs

---

## Structure du Plan

```
Phase 1 : Infrastructure de base
├── Étape 1.1 : Structure du projet
├── Étape 1.2 : Configuration Docker Kafka
├── Étape 1.3 : Stack d'observabilité
└── Étape 1.4 : Validation infrastructure

Phase 2 : Patron Producteur/Consommateur
├── Étape 2.1 : Service Quotation
├── Étape 2.2 : Service Souscription
├── Étape 2.3 : Service Réclamation
├── Étape 2.4 : Dashboard minimal
└── Étape 2.5 : Simulateur d'événements

Phase 3 : Event Sourcing
├── Étape 3.1 : Refactoring pour Event Store
├── Étape 3.2 : Reconstruction d'état
└── Étape 3.3 : Snapshots

Phase 4 : CQRS
├── Étape 4.1 : Séparation Command/Query
├── Étape 4.2 : Vues matérialisées
└── Étape 4.3 : Synchronisation

Phase 5 : Saga Choreography
├── Étape 5.1 : Transactions distribuées
├── Étape 5.2 : Compensation
└── Étape 5.3 : Scénarios complexes

Phase 6 : Dead Letter Queue
├── Étape 6.1 : Gestion des erreurs
├── Étape 6.2 : Retry et DLQ
└── Étape 6.3 : Scénarios de panne

Phase 7 : Finalisation
├── Étape 7.1 : Documentation complète
├── Étape 7.2 : Tests de charge
└── Étape 7.3 : Polish et release
```

---

# PHASE 1 : Infrastructure de Base

## Étape 1.1 : Structure du Projet

### Sous-étape 1.1.1 : Initialisation du module Go

**Contexte :** Créer la structure de base du projet Go avec les dossiers nécessaires.

**Prérequis :** Aucun

**Critère de validation :** `go mod tidy` s'exécute sans erreur

```text
INSTRUCTION 1.1.1 - Initialisation du module Go

Objectif : Initialiser la structure de base du projet kafka-eda-lab

Actions à réaliser :
1. Créer le fichier go.mod avec le module "github.com/[user]/kafka-eda-lab"
2. Créer l'arborescence de dossiers suivante :
   - cmd/ (avec sous-dossiers : quotation, souscription, reclamation, dashboard, simulator)
   - internal/ (avec sous-dossiers : kafka, models, observability, database)
   - pkg/
   - schemas/
   - web/templates/, web/static/, web/handlers/
   - docker/
   - docs/
   - tests/integration/, tests/load/
3. Créer un fichier main.go minimal dans cmd/dashboard/ qui affiche "kafka-eda-lab starting..."
4. Créer un Makefile avec les commandes de base (build, clean)

Test de validation :
- Exécuter : go build ./cmd/dashboard
- Exécuter : go mod tidy
- Vérifier que le binaire s'exécute et affiche le message

Ne pas créer de code métier, uniquement la structure.
```

---

### Sous-étape 1.1.2 : Configuration du Makefile complet

**Contexte :** Le Makefile doit fournir toutes les commandes utilitaires définies dans le cahier des charges.

**Prérequis :** Sous-étape 1.1.1 complétée

**Critère de validation :** `make help` affiche toutes les commandes disponibles

```text
INSTRUCTION 1.1.2 - Configuration du Makefile

Objectif : Créer un Makefile complet avec toutes les commandes utilitaires

Contexte : Le projet kafka-eda-lab utilise Docker Compose pour l'infrastructure.
Le Makefile doit simplifier l'utilisation pour un utilisateur Windows.

Actions à réaliser :
1. Mettre à jour le Makefile avec les commandes suivantes :
   - make up : docker-compose up -d
   - make down : docker-compose down
   - make reset : docker-compose down -v && docker-compose up -d
   - make logs : docker-compose logs -f
   - make status : docker-compose ps
   - make build : go build pour tous les services dans cmd/
   - make test : go test ./...
   - make test-integration : go test ./tests/integration/...
   - make test-load : (placeholder pour k6)
   - make dashboard : start http://localhost:8080 (Windows)
   - make grafana : start http://localhost:3000
   - make jaeger : start http://localhost:16686
   - make help : affiche la liste des commandes

2. Ajouter des variables pour les chemins et versions
3. Ajouter une cible .PHONY pour chaque commande

Test de validation :
- Exécuter : make help
- Vérifier que toutes les commandes sont listées avec leur description
- Exécuter : make build (doit compiler sans erreur même si les services sont vides)
```

---

### Sous-étape 1.1.3 : Fichier .gitignore et configuration

**Contexte :** Configurer les fichiers ignorés et les configurations de base.

**Prérequis :** Sous-étape 1.1.2 complétée

**Critère de validation :** Les fichiers sensibles et générés sont ignorés

```text
INSTRUCTION 1.1.3 - Configuration Git et environnement

Objectif : Configurer les fichiers de projet (.gitignore, .env.example, README)

Actions à réaliser :
1. Créer .gitignore avec :
   - Binaires Go (*.exe, /bin/)
   - Fichiers de données (*.db, /data/)
   - Fichiers d'environnement (.env)
   - Dossiers IDE (.idea/, .vscode/)
   - Dossiers Docker volumes
   - Fichiers de test coverage

2. Créer .env.example avec les variables d'environnement :
   - KAFKA_BOOTSTRAP_SERVERS=localhost:9092
   - SCHEMA_REGISTRY_URL=http://localhost:8081
   - GRAFANA_PORT=3000
   - JAEGER_PORT=16686
   - DASHBOARD_PORT=8080

3. Créer un README.md initial avec :
   - Nom et description du projet
   - Prérequis (Docker Desktop, Go 1.21+)
   - Instructions de démarrage rapide
   - Liste des commandes make disponibles

Test de validation :
- Vérifier que .gitignore exclut les fichiers appropriés
- Vérifier que README.md est lisible et complet
```

---

## Étape 1.2 : Configuration Docker Kafka

### Sous-étape 1.2.1 : Docker Compose Kafka KRaft minimal

**Contexte :** Configurer un broker Kafka en mode KRaft (sans Zookeeper).

**Prérequis :** Étape 1.1 complétée

**Critère de validation :** `docker-compose up kafka` démarre sans erreur

```text
INSTRUCTION 1.2.1 - Docker Compose Kafka KRaft

Objectif : Créer la configuration Docker Compose pour Kafka en mode KRaft

Contexte :
- Utiliser l'image apache/kafka:3.7.0
- Mode KRaft (sans Zookeeper)
- Un seul broker pour la simulation
- Plateforme Windows (Docker Desktop)

Actions à réaliser :
1. Créer docker-compose.yml à la racine avec le service kafka :
   - Image : apache/kafka:3.7.0
   - Ports : 9092:9092
   - Variables d'environnement pour KRaft :
     * KAFKA_NODE_ID=1
     * KAFKA_PROCESS_ROLES=broker,controller
     * KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
     * KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092
     * KAFKA_CONTROLLER_QUORUM_VOTERS=1@kafka:9093
     * KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER
     * KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
     * KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
     * KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1
     * KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1
   - Volume pour la persistance : kafka-data:/var/lib/kafka/data
   - Healthcheck avec kafka-broker-api-versions

2. Déclarer le volume kafka-data

Test de validation :
- Exécuter : docker-compose up -d kafka
- Attendre 30 secondes
- Exécuter : docker-compose logs kafka | grep "Kafka Server started"
- Exécuter : docker-compose exec kafka kafka-topics.sh --bootstrap-server localhost:9092 --list
- La commande doit s'exécuter sans erreur (liste vide OK)
```

---

### Sous-étape 1.2.2 : Ajout du Schema Registry

**Contexte :** Ajouter Confluent Schema Registry pour la gestion des schémas Avro.

**Prérequis :** Sous-étape 1.2.1 complétée (Kafka fonctionnel)

**Critère de validation :** Schema Registry accessible sur http://localhost:8081

```text
INSTRUCTION 1.2.2 - Ajout du Schema Registry

Objectif : Ajouter Confluent Schema Registry au docker-compose

Contexte :
- Le Schema Registry dépend de Kafka
- Il stocke les schémas Avro pour les événements
- Port 8081

Actions à réaliser :
1. Ajouter le service schema-registry au docker-compose.yml :
   - Image : confluentinc/cp-schema-registry:7.6.0
   - Ports : 8081:8081
   - Dépendance : kafka
   - Variables d'environnement :
     * SCHEMA_REGISTRY_HOST_NAME=schema-registry
     * SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS=kafka:9092
     * SCHEMA_REGISTRY_LISTENERS=http://0.0.0.0:8081
   - Healthcheck : curl -f http://localhost:8081/subjects

2. Mettre à jour le service kafka pour attendre qu'il soit prêt avant schema-registry

Test de validation :
- Exécuter : docker-compose up -d
- Attendre que tous les services soient healthy
- Exécuter : curl http://localhost:8081/subjects
- Doit retourner : [] (liste vide)
- Exécuter : curl http://localhost:8081/config
- Doit retourner la configuration par défaut
```

---

### Sous-étape 1.2.3 : Création des topics Kafka

**Contexte :** Créer les topics nécessaires pour les événements métier.

**Prérequis :** Sous-étape 1.2.2 complétée

**Critère de validation :** Les 8 topics sont créés et listés

```text
INSTRUCTION 1.2.3 - Création des topics Kafka

Objectif : Créer un script d'initialisation des topics Kafka

Contexte : Les événements métier définis sont :
- DevisGenere, DevisExpire (Quotation)
- ContratEmis, ContratModifie, ContratResilie (Souscription)
- SinistreDeclare, SinistreEvalue, IndemnisationEffectuee (Réclamation)

Actions à réaliser :
1. Créer le fichier docker/kafka/create-topics.sh :
   - Attendre que Kafka soit prêt
   - Créer les topics avec :
     * 3 partitions chacun
     * Replication factor = 1
     * Retention = 7 jours
   - Topics à créer :
     * quotation.devis-genere
     * quotation.devis-expire
     * souscription.contrat-emis
     * souscription.contrat-modifie
     * souscription.contrat-resilie
     * reclamation.sinistre-declare
     * reclamation.sinistre-evalue
     * reclamation.indemnisation-effectuee
   - Ajouter un topic DLQ : dlq.errors

2. Créer un service kafka-init dans docker-compose.yml :
   - Utilise l'image apache/kafka:3.7.0
   - Monte le script create-topics.sh
   - S'exécute une seule fois (restart: "no")
   - Dépend de kafka

3. Mettre à jour le Makefile :
   - Ajouter : make topics (liste les topics)

Test de validation :
- Exécuter : docker-compose up -d
- Attendre la fin de kafka-init
- Exécuter : make topics
- Vérifier que les 9 topics sont listés
```

---

### Sous-étape 1.2.4 : Interface Kafka UI

**Contexte :** Ajouter une interface web pour visualiser Kafka.

**Prérequis :** Sous-étape 1.2.3 complétée

**Critère de validation :** Kafka UI accessible sur http://localhost:8090

```text
INSTRUCTION 1.2.4 - Ajout de Kafka UI

Objectif : Ajouter une interface graphique pour explorer Kafka

Contexte :
- Kafka UI permet de visualiser topics, messages, consumer groups
- Utile pour le débogage et l'apprentissage
- Port 8090 pour éviter conflit avec le dashboard (8080)

Actions à réaliser :
1. Ajouter le service kafka-ui au docker-compose.yml :
   - Image : provectuslabs/kafka-ui:latest
   - Ports : 8090:8080
   - Variables d'environnement :
     * KAFKA_CLUSTERS_0_NAME=kafka-eda-lab
     * KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092
     * KAFKA_CLUSTERS_0_SCHEMAREGISTRY=http://schema-registry:8081
   - Dépendances : kafka, schema-registry

2. Mettre à jour le Makefile :
   - Ajouter : make kafka-ui (ouvre http://localhost:8090)

Test de validation :
- Exécuter : docker-compose up -d
- Ouvrir http://localhost:8090 dans un navigateur
- Vérifier que les topics sont visibles
- Vérifier que le Schema Registry est connecté
```

---

## Étape 1.3 : Stack d'Observabilité

### Sous-étape 1.3.1 : Prometheus

**Contexte :** Configurer Prometheus pour la collecte des métriques.

**Prérequis :** Étape 1.2 complétée

**Critère de validation :** Prometheus accessible sur http://localhost:9090

```text
INSTRUCTION 1.3.1 - Configuration de Prometheus

Objectif : Ajouter Prometheus pour la collecte des métriques

Contexte :
- Prometheus collecte les métriques des services Go et de Kafka
- Configuration via fichier prometheus.yml
- Port 9090

Actions à réaliser :
1. Créer le dossier docker/prometheus/

2. Créer docker/prometheus/prometheus.yml :
   - Global : scrape_interval: 15s
   - Scrape configs pour :
     * prometheus (self-monitoring)
     * kafka (JMX exporter, port 7071 - à ajouter plus tard)
     * quotation (port 8081/metrics - à venir)
     * souscription (port 8082/metrics - à venir)
     * reclamation (port 8083/metrics - à venir)
     * dashboard (port 8080/metrics - à venir)

3. Ajouter le service prometheus au docker-compose.yml :
   - Image : prom/prometheus:v2.50.0
   - Ports : 9090:9090
   - Volumes :
     * ./docker/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
     * prometheus-data:/prometheus
   - Commande avec retention de 15 jours

4. Déclarer le volume prometheus-data

Test de validation :
- Exécuter : docker-compose up -d prometheus
- Ouvrir http://localhost:9090
- Vérifier Status > Targets (prometheus doit être UP)
- Les autres targets seront DOWN (normal, services pas encore créés)
```

---

### Sous-étape 1.3.2 : Grafana avec provisioning

**Contexte :** Configurer Grafana avec datasources pré-configurées.

**Prérequis :** Sous-étape 1.3.1 complétée

**Critère de validation :** Grafana accessible avec Prometheus comme datasource

```text
INSTRUCTION 1.3.2 - Configuration de Grafana

Objectif : Ajouter Grafana avec provisioning automatique des datasources

Contexte :
- Grafana visualise les métriques de Prometheus et les logs de Loki
- Configuration via provisioning (pas de setup manuel)
- Port 3000, accès anonyme pour simplifier

Actions à réaliser :
1. Créer l'arborescence docker/grafana/provisioning/datasources/

2. Créer docker/grafana/provisioning/datasources/datasources.yml :
   - Datasource Prometheus :
     * name: Prometheus
     * type: prometheus
     * url: http://prometheus:9090
     * isDefault: true
   - Datasource Loki (préparation) :
     * name: Loki
     * type: loki
     * url: http://loki:3100

3. Créer docker/grafana/grafana.ini :
   - [auth.anonymous] enabled = true
   - [auth.anonymous] org_role = Admin
   - [security] admin_password = admin

4. Ajouter le service grafana au docker-compose.yml :
   - Image : grafana/grafana:10.3.0
   - Ports : 3000:3000
   - Volumes :
     * ./docker/grafana/provisioning:/etc/grafana/provisioning
     * ./docker/grafana/grafana.ini:/etc/grafana/grafana.ini
     * grafana-data:/var/lib/grafana
   - Dépendance : prometheus

5. Déclarer le volume grafana-data

Test de validation :
- Exécuter : docker-compose up -d grafana
- Ouvrir http://localhost:3000
- Aller dans Configuration > Data Sources
- Vérifier que Prometheus est listé et fonctionnel (bouton "Test")
```

---

### Sous-étape 1.3.3 : Loki pour les logs

**Contexte :** Configurer Loki pour la centralisation des logs.

**Prérequis :** Sous-étape 1.3.2 complétée

**Critère de validation :** Loki accessible et connecté à Grafana

```text
INSTRUCTION 1.3.3 - Configuration de Loki

Objectif : Ajouter Loki pour la centralisation des logs

Contexte :
- Loki stocke les logs de tous les services
- Les services Go enverront leurs logs à Loki via le driver Docker ou Promtail
- Intégration avec Grafana déjà préparée

Actions à réaliser :
1. Créer docker/loki/loki-config.yml :
   - auth_enabled: false
   - server: http_listen_port: 3100
   - ingester: lifecycler avec ring kvstore inmemory
   - schema_config: configs avec boltdb-shipper et filesystem
   - storage_config: boltdb_shipper et filesystem dans /loki
   - limits_config: reject_old_samples: true, max_entries_limit_per_query: 5000
   - chunk_store_config: max_look_back_period: 0s

2. Ajouter le service loki au docker-compose.yml :
   - Image : grafana/loki:2.9.4
   - Ports : 3100:3100
   - Volumes :
     * ./docker/loki/loki-config.yml:/etc/loki/local-config.yaml
     * loki-data:/loki
   - Commande : -config.file=/etc/loki/local-config.yaml

3. Déclarer le volume loki-data

4. Mettre à jour grafana pour dépendre de loki

Test de validation :
- Exécuter : docker-compose up -d loki grafana
- Ouvrir http://localhost:3000
- Aller dans Explore, sélectionner Loki
- La connexion doit être OK (pas de logs encore, c'est normal)
- Exécuter : curl http://localhost:3100/ready (doit retourner "ready")
```

---

### Sous-étape 1.3.4 : Jaeger pour le tracing

**Contexte :** Configurer Jaeger pour le tracing distribué.

**Prérequis :** Sous-étape 1.3.3 complétée

**Critère de validation :** Jaeger UI accessible sur http://localhost:16686

```text
INSTRUCTION 1.3.4 - Configuration de Jaeger

Objectif : Ajouter Jaeger pour le tracing distribué

Contexte :
- Jaeger permet de suivre le parcours des événements entre services
- Utiliser l'image all-in-one pour simplifier
- Les services Go utiliseront OpenTelemetry pour envoyer les traces

Actions à réaliser :
1. Ajouter le service jaeger au docker-compose.yml :
   - Image : jaegertracing/all-in-one:1.54
   - Ports :
     * 16686:16686 (UI)
     * 4317:4317 (OTLP gRPC)
     * 4318:4318 (OTLP HTTP)
   - Variables d'environnement :
     * COLLECTOR_OTLP_ENABLED=true

2. Mettre à jour le Makefile :
   - Vérifier que make jaeger ouvre http://localhost:16686

Test de validation :
- Exécuter : docker-compose up -d jaeger
- Ouvrir http://localhost:16686
- L'interface Jaeger doit s'afficher
- Aucune trace n'est encore présente (normal)
```

---

### Sous-étape 1.3.5 : Dashboard Grafana - Kafka Overview

**Contexte :** Créer le premier dashboard Grafana pour Kafka.

**Prérequis :** Sous-étape 1.3.4 complétée

**Critère de validation :** Dashboard visible dans Grafana

```text
INSTRUCTION 1.3.5 - Dashboard Grafana Kafka Overview

Objectif : Créer le dashboard de monitoring Kafka

Contexte :
- Le dashboard affichera les métriques Kafka (quand disponibles)
- Pour l'instant, créer la structure avec des panels placeholder
- Le provisioning automatique chargera le dashboard au démarrage

Actions à réaliser :
1. Créer docker/grafana/provisioning/dashboards/dashboards.yml :
   - apiVersion: 1
   - providers:
     * name: 'default'
     * folder: 'kafka-eda-lab'
     * type: file
     * options: path: /etc/grafana/provisioning/dashboards

2. Créer docker/grafana/provisioning/dashboards/kafka-overview.json :
   - Dashboard avec :
     * Titre : "Kafka Overview"
     * Tags : ["kafka", "infrastructure"]
     * Panels (placeholder pour l'instant) :
       - "Brokers Status" (stat panel)
       - "Messages/sec" (graph panel)
       - "Topics" (table panel)
       - "Consumer Lag" (graph panel)
   - Utiliser des queries Prometheus basiques
   - Les panels afficheront "No data" tant que les métriques ne sont pas exposées

3. Mettre à jour le volume Grafana pour inclure le dossier dashboards

Test de validation :
- Exécuter : docker-compose up -d grafana
- Ouvrir http://localhost:3000
- Aller dans Dashboards > Browse
- Le dossier "kafka-eda-lab" doit contenir "Kafka Overview"
- Le dashboard s'ouvre (panels vides, c'est normal)
```

---

## Étape 1.4 : Validation Infrastructure

### Sous-étape 1.4.1 : Script de vérification de santé

**Contexte :** Créer un script qui vérifie que toute l'infrastructure est opérationnelle.

**Prérequis :** Toutes les sous-étapes précédentes complétées

**Critère de validation :** Le script affiche le statut de tous les services

```text
INSTRUCTION 1.4.1 - Script de vérification de santé

Objectif : Créer un script de validation de l'infrastructure complète

Contexte :
- Le script vérifie que tous les services sont UP
- Affiche un rapport de santé
- Utilisé pour valider la Phase 1

Actions à réaliser :
1. Créer scripts/health-check.sh (compatible Git Bash sur Windows) :
   - Vérifier Kafka : kafka-broker-api-versions via docker exec
   - Vérifier Schema Registry : curl http://localhost:8081/subjects
   - Vérifier Prometheus : curl http://localhost:9090/-/healthy
   - Vérifier Grafana : curl http://localhost:3000/api/health
   - Vérifier Loki : curl http://localhost:3100/ready
   - Vérifier Jaeger : curl http://localhost:16686/
   - Afficher un résumé avec codes couleur (OK/FAIL)

2. Mettre à jour le Makefile :
   - Ajouter : make health (exécute le script)

3. Créer scripts/health-check.ps1 (version PowerShell pour Windows natif) :
   - Même logique que le script bash

Test de validation :
- Exécuter : docker-compose up -d
- Attendre 60 secondes
- Exécuter : make health
- Tous les services doivent être OK
```

---

### Sous-étape 1.4.2 : Documentation de la Phase 1

**Contexte :** Documenter l'infrastructure mise en place.

**Prérequis :** Sous-étape 1.4.1 complétée

**Critère de validation :** Documentation lisible et complète

```text
INSTRUCTION 1.4.2 - Documentation Phase 1

Objectif : Créer la documentation de l'infrastructure

Actions à réaliser :
1. Créer docs/00-introduction.md :
   - Présentation du projet kafka-eda-lab
   - Objectifs pédagogiques
   - Architecture globale (schéma ASCII ou description)

2. Créer docs/01-producteur-consommateur/README.md :
   - Introduction au patron Producteur/Consommateur
   - Concepts Kafka (topics, partitions, offsets)
   - Lien vers l'infrastructure

3. Mettre à jour README.md :
   - Instructions de démarrage complètes
   - Liste des URLs d'accès :
     * Kafka UI : http://localhost:8090
     * Grafana : http://localhost:3000
     * Jaeger : http://localhost:16686
     * Prometheus : http://localhost:9090
   - Commandes make disponibles
   - Troubleshooting basique

Test de validation :
- Lire README.md et suivre les instructions
- Vérifier que l'infrastructure démarre correctement
- Vérifier que tous les liens sont accessibles
```

---

### Sous-étape 1.4.3 : Tag Git v1.0-infra

**Contexte :** Marquer la fin de la Phase 1 avec un tag Git.

**Prérequis :** Sous-étape 1.4.2 complétée

**Critère de validation :** Tag créé et poussé

```text
INSTRUCTION 1.4.3 - Tag Git Phase 1

Objectif : Créer le tag de fin de Phase 1

Actions à réaliser :
1. Vérifier que tous les fichiers sont commités
2. Exécuter les tests de validation finale :
   - make up
   - make health (tous services OK)
   - make down
3. Créer le tag : git tag -a v1.0-infra -m "Phase 1: Infrastructure de base"
4. Documenter dans CHANGELOG.md :
   - ## v1.0-infra
   - Date
   - Liste des composants :
     * Kafka KRaft
     * Schema Registry
     * Prometheus
     * Grafana
     * Loki
     * Jaeger
     * Kafka UI

Test de validation :
- git tag --list doit afficher v1.0-infra
- git checkout v1.0-infra doit fonctionner
- L'infrastructure doit démarrer depuis ce tag
```

---

# PHASE 2 : Patron Producteur/Consommateur

## Étape 2.1 : Service Quotation

### Sous-étape 2.1.1 : Structure et modèles de données

**Contexte :** Créer la structure du service Quotation avec ses modèles.

**Prérequis :** Phase 1 complétée (tag v1.0-infra)

**Critère de validation :** Le code compile sans erreur

```text
INSTRUCTION 2.1.1 - Service Quotation : Structure et modèles

Objectif : Créer la structure du service Quotation avec les modèles de données

Contexte :
- Service de génération de devis (Quotation)
- Produit les événements : DevisGenere, DevisExpire
- Langage : Go
- Base de données : SQLite

Actions à réaliser :
1. Créer internal/models/quotation.go :
   - struct Devis :
     * ID string (UUID)
     * ClientID string
     * TypeBien string (AUTO, HABITATION, AUTRE)
     * Valeur float64
     * Prime float64
     * DateCreation time.Time
     * DateExpiration time.Time
     * Statut string (GENERE, CONVERTI, EXPIRE)

2. Créer internal/models/events.go :
   - struct DevisGenere :
     * DevisID string
     * ClientID string
     * TypeBien string
     * Valeur float64
     * Prime float64
     * Timestamp time.Time
   - struct DevisExpire :
     * DevisID string
     * DateExpiration time.Time
     * Timestamp time.Time

3. Créer cmd/quotation/main.go :
   - Import des packages
   - Fonction main() avec log de démarrage
   - Placeholder pour initialisation DB, Kafka, HTTP

4. Ajouter les dépendances dans go.mod :
   - github.com/google/uuid
   - github.com/mattn/go-sqlite3

Test de validation :
- Exécuter : go build ./cmd/quotation
- Le binaire doit être créé sans erreur
- Exécuter : go test ./internal/models/... (pas de tests encore, doit passer)
```

---

### Sous-étape 2.1.2 : Couche base de données SQLite

**Contexte :** Implémenter la persistance SQLite pour Quotation.

**Prérequis :** Sous-étape 2.1.1 complétée

**Critère de validation :** Tests unitaires passent

```text
INSTRUCTION 2.1.2 - Service Quotation : Base de données SQLite

Objectif : Implémenter la couche de persistance SQLite pour le service Quotation

Contexte :
- SQLite pour la simplicité (fichier embarqué)
- CRUD pour les devis
- Migrations automatiques au démarrage

Actions à réaliser :
1. Créer internal/database/sqlite.go :
   - Fonction NewSQLiteDB(path string) (*sql.DB, error)
   - Fonction InitSchema(db *sql.DB) error (crée les tables)

2. Créer internal/database/quotation_repository.go :
   - Interface QuotationRepository :
     * Create(devis *models.Devis) error
     * GetByID(id string) (*models.Devis, error)
     * GetByClientID(clientID string) ([]*models.Devis, error)
     * UpdateStatus(id string, status string) error
     * GetExpiredDevis() ([]*models.Devis, error)
   - Implémentation SQLiteQuotationRepository

3. Créer internal/database/quotation_repository_test.go :
   - Test Create et GetByID
   - Test UpdateStatus
   - Test GetExpiredDevis
   - Utiliser une DB en mémoire (:memory:) pour les tests

Test de validation :
- Exécuter : go test ./internal/database/... -v
- Tous les tests doivent passer
- Vérifier la couverture : go test ./internal/database/... -cover (objectif > 80%)
```

---

### Sous-étape 2.1.3 : Client Kafka producteur

**Contexte :** Créer le client Kafka pour produire des événements.

**Prérequis :** Sous-étape 2.1.2 complétée

**Critère de validation :** Test d'intégration avec Kafka réel passe

```text
INSTRUCTION 2.1.3 - Service Quotation : Producteur Kafka

Objectif : Implémenter le producteur Kafka pour les événements Quotation

Contexte :
- Utiliser github.com/confluentinc/confluent-kafka-go/v2
- Sérialisation Avro avec Schema Registry
- Événements : DevisGenere, DevisExpire

Actions à réaliser :
1. Ajouter la dépendance :
   - github.com/confluentinc/confluent-kafka-go/v2/kafka
   - github.com/riferrei/srclient

2. Créer schemas/DevisGenere.avsc :
   - Schema Avro pour l'événement DevisGenere
   - Namespace : com.kafkaedalab.quotation
   - Champs correspondant à struct DevisGenere

3. Créer schemas/DevisExpire.avsc :
   - Schema Avro pour l'événement DevisExpire

4. Créer internal/kafka/producer.go :
   - struct KafkaProducer avec client et schemaRegistry
   - Fonction NewKafkaProducer(bootstrapServers, schemaRegistryURL string)
   - Méthode Produce(topic string, key string, value interface{}) error
   - Méthode Close()

5. Créer internal/kafka/avro.go :
   - Fonction SerializeAvro(schemaRegistry, subject string, data interface{}) ([]byte, error)
   - Gestion du cache des schémas

6. Créer tests/integration/kafka_producer_test.go :
   - Test avec Kafka réel (docker-compose doit être up)
   - Produire un événement DevisGenere
   - Vérifier dans Kafka UI que le message est présent

Test de validation :
- Démarrer l'infrastructure : make up
- Exécuter : go test ./tests/integration/... -v -tags=integration
- Le test doit produire un message visible dans Kafka UI
```

---

### Sous-étape 2.1.4 : Logique métier Quotation

**Contexte :** Implémenter la logique métier de génération de devis.

**Prérequis :** Sous-étape 2.1.3 complétée

**Critère de validation :** Tests unitaires de la logique métier passent

```text
INSTRUCTION 2.1.4 - Service Quotation : Logique métier

Objectif : Implémenter la logique métier de génération et expiration des devis

Contexte :
- Génération de devis avec calcul de prime
- Expiration automatique après 30 jours
- Publication des événements Kafka

Actions à réaliser :
1. Créer internal/services/quotation_service.go :
   - struct QuotationService avec repository et producer
   - Fonction NewQuotationService(repo, producer)
   - Méthode GenererDevis(clientID, typeBien string, valeur float64) (*models.Devis, error) :
     * Calcul de la prime (règles simples : 2% pour AUTO, 1.5% pour HABITATION)
     * Création du devis en DB
     * Publication de DevisGenere
   - Méthode VerifierExpirations() error :
     * Récupère les devis expirés
     * Met à jour le statut
     * Publie DevisExpire pour chaque

2. Créer internal/services/quotation_service_test.go :
   - Mock du repository et du producer
   - Test GenererDevis avec différents types de biens
   - Test calcul de prime
   - Test VerifierExpirations

3. Créer internal/services/mocks/quotation_mocks.go :
   - Mock QuotationRepository
   - Mock KafkaProducer

Test de validation :
- Exécuter : go test ./internal/services/... -v
- Tous les tests doivent passer
- Vérifier que les événements sont produits correctement (via mock)
```

---

### Sous-étape 2.1.5 : API HTTP et métriques

**Contexte :** Exposer l'API HTTP et les métriques Prometheus.

**Prérequis :** Sous-étape 2.1.4 complétée

**Critère de validation :** API accessible et métriques exposées

```text
INSTRUCTION 2.1.5 - Service Quotation : API HTTP et métriques

Objectif : Exposer l'API HTTP REST et les métriques Prometheus

Contexte :
- API pour déclencher la génération de devis (utilisé par le simulateur)
- Métriques Prometheus pour l'observabilité
- Port 8081

Actions à réaliser :
1. Ajouter les dépendances :
   - github.com/gorilla/mux
   - github.com/prometheus/client_golang/prometheus

2. Créer internal/observability/metrics.go :
   - Compteurs :
     * devis_generes_total (counter)
     * devis_expires_total (counter)
   - Histogrammes :
     * devis_generation_duration_seconds (histogram)
   - Fonction RegisterMetrics()
   - Fonction IncrementDevisGeneres(), etc.

3. Créer cmd/quotation/handlers.go :
   - Handler POST /api/devis : génère un devis
     * Body : { "clientId": "...", "typeBien": "AUTO", "valeur": 25000 }
     * Response : { "devisId": "...", "prime": 500.0 }
   - Handler GET /api/devis/{id} : récupère un devis
   - Handler GET /health : healthcheck
   - Handler GET /metrics : métriques Prometheus

4. Mettre à jour cmd/quotation/main.go :
   - Initialisation complète :
     * Configuration via variables d'environnement
     * Connexion SQLite
     * Connexion Kafka
     * Démarrage serveur HTTP sur :8081
   - Graceful shutdown

5. Créer docker/quotation/Dockerfile :
   - Multi-stage build
   - Image finale légère (alpine ou scratch)

6. Ajouter le service quotation au docker-compose.yml :
   - Build depuis docker/quotation/Dockerfile
   - Ports : 8081:8081
   - Variables d'environnement
   - Dépendances : kafka, schema-registry
   - Healthcheck

Test de validation :
- Exécuter : make up
- Exécuter : curl http://localhost:8081/health (doit retourner OK)
- Exécuter : curl -X POST http://localhost:8081/api/devis -d '{"clientId":"C001","typeBien":"AUTO","valeur":25000}'
- Vérifier dans Kafka UI que l'événement DevisGenere est présent
- Vérifier : curl http://localhost:8081/metrics (métriques Prometheus)
```

---

### Sous-étape 2.1.6 : Tests d'intégration Quotation

**Contexte :** Créer les tests d'intégration complets pour Quotation.

**Prérequis :** Sous-étape 2.1.5 complétée

**Critère de validation :** Tests d'intégration passent avec infrastructure réelle

```text
INSTRUCTION 2.1.6 - Service Quotation : Tests d'intégration

Objectif : Créer les tests d'intégration end-to-end pour le service Quotation

Contexte :
- Tests avec Kafka et SQLite réels
- Vérification du flux complet : API -> DB -> Kafka
- Utilisation de testcontainers optionnelle

Actions à réaliser :
1. Créer tests/integration/quotation_test.go :
   - Setup : connexion à l'infrastructure (docker-compose up requis)
   - Test flux complet génération de devis :
     * POST /api/devis
     * Vérifier réponse HTTP
     * Vérifier présence en DB
     * Consommer l'événement Kafka et vérifier le contenu
   - Test expiration de devis :
     * Créer un devis avec date d'expiration passée
     * Déclencher la vérification
     * Vérifier l'événement DevisExpire

2. Créer un consumer Kafka de test dans tests/integration/helpers.go :
   - Fonction ConsumeOne(topic string, timeout time.Duration) ([]byte, error)
   - Utilisé pour vérifier les événements produits

3. Mettre à jour le Makefile :
   - make test-integration-quotation

Test de validation :
- Exécuter : make up
- Exécuter : make test-integration-quotation
- Tous les tests doivent passer
- Les événements doivent être visibles dans Kafka UI
```

---

## Étape 2.2 : Service Souscription

### Sous-étape 2.2.1 : Structure et modèles Souscription

**Contexte :** Créer la structure du service Souscription.

**Prérequis :** Étape 2.1 complétée

**Critère de validation :** Le code compile sans erreur

```text
INSTRUCTION 2.2.1 - Service Souscription : Structure et modèles

Objectif : Créer la structure du service Souscription avec les modèles de données

Contexte :
- Service de gestion des contrats (Souscription)
- Consomme : DevisGenere (pour conversion)
- Produit : ContratEmis, ContratModifie, ContratResilie
- Consomme aussi : SinistreDeclare, IndemnisationEffectuee (pour historique)

Actions à réaliser :
1. Créer internal/models/souscription.go :
   - struct Contrat :
     * ID string (UUID)
     * DevisID string (référence au devis d'origine)
     * ClientID string
     * TypeBien string
     * Prime float64
     * DateEffet time.Time
     * DateFin time.Time
     * Statut string (ACTIF, SUSPENDU, RESILIE)
     * NombreSinistres int
     * MontantIndemnise float64

2. Créer internal/models/events_souscription.go :
   - struct ContratEmis :
     * ContratID, DevisID, ClientID, TypeBien, Prime, DateEffet, Timestamp
   - struct ContratModifie :
     * ContratID, Modification string, NouvelleValeur interface{}, Timestamp
   - struct ContratResilie :
     * ContratID, Motif string, DateResiliation, Timestamp

3. Créer les schémas Avro :
   - schemas/ContratEmis.avsc
   - schemas/ContratModifie.avsc
   - schemas/ContratResilie.avsc

4. Créer cmd/souscription/main.go :
   - Structure similaire à Quotation
   - Port 8082

Test de validation :
- Exécuter : go build ./cmd/souscription
- Vérifier que les schémas Avro sont valides
```

---

### Sous-étape 2.2.2 : Consumer Kafka

**Contexte :** Implémenter le consommateur Kafka pour Souscription.

**Prérequis :** Sous-étape 2.2.1 complétée

**Critère de validation :** Le service consomme les événements DevisGenere

```text
INSTRUCTION 2.2.2 - Service Souscription : Consumer Kafka

Objectif : Implémenter le consommateur Kafka pour traiter les événements entrants

Contexte :
- Consumer group : souscription-service
- Topics à consommer :
  * quotation.devis-genere (pour proposer la conversion)
  * reclamation.sinistre-declare (pour historique)
  * reclamation.indemnisation-effectuee (pour mise à jour risque)

Actions à réaliser :
1. Créer internal/kafka/consumer.go :
   - struct KafkaConsumer avec client et handlers
   - Fonction NewKafkaConsumer(bootstrapServers, groupID string, topics []string)
   - Méthode Start(ctx context.Context) : boucle de consommation
   - Méthode RegisterHandler(topic string, handler func(message []byte) error)
   - Méthode Stop()
   - Gestion des erreurs et retry

2. Créer internal/kafka/avro_deserialize.go :
   - Fonction DeserializeAvro(schemaRegistry string, data []byte, target interface{}) error
   - Extraction du schema ID depuis le message

3. Créer internal/services/souscription_handlers.go :
   - Handler pour DevisGenere :
     * Log de réception
     * Stockage pour traitement ultérieur (le devis peut être converti manuellement)
   - Handler pour SinistreDeclare :
     * Mise à jour du compteur de sinistres sur le contrat
   - Handler pour IndemnisationEffectuee :
     * Mise à jour du montant total indemnisé

4. Créer tests/integration/kafka_consumer_test.go :
   - Test de consommation d'un événement DevisGenere
   - Vérifier que le handler est appelé

Test de validation :
- Démarrer l'infrastructure : make up
- Produire un événement DevisGenere (via service Quotation)
- Vérifier dans les logs que Souscription l'a consommé
```

---

### Sous-étape 2.2.3 : Logique métier et API Souscription

**Contexte :** Implémenter la logique métier complète de Souscription.

**Prérequis :** Sous-étape 2.2.2 complétée

**Critère de validation :** Flux complet Devis -> Contrat fonctionne

```text
INSTRUCTION 2.2.3 - Service Souscription : Logique métier et API

Objectif : Implémenter la logique métier et l'API HTTP du service Souscription

Contexte :
- Conversion de devis en contrat
- Gestion du cycle de vie des contrats
- API HTTP sur port 8082

Actions à réaliser :
1. Créer internal/database/souscription_repository.go :
   - Interface ContratRepository (CRUD)
   - Implémentation SQLite

2. Créer internal/services/souscription_service.go :
   - Méthode ConvertirDevis(devisID string) (*models.Contrat, error) :
     * Récupère les infos du devis (via événement stocké ou API)
     * Crée le contrat
     * Publie ContratEmis
   - Méthode ModifierContrat(contratID string, modification string, valeur interface{}) error :
     * Met à jour le contrat
     * Publie ContratModifie
   - Méthode ResilierContrat(contratID string, motif string) error :
     * Change le statut
     * Publie ContratResilie

3. Créer cmd/souscription/handlers.go :
   - POST /api/contrats/convertir : { "devisId": "..." }
   - GET /api/contrats/{id}
   - PUT /api/contrats/{id}/modifier
   - POST /api/contrats/{id}/resilier
   - GET /health
   - GET /metrics

4. Créer internal/observability/souscription_metrics.go :
   - contrats_emis_total
   - contrats_resilies_total
   - sinistres_par_contrat (gauge)

5. Créer docker/souscription/Dockerfile

6. Ajouter service souscription au docker-compose.yml (port 8082)

Test de validation :
- make up
- Créer un devis via Quotation
- Convertir le devis en contrat via Souscription
- Vérifier ContratEmis dans Kafka UI
- Vérifier les métriques
```

---

## Étape 2.3 : Service Réclamation

### Sous-étape 2.3.1 : Structure complète Réclamation

**Contexte :** Créer le service Réclamation complet (structure similaire).

**Prérequis :** Étape 2.2 complétée

**Critère de validation :** Service fonctionnel de bout en bout

```text
INSTRUCTION 2.3.1 - Service Réclamation : Implémentation complète

Objectif : Implémenter le service Réclamation complet

Contexte :
- Gestion des sinistres et indemnisations
- Consomme : ContratEmis, ContratResilie (pour vérifier couverture)
- Produit : SinistreDeclare, SinistreEvalue, IndemnisationEffectuee
- Port 8083

Actions à réaliser :
1. Créer internal/models/reclamation.go :
   - struct Sinistre :
     * ID, ContratID, Type, Description, DateSurvenance
     * MontantEstime, MontantEvalue, MontantIndemnise
     * Statut (DECLARE, EN_EXPERTISE, EVALUE, INDEMNISE, REJETE)
   - struct Indemnisation :
     * ID, SinistreID, Montant, DatePaiement

2. Créer les schémas Avro :
   - schemas/SinistreDeclare.avsc
   - schemas/SinistreEvalue.avsc
   - schemas/IndemnisationEffectuee.avsc

3. Créer internal/database/reclamation_repository.go

4. Créer internal/services/reclamation_service.go :
   - DeclarerSinistre(contratID, type, description string, montantEstime float64) :
     * Vérifie que le contrat est actif (via cache local des ContratEmis reçus)
     * Crée le sinistre
     * Publie SinistreDeclare
   - EvaluerSinistre(sinistreID string, montantEvalue float64) :
     * Met à jour le montant
     * Publie SinistreEvalue
   - Indemniser(sinistreID string) :
     * Crée l'indemnisation
     * Publie IndemnisationEffectuee

5. Créer cmd/reclamation/main.go et handlers.go :
   - POST /api/sinistres : déclarer
   - PUT /api/sinistres/{id}/evaluer
   - POST /api/sinistres/{id}/indemniser
   - GET /health, /metrics

6. Créer internal/services/reclamation_handlers.go :
   - Handler ContratEmis : ajoute le contrat au cache local
   - Handler ContratResilie : marque le contrat comme inactif

7. Dockerfile et docker-compose (port 8083)

Test de validation :
- make up
- Créer un devis, le convertir en contrat
- Déclarer un sinistre sur ce contrat
- Évaluer et indemniser
- Vérifier les 3 événements dans Kafka UI
- Vérifier que Souscription a reçu les événements (logs + compteurs)
```

---

## Étape 2.4 : Dashboard de Contrôle

### Sous-étape 2.4.1 : Structure du Dashboard

**Contexte :** Créer la structure du dashboard web avec HTMX.

**Prérequis :** Étape 2.3 complétée

**Critère de validation :** Page d'accueil s'affiche

```text
INSTRUCTION 2.4.1 - Dashboard : Structure de base

Objectif : Créer la structure du dashboard web avec Go Templates et HTMX

Contexte :
- Interface de contrôle de la simulation
- Pas de framework JS, uniquement HTMX
- Port 8080

Actions à réaliser :
1. Créer web/templates/layout.html :
   - Structure HTML5 de base
   - Inclusion HTMX via CDN
   - CSS de base (ou Tailwind via CDN)
   - Header avec titre et navigation
   - Zone de contenu principale
   - Footer avec liens Grafana/Jaeger

2. Créer web/templates/index.html :
   - Section "Contrôle de la simulation"
     * Boutons Start/Stop
     * Sélecteur de vitesse
     * Sélecteur de scénario
   - Section "Statistiques" (placeholder)
   - Section "Événements récents" (placeholder)

3. Créer web/static/style.css :
   - Styles de base pour le dashboard
   - Couleurs et mise en page

4. Créer web/handlers/dashboard.go :
   - Handler GET / : rendu de index.html
   - Handler GET /health
   - Handler GET /metrics

5. Mettre à jour cmd/dashboard/main.go :
   - Configuration serveur HTTP
   - Servir les fichiers statiques
   - Graceful shutdown

6. Dockerfile et docker-compose (port 8080)

Test de validation :
- make up
- Ouvrir http://localhost:8080
- La page d'accueil doit s'afficher avec le layout
- Les boutons sont présents (pas fonctionnels encore)
```

---

### Sous-étape 2.4.2 : Visualisation temps réel avec SSE

**Contexte :** Implémenter le flux d'événements en temps réel.

**Prérequis :** Sous-étape 2.4.1 complétée

**Critère de validation :** Les événements s'affichent en temps réel

```text
INSTRUCTION 2.4.2 - Dashboard : Événements temps réel

Objectif : Afficher les événements Kafka en temps réel via Server-Sent Events

Contexte :
- Le dashboard consomme tous les topics Kafka
- Affiche les événements dans une timeline
- Mise à jour via SSE (Server-Sent Events) + HTMX

Actions à réaliser :
1. Créer internal/services/event_aggregator.go :
   - Consumer Kafka qui écoute tous les topics
   - Channel pour distribuer les événements aux clients SSE
   - struct EventDisplay :
     * Timestamp, Type, Source, Destination, Payload (résumé), Status

2. Créer web/handlers/events.go :
   - Handler GET /api/events/stream (SSE) :
     * Content-Type: text/event-stream
     * Envoi des événements au format JSON
   - Handler GET /api/events/recent :
     * Retourne les 50 derniers événements

3. Mettre à jour web/templates/index.html :
   - Section "Événements récents" avec :
     * Liste ul/li mise à jour via hx-ext="sse"
     * hx-sse="connect:/api/events/stream"
     * Chaque événement affiché avec : heure, type, source->dest, statut
   - Indicateur de connexion SSE

4. Créer web/templates/partials/event_row.html :
   - Template pour une ligne d'événement
   - Utilisé par SSE pour l'insertion

5. Ajouter du CSS pour :
   - Animation d'apparition des nouveaux événements
   - Code couleur par type d'événement

Test de validation :
- make up
- Ouvrir http://localhost:8080
- Créer un devis via curl http://localhost:8081/api/devis
- L'événement doit apparaître dans la timeline du dashboard
- Tester plusieurs événements consécutifs
```

---

### Sous-étape 2.4.3 : Visualisation du flux inter-systèmes

**Contexte :** Créer le diagramme animé des flux entre systèmes.

**Prérequis :** Sous-étape 2.4.2 complétée

**Critère de validation :** Animation visible lors des événements

```text
INSTRUCTION 2.4.3 - Dashboard : Diagramme de flux animé

Objectif : Créer une visualisation animée des événements entre les 3 systèmes

Contexte :
- Représentation visuelle de Quotation -> Souscription -> Réclamation
- Animation lors du passage d'un événement
- SVG ou Canvas simple

Actions à réaliser :
1. Créer web/templates/partials/flow_diagram.html :
   - SVG avec les 3 boîtes : Quotation, Souscription, Réclamation
   - Flèches entre les systèmes
   - Zones pour afficher le dernier événement sur chaque flèche

2. Créer web/static/flow.js (JavaScript minimal) :
   - Fonction animateEvent(from, to, eventType)
   - Animation CSS de la flèche (pulse, couleur)
   - Affichage temporaire du nom de l'événement

3. Mettre à jour web/handlers/events.go :
   - Inclure source et destination dans les événements SSE
   - Format : { type: "DevisGenere", from: "quotation", to: "souscription" }

4. Mettre à jour web/templates/index.html :
   - Intégrer le diagramme de flux
   - Connecter les événements SSE à l'animation

5. Ajouter du CSS pour les animations :
   - @keyframes pour le pulse des flèches
   - Transitions de couleur

Test de validation :
- make up
- Ouvrir http://localhost:8080
- Créer un devis, le convertir en contrat, déclarer un sinistre
- Observer les animations sur le diagramme
- Chaque flux doit s'animer au passage de l'événement
```

---

## Étape 2.5 : Simulateur d'Événements

### Sous-étape 2.5.1 : Générateur d'événements automatique

**Contexte :** Créer le simulateur qui génère des événements automatiquement.

**Prérequis :** Étape 2.4 complétée

**Critère de validation :** Événements générés au rythme configuré

```text
INSTRUCTION 2.5.1 - Simulateur : Générateur d'événements

Objectif : Créer le simulateur qui génère des événements automatiquement

Contexte :
- Génère des scénarios métier complets (devis -> contrat -> sinistre)
- Configurable en vitesse et scénario
- Appelé depuis le dashboard

Actions à réaliser :
1. Créer cmd/simulator/main.go :
   - Service HTTP sur port 8084
   - Endpoints de contrôle

2. Créer internal/services/simulator.go :
   - struct Simulator avec configuration :
     * Speed (events per second)
     * Scenario (NORMAL, PIC_CHARGE, ERREURS, etc.)
     * Running (bool)
   - Méthode Start() : démarre la génération
   - Méthode Stop() : arrête
   - Méthode SetSpeed(speed int)
   - Méthode SetScenario(scenario string)

3. Créer internal/services/scenarios.go :
   - Scénario NORMAL :
     * Génère des devis aléatoires
     * 60% sont convertis en contrats
     * 20% des contrats ont un sinistre
   - Scénario PIC_CHARGE :
     * Multiplie la vitesse par 10 pendant 30 secondes
   - Scénario ERREURS :
     * Introduit des données invalides (10% des événements)
   - Scénario CONSOMMATEUR_LENT :
     * Injecte des délais dans les handlers

4. Créer internal/services/data_generator.go :
   - Génération de données réalistes :
     * Noms de clients (liste prédéfinie)
     * Types de biens (AUTO, HABITATION)
     * Valeurs réalistes (10k-500k)
     * Types de sinistres (VOL, ACCIDENT, DEGAT_DES_EAUX, etc.)

5. Créer cmd/simulator/handlers.go :
   - POST /api/simulator/start
   - POST /api/simulator/stop
   - PUT /api/simulator/speed : { "eventsPerSecond": 5 }
   - PUT /api/simulator/scenario : { "scenario": "NORMAL" }
   - GET /api/simulator/status

6. Dockerfile et docker-compose (port 8084)

Test de validation :
- make up
- curl -X POST http://localhost:8084/api/simulator/start
- Observer les événements dans le dashboard
- curl -X PUT http://localhost:8084/api/simulator/speed -d '{"eventsPerSecond":10}'
- Observer l'accélération
- curl -X POST http://localhost:8084/api/simulator/stop
```

---

### Sous-étape 2.5.2 : Intégration Dashboard-Simulateur

**Contexte :** Connecter les contrôles du dashboard au simulateur.

**Prérequis :** Sous-étape 2.5.1 complétée

**Critère de validation :** Contrôle complet depuis le dashboard

```text
INSTRUCTION 2.5.2 - Dashboard : Contrôle du simulateur

Objectif : Connecter les boutons du dashboard au simulateur

Contexte :
- Les boutons Start/Stop appellent l'API du simulateur
- Les sélecteurs de vitesse et scénario sont fonctionnels
- Affichage du statut en temps réel

Actions à réaliser :
1. Mettre à jour web/handlers/dashboard.go :
   - Proxy vers le simulateur :
     * POST /api/simulation/start -> http://simulator:8084/api/simulator/start
     * POST /api/simulation/stop -> ...
     * etc.
   - Ou appel direct HTMX au simulateur (CORS)

2. Mettre à jour web/templates/index.html :
   - Bouton Start :
     * hx-post="/api/simulation/start"
     * hx-swap="none"
     * Change de style quand actif
   - Bouton Stop :
     * hx-post="/api/simulation/stop"
   - Sélecteur de vitesse :
     * <select> avec options Lente/Normale/Rapide
     * hx-put="/api/simulation/speed"
     * hx-vals pour envoyer la valeur
   - Sélecteur de scénario :
     * <select> avec les 5 scénarios
     * hx-put="/api/simulation/scenario"
   - Indicateur de statut :
     * hx-get="/api/simulation/status"
     * hx-trigger="every 2s"

3. Créer web/templates/partials/simulation_status.html :
   - Affiche : Running/Stopped, vitesse actuelle, scénario actuel
   - Compteur d'événements générés

4. Ajouter du CSS :
   - Style des boutons actifs/inactifs
   - Animation du bouton Start quand running

Test de validation :
- make up
- Ouvrir http://localhost:8080
- Cliquer sur Start : les événements commencent à défiler
- Changer la vitesse : le rythme change
- Changer le scénario : le comportement change
- Cliquer sur Stop : les événements s'arrêtent
```

---

### Sous-étape 2.5.3 : Dashboards Grafana complets

**Contexte :** Compléter les dashboards Grafana avec les métriques réelles.

**Prérequis :** Sous-étape 2.5.2 complétée

**Critère de validation :** Dashboards fonctionnels avec données réelles

```text
INSTRUCTION 2.5.3 - Grafana : Dashboards complets

Objectif : Créer les 4 dashboards Grafana fonctionnels

Contexte :
- Les services exposent maintenant des métriques
- Les dashboards doivent afficher les données réelles

Actions à réaliser :
1. Mettre à jour docker/grafana/provisioning/dashboards/kafka-overview.json :
   - Panel "Topics" : liste des topics avec messages count
   - Panel "Messages/sec" : rate(kafka_messages_total[1m])
   - Panel "Consumer Lag" : kafka_consumer_lag (si disponible)

2. Créer docker/grafana/provisioning/dashboards/services-health.json :
   - Row par service (Quotation, Souscription, Réclamation)
   - Panels par service :
     * Stat : Status (up/down)
     * Graph : Requêtes/sec
     * Graph : Latence p50/p95/p99
     * Stat : Taux d'erreur

3. Créer docker/grafana/provisioning/dashboards/business-events.json :
   - Panel : Compteur par type d'événement (pie chart)
   - Panel : Évolution temporelle des événements (stacked graph)
   - Panel : Table des derniers événements
   - Variables pour filtrer par type

4. Créer docker/grafana/provisioning/dashboards/simulation-control.json :
   - Panel : Statut simulateur (running/stopped)
   - Panel : Vitesse actuelle
   - Panel : Scénario actif
   - Panel : Total événements générés
   - Panel : Graphe des événements/sec en temps réel

Test de validation :
- make up
- Démarrer la simulation depuis le dashboard
- Ouvrir Grafana http://localhost:3000
- Vérifier chaque dashboard
- Les panels doivent afficher des données réelles
- Laisser tourner 5 minutes et vérifier les tendances
```

---

### Sous-étape 2.5.4 : Tests d'intégration Phase 2

**Contexte :** Tests d'intégration complets de la Phase 2.

**Prérequis :** Sous-étape 2.5.3 complétée

**Critère de validation :** Tous les tests passent

```text
INSTRUCTION 2.5.4 - Tests d'intégration Phase 2

Objectif : Créer la suite de tests d'intégration pour le patron Producteur/Consommateur

Actions à réaliser :
1. Créer tests/integration/phase2_test.go :
   - Test flux complet sans simulateur :
     * Créer un devis
     * Vérifier événement DevisGenere
     * Convertir en contrat
     * Vérifier événement ContratEmis
     * Vérifier que Réclamation a reçu (via log ou métrique)
     * Déclarer un sinistre
     * Vérifier événement SinistreDeclare
     * Vérifier que Souscription a mis à jour le compteur

2. Créer tests/integration/simulator_test.go :
   - Test démarrage/arrêt du simulateur
   - Test changement de vitesse
   - Test scénario NORMAL (vérifier distribution des événements)

3. Créer tests/integration/dashboard_test.go :
   - Test page d'accueil accessible
   - Test endpoint SSE (connexion et réception)
   - Test contrôles du simulateur

4. Mettre à jour Makefile :
   - make test-integration : tous les tests
   - make test-integration-phase2 : seulement phase 2

5. Créer un script de test end-to-end : scripts/e2e-phase2.sh :
   - Démarre l'infra
   - Attend que tout soit healthy
   - Exécute les tests
   - Vérifie les métriques dans Prometheus
   - Affiche un rapport

Test de validation :
- make up
- make test-integration-phase2
- Tous les tests passent
- Rapport de couverture généré
```

---

### Sous-étape 2.5.5 : Documentation et Tag Phase 2

**Contexte :** Finaliser la Phase 2 avec documentation et tag.

**Prérequis :** Sous-étape 2.5.4 complétée

**Critère de validation :** Tag v2.0-pubsub créé

```text
INSTRUCTION 2.5.5 - Finalisation Phase 2

Objectif : Documenter et tagger la Phase 2

Actions à réaliser :
1. Compléter docs/01-producteur-consommateur/README.md :
   - Explication complète du patron
   - Schéma de l'architecture implémentée
   - Description des 3 services
   - Liste des événements et leur flux
   - Guide d'utilisation du dashboard
   - Exercices de compréhension :
     * Que se passe-t-il si on arrête Souscription ?
     * Observer le consumer lag dans Grafana
     * Modifier la vitesse et observer l'impact

2. Mettre à jour README.md :
   - Ajouter la section "Utilisation"
   - Screenshots du dashboard et Grafana

3. Mettre à jour CHANGELOG.md :
   - ## v2.0-pubsub
   - Liste des fonctionnalités :
     * 3 services métier (Quotation, Souscription, Réclamation)
     * 8 types d'événements
     * Dashboard de contrôle et visualisation
     * Simulateur avec 5 scénarios
     * 4 dashboards Grafana

4. Vérification finale :
   - make reset
   - make up
   - make health
   - Démarrer simulation, laisser tourner 2 minutes
   - Vérifier tous les dashboards

5. Créer le tag :
   - git add .
   - git commit -m "Phase 2: Patron Producteur/Consommateur complet"
   - git tag -a v2.0-pubsub -m "Phase 2: Producteur/Consommateur"

Test de validation :
- git checkout v2.0-pubsub
- make up && make health
- La simulation fonctionne complètement
```

---

# PHASE 3 : Event Sourcing

## Étape 3.1 : Refactoring pour Event Store

### Sous-étape 3.1.1 : Event Store abstraction

```text
INSTRUCTION 3.1.1 - Event Sourcing : Abstraction Event Store

Objectif : Créer l'abstraction Event Store pour stocker et rejouer les événements

Contexte :
- Kafka comme Event Store
- Les événements sont la source de vérité
- L'état est dérivé des événements

Actions à réaliser :
1. Créer internal/eventsourcing/event_store.go :
   - Interface EventStore :
     * Append(aggregateID string, events []Event) error
     * Load(aggregateID string) ([]Event, error)
     * LoadFromVersion(aggregateID, fromVersion int) ([]Event, error)
   - struct Event :
     * ID, AggregateID, Type, Version, Timestamp, Data

2. Créer internal/eventsourcing/kafka_event_store.go :
   - Implémentation avec Kafka
   - Un topic par type d'agrégat
   - Clé = aggregateID pour ordering

3. Créer internal/eventsourcing/aggregate.go :
   - Interface Aggregate :
     * ID() string
     * Version() int
     * ApplyEvent(event Event) error
     * UncommittedEvents() []Event
   - BaseAggregate avec implémentation commune

4. Créer tests pour l'Event Store

Test de validation :
- Tests unitaires passent
- Test d'intégration avec Kafka réel
```

---

### Sous-étape 3.1.2 : Agrégat Contrat avec Event Sourcing

```text
INSTRUCTION 3.1.2 - Event Sourcing : Agrégat Contrat

Objectif : Transformer le service Souscription pour utiliser Event Sourcing

Contexte :
- L'agrégat Contrat maintient son état via les événements
- Plus de mise à jour directe en base
- SQLite devient une projection (vue matérialisée)

Actions à réaliser :
1. Créer internal/aggregates/contrat.go :
   - struct ContratAggregate implémente Aggregate
   - État interne : ID, ClientID, Statut, Prime, NombreSinistres, etc.
   - Méthodes de commande :
     * Emettre(devisID, clientID, typeBien, prime) : génère ContratEmis
     * Modifier(modification, valeur) : génère ContratModifie
     * Resilier(motif) : génère ContratResilie
     * EnregistrerSinistre() : incrémente compteur
   - Méthode ApplyEvent : met à jour l'état selon le type d'événement

2. Mettre à jour internal/services/souscription_service.go :
   - Charger l'agrégat depuis l'Event Store
   - Appliquer les commandes
   - Sauvegarder les nouveaux événements
   - La projection SQLite est mise à jour en async

3. Créer internal/projections/contrat_projection.go :
   - Consumer qui écoute les événements Contrat
   - Met à jour SQLite pour les requêtes de lecture

4. Tests unitaires et d'intégration

Test de validation :
- Créer un contrat, le modifier, le résilier
- Redémarrer le service
- L'état doit être reconstruit depuis Kafka
- La projection SQLite doit être synchronisée
```

---

## Étape 3.2 : Reconstruction d'état

### Sous-étape 3.2.1 : Rebuild complet depuis Kafka

```text
INSTRUCTION 3.2.1 - Event Sourcing : Reconstruction d'état

Objectif : Implémenter la reconstruction d'état au démarrage

Contexte :
- Au démarrage, le service relit tous les événements
- L'état est reconstruit avant d'accepter des requêtes
- Démonstration pédagogique claire

Actions à réaliser :
1. Créer internal/eventsourcing/rebuilder.go :
   - Fonction RebuildProjection(eventStore, projection, fromOffset) :
     * Lit tous les événements depuis l'offset
     * Applique chaque événement à la projection
     * Affiche la progression
   - Support du rebuild partiel (depuis un offset)

2. Mettre à jour cmd/souscription/main.go :
   - Au démarrage, appeler RebuildProjection si flag --rebuild
   - Log détaillé du processus de reconstruction
   - Métrique : rebuild_events_processed_total

3. Ajouter un endpoint de diagnostic :
   - GET /api/debug/state : affiche l'état actuel de la projection
   - GET /api/debug/events?aggregateId=X : liste les événements d'un agrégat

4. Mettre à jour le dashboard :
   - Indicateur de "reconstruction en cours"
   - Bouton pour déclencher un rebuild manuel

5. Documentation :
   - Expliquer le concept de reconstruction
   - Exercice : supprimer la DB SQLite et observer le rebuild

Test de validation :
- Créer plusieurs contrats
- Arrêter le service
- Supprimer souscription.db
- Redémarrer avec --rebuild
- Vérifier que l'état est identique
```

---

## Étape 3.3 : Snapshots

### Sous-étape 3.3.1 : Mécanisme de snapshots

```text
INSTRUCTION 3.3.1 - Event Sourcing : Snapshots

Objectif : Implémenter les snapshots pour optimiser la reconstruction

Contexte :
- Un snapshot capture l'état à un instant T
- La reconstruction part du dernier snapshot
- Réduit le temps de démarrage

Actions à réaliser :
1. Créer internal/eventsourcing/snapshot_store.go :
   - Interface SnapshotStore :
     * Save(aggregateID string, version int, state []byte) error
     * Load(aggregateID string) (*Snapshot, error)
   - struct Snapshot : AggregateID, Version, Timestamp, State

2. Créer internal/eventsourcing/sqlite_snapshot_store.go :
   - Implémentation avec SQLite
   - Table snapshots : aggregate_id, version, timestamp, state_json

3. Mettre à jour l'agrégat Contrat :
   - Méthode ToSnapshot() ([]byte, error)
   - Méthode FromSnapshot(data []byte) error
   - Créer un snapshot tous les N événements (configurable, défaut 100)

4. Mettre à jour le rebuilder :
   - Charger le dernier snapshot
   - Rejouer uniquement les événements depuis le snapshot

5. Métriques et observabilité :
   - snapshots_created_total
   - rebuild_from_snapshot_events (combien d'événements depuis le snapshot)

6. Documentation :
   - Expliquer les snapshots et leur utilité
   - Trade-offs : fréquence vs taille

Test de validation :
- Créer 200 événements sur un contrat
- Vérifier qu'un snapshot est créé
- Redémarrer et mesurer le temps de reconstruction
- Comparer avec/sans snapshot
```

---

### Sous-étape 3.3.2 : Tests et Tag Phase 3

```text
INSTRUCTION 3.3.2 - Finalisation Phase 3

Objectif : Tests complets et tag pour Event Sourcing

Actions à réaliser :
1. Créer tests/integration/phase3_test.go :
   - Test reconstruction complète
   - Test reconstruction depuis snapshot
   - Test cohérence après redémarrage
   - Test performance (mesurer le temps)

2. Compléter docs/02-event-sourcing/README.md :
   - Théorie de l'Event Sourcing
   - Avantages (audit, time travel, debugging)
   - Inconvénients (complexité, stockage)
   - Schéma de l'implémentation
   - Exercices :
     * Observer la reconstruction
     * Time travel : reconstruire l'état à une date passée
     * Comparer performance avec/sans snapshot

3. Tag v3.0-eventsourcing

Test de validation :
- Tous les tests Phase 3 passent
- Documentation complète
- Tag créé et fonctionnel
```

---

# PHASES 4-7 : Structure Résumée

Les phases suivantes suivent le même pattern de décomposition. Voici la structure résumée :

## Phase 4 : CQRS

```text
INSTRUCTION 4.1.1 - Séparation des modèles Command/Query
INSTRUCTION 4.1.2 - Command handlers dédiés
INSTRUCTION 4.2.1 - Vues matérialisées optimisées
INSTRUCTION 4.2.2 - Projections spécialisées (liste, détail, statistiques)
INSTRUCTION 4.3.1 - Synchronisation et eventual consistency
INSTRUCTION 4.3.2 - Tests et Tag v4.0-cqrs
```

## Phase 5 : Saga Choreography

```text
INSTRUCTION 5.1.1 - Définition du processus de souscription complète
INSTRUCTION 5.1.2 - Coordination par événements
INSTRUCTION 5.2.1 - Événements de compensation
INSTRUCTION 5.2.2 - Rollback automatique sur échec
INSTRUCTION 5.3.1 - Scénarios de test (succès, échec partiel, timeout)
INSTRUCTION 5.3.2 - Tests et Tag v5.0-saga
```

## Phase 6 : Dead Letter Queue

```text
INSTRUCTION 6.1.1 - Gestion des erreurs de traitement
INSTRUCTION 6.1.2 - Retry avec backoff exponentiel
INSTRUCTION 6.2.1 - Configuration DLQ par topic
INSTRUCTION 6.2.2 - Interface de visualisation des erreurs
INSTRUCTION 6.3.1 - Scénarios : Service en panne, message invalide
INSTRUCTION 6.3.2 - Tests et Tag v6.0-dlq
```

## Phase 7 : Finalisation

```text
INSTRUCTION 7.1.1 - Documentation complète de tous les patrons
INSTRUCTION 7.1.2 - Guide d'utilisation final
INSTRUCTION 7.2.1 - Tests de charge avec k6
INSTRUCTION 7.2.2 - Rapport de performance
INSTRUCTION 7.3.1 - Polish UI et UX
INSTRUCTION 7.3.2 - Tag final v7.0-final
```

---

# Index des Instructions

| # | Instruction | Phase | Prérequis |
|---|-------------|-------|-----------|
| 1.1.1 | Initialisation module Go | 1 | Aucun |
| 1.1.2 | Makefile complet | 1 | 1.1.1 |
| 1.1.3 | Configuration Git | 1 | 1.1.2 |
| 1.2.1 | Docker Kafka KRaft | 1 | 1.1.3 |
| 1.2.2 | Schema Registry | 1 | 1.2.1 |
| 1.2.3 | Topics Kafka | 1 | 1.2.2 |
| 1.2.4 | Kafka UI | 1 | 1.2.3 |
| 1.3.1 | Prometheus | 1 | 1.2.4 |
| 1.3.2 | Grafana | 1 | 1.3.1 |
| 1.3.3 | Loki | 1 | 1.3.2 |
| 1.3.4 | Jaeger | 1 | 1.3.3 |
| 1.3.5 | Dashboard Kafka | 1 | 1.3.4 |
| 1.4.1 | Health check | 1 | 1.3.5 |
| 1.4.2 | Documentation P1 | 1 | 1.4.1 |
| 1.4.3 | Tag v1.0-infra | 1 | 1.4.2 |
| 2.1.1 | Quotation structure | 2 | 1.4.3 |
| 2.1.2 | Quotation DB | 2 | 2.1.1 |
| 2.1.3 | Quotation Kafka | 2 | 2.1.2 |
| 2.1.4 | Quotation métier | 2 | 2.1.3 |
| 2.1.5 | Quotation API | 2 | 2.1.4 |
| 2.1.6 | Quotation tests | 2 | 2.1.5 |
| 2.2.1 | Souscription structure | 2 | 2.1.6 |
| 2.2.2 | Souscription consumer | 2 | 2.2.1 |
| 2.2.3 | Souscription métier | 2 | 2.2.2 |
| 2.3.1 | Réclamation complet | 2 | 2.2.3 |
| 2.4.1 | Dashboard structure | 2 | 2.3.1 |
| 2.4.2 | Dashboard SSE | 2 | 2.4.1 |
| 2.4.3 | Dashboard flux | 2 | 2.4.2 |
| 2.5.1 | Simulateur | 2 | 2.4.3 |
| 2.5.2 | Dashboard-Simulateur | 2 | 2.5.1 |
| 2.5.3 | Grafana dashboards | 2 | 2.5.2 |
| 2.5.4 | Tests Phase 2 | 2 | 2.5.3 |
| 2.5.5 | Tag v2.0-pubsub | 2 | 2.5.4 |
| ... | ... | ... | ... |

---

**Total : 52 instructions détaillées**

**Temps estimé par instruction : Variable selon complexité**

**Approche : Chaque instruction est autonome et testable**
