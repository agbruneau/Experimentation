# TODO - kafka-eda-lab

## Progression Globale

| Phase | Progression | Statut |
|-------|-------------|--------|
| Phase 1 - Infrastructure | 0/15 | Non commencé |
| Phase 2 - Pub/Sub | 0/20 | Non commencé |
| Phase 3 - Event Sourcing | 0/5 | Non commencé |
| Phase 4 - CQRS | 0/6 | Non commencé |
| Phase 5 - Saga | 0/6 | Non commencé |
| Phase 6 - DLQ | 0/6 | Non commencé |
| Phase 7 - Finalisation | 0/6 | Non commencé |
| **Total** | **0/64** | |

---

# PHASE 1 : Infrastructure de Base

## Étape 1.1 : Structure du Projet

- [ ] **1.1.1** Initialisation du module Go
  - [ ] Créer go.mod avec le module github.com/[user]/kafka-eda-lab
  - [ ] Créer l'arborescence cmd/ (quotation, souscription, reclamation, dashboard, simulator)
  - [ ] Créer l'arborescence internal/ (kafka, models, observability, database)
  - [ ] Créer les dossiers pkg/, schemas/, web/, docker/, docs/, tests/
  - [ ] Créer cmd/dashboard/main.go minimal
  - [ ] Créer Makefile de base (build, clean)
  - [ ] Valider : `go build ./cmd/dashboard` et `go mod tidy`

- [ ] **1.1.2** Configuration du Makefile complet
  - [ ] Commande `make up` (docker-compose up -d)
  - [ ] Commande `make down` (docker-compose down)
  - [ ] Commande `make reset` (down -v && up)
  - [ ] Commande `make logs` (docker-compose logs -f)
  - [ ] Commande `make status` (docker-compose ps)
  - [ ] Commande `make build` (go build tous les services)
  - [ ] Commande `make test` (go test ./...)
  - [ ] Commande `make test-integration`
  - [ ] Commande `make test-load`
  - [ ] Commande `make dashboard` (ouvre localhost:8080)
  - [ ] Commande `make grafana` (ouvre localhost:3000)
  - [ ] Commande `make jaeger` (ouvre localhost:16686)
  - [ ] Commande `make help` (liste des commandes)
  - [ ] Ajouter .PHONY pour toutes les commandes
  - [ ] Valider : `make help` affiche toutes les commandes

- [ ] **1.1.3** Configuration Git et environnement
  - [ ] Créer .gitignore (binaires, .db, .env, IDE, volumes)
  - [ ] Créer .env.example avec toutes les variables
  - [ ] Créer README.md initial (description, prérequis, démarrage rapide)
  - [ ] Valider : fichiers ignorés correctement

## Étape 1.2 : Configuration Docker Kafka

- [ ] **1.2.1** Docker Compose Kafka KRaft
  - [ ] Créer docker-compose.yml
  - [ ] Configurer service kafka (image apache/kafka:3.7.0)
  - [ ] Variables KRaft (NODE_ID, PROCESS_ROLES, CONTROLLER_QUORUM_VOTERS)
  - [ ] Port 9092 exposé
  - [ ] Volume kafka-data
  - [ ] Healthcheck configuré
  - [ ] Valider : `docker-compose up -d kafka` démarre sans erreur
  - [ ] Valider : `kafka-topics.sh --list` fonctionne

- [ ] **1.2.2** Ajout du Schema Registry
  - [ ] Ajouter service schema-registry (confluentinc/cp-schema-registry:7.6.0)
  - [ ] Port 8081 exposé
  - [ ] Dépendance sur kafka
  - [ ] Variables d'environnement configurées
  - [ ] Healthcheck configuré
  - [ ] Valider : `curl http://localhost:8081/subjects` retourne []

- [ ] **1.2.3** Création des topics Kafka
  - [ ] Créer docker/kafka/create-topics.sh
  - [ ] Topic quotation.devis-genere (3 partitions)
  - [ ] Topic quotation.devis-expire (3 partitions)
  - [ ] Topic souscription.contrat-emis (3 partitions)
  - [ ] Topic souscription.contrat-modifie (3 partitions)
  - [ ] Topic souscription.contrat-resilie (3 partitions)
  - [ ] Topic reclamation.sinistre-declare (3 partitions)
  - [ ] Topic reclamation.sinistre-evalue (3 partitions)
  - [ ] Topic reclamation.indemnisation-effectuee (3 partitions)
  - [ ] Topic dlq.errors (1 partition)
  - [ ] Ajouter service kafka-init dans docker-compose
  - [ ] Ajouter `make topics` dans Makefile
  - [ ] Valider : 9 topics créés

- [ ] **1.2.4** Interface Kafka UI
  - [ ] Ajouter service kafka-ui (provectuslabs/kafka-ui:latest)
  - [ ] Port 8090 exposé
  - [ ] Configuration cluster et schema registry
  - [ ] Ajouter `make kafka-ui` dans Makefile
  - [ ] Valider : http://localhost:8090 accessible

## Étape 1.3 : Stack d'Observabilité

- [ ] **1.3.1** Configuration Prometheus
  - [ ] Créer dossier docker/prometheus/
  - [ ] Créer prometheus.yml avec scrape configs
  - [ ] Ajouter service prometheus (prom/prometheus:v2.50.0)
  - [ ] Port 9090 exposé
  - [ ] Volume prometheus-data
  - [ ] Valider : http://localhost:9090 accessible

- [ ] **1.3.2** Configuration Grafana
  - [ ] Créer docker/grafana/provisioning/datasources/
  - [ ] Créer datasources.yml (Prometheus + Loki)
  - [ ] Créer grafana.ini (auth anonyme)
  - [ ] Ajouter service grafana (grafana/grafana:10.3.0)
  - [ ] Port 3000 exposé
  - [ ] Volume grafana-data
  - [ ] Valider : http://localhost:3000 accessible
  - [ ] Valider : Prometheus datasource fonctionnel

- [ ] **1.3.3** Configuration Loki
  - [ ] Créer docker/loki/loki-config.yml
  - [ ] Ajouter service loki (grafana/loki:2.9.4)
  - [ ] Port 3100 exposé
  - [ ] Volume loki-data
  - [ ] Valider : `curl http://localhost:3100/ready` retourne ready

- [ ] **1.3.4** Configuration Jaeger
  - [ ] Ajouter service jaeger (jaegertracing/all-in-one:1.54)
  - [ ] Ports 16686, 4317, 4318 exposés
  - [ ] OTLP enabled
  - [ ] Valider : http://localhost:16686 accessible

- [ ] **1.3.5** Dashboard Grafana Kafka Overview
  - [ ] Créer docker/grafana/provisioning/dashboards/dashboards.yml
  - [ ] Créer kafka-overview.json (placeholder)
  - [ ] Panel Brokers Status
  - [ ] Panel Messages/sec
  - [ ] Panel Topics
  - [ ] Panel Consumer Lag
  - [ ] Valider : Dashboard visible dans Grafana

## Étape 1.4 : Validation Infrastructure

- [ ] **1.4.1** Script de vérification de santé
  - [ ] Créer scripts/health-check.sh
  - [ ] Vérification Kafka
  - [ ] Vérification Schema Registry
  - [ ] Vérification Prometheus
  - [ ] Vérification Grafana
  - [ ] Vérification Loki
  - [ ] Vérification Jaeger
  - [ ] Rapport avec codes couleur
  - [ ] Créer scripts/health-check.ps1 (Windows)
  - [ ] Ajouter `make health` dans Makefile
  - [ ] Valider : tous les services OK

- [ ] **1.4.2** Documentation Phase 1
  - [ ] Créer docs/00-introduction.md
  - [ ] Créer docs/01-producteur-consommateur/README.md (intro)
  - [ ] Mettre à jour README.md (instructions complètes)
  - [ ] Liste des URLs d'accès
  - [ ] Troubleshooting basique
  - [ ] Valider : documentation lisible et complète

- [ ] **1.4.3** Tag Git v1.0-infra
  - [ ] Vérifier tous les fichiers commités
  - [ ] Exécuter `make up` et `make health`
  - [ ] Créer CHANGELOG.md
  - [ ] Créer tag v1.0-infra
  - [ ] Valider : `git checkout v1.0-infra` fonctionne

---

# PHASE 2 : Patron Producteur/Consommateur

## Étape 2.1 : Service Quotation

- [ ] **2.1.1** Structure et modèles Quotation
  - [ ] Créer internal/models/quotation.go (struct Devis)
  - [ ] Créer internal/models/events.go (DevisGenere, DevisExpire)
  - [ ] Créer cmd/quotation/main.go minimal
  - [ ] Ajouter dépendances go.mod (uuid, sqlite3)
  - [ ] Valider : `go build ./cmd/quotation`

- [ ] **2.1.2** Base de données SQLite Quotation
  - [ ] Créer internal/database/sqlite.go
  - [ ] Créer internal/database/quotation_repository.go
  - [ ] Interface QuotationRepository (CRUD)
  - [ ] Implémentation SQLite
  - [ ] Créer internal/database/quotation_repository_test.go
  - [ ] Test Create et GetByID
  - [ ] Test UpdateStatus
  - [ ] Test GetExpiredDevis
  - [ ] Valider : tests passent, couverture > 80%

- [ ] **2.1.3** Producteur Kafka Quotation
  - [ ] Ajouter dépendances Kafka Go
  - [ ] Créer schemas/DevisGenere.avsc
  - [ ] Créer schemas/DevisExpire.avsc
  - [ ] Créer internal/kafka/producer.go
  - [ ] Créer internal/kafka/avro.go (sérialisation)
  - [ ] Créer tests/integration/kafka_producer_test.go
  - [ ] Valider : message visible dans Kafka UI

- [ ] **2.1.4** Logique métier Quotation
  - [ ] Créer internal/services/quotation_service.go
  - [ ] Méthode GenererDevis (calcul prime, DB, Kafka)
  - [ ] Méthode VerifierExpirations
  - [ ] Créer internal/services/quotation_service_test.go
  - [ ] Créer internal/services/mocks/quotation_mocks.go
  - [ ] Valider : tests unitaires passent

- [ ] **2.1.5** API HTTP et métriques Quotation
  - [ ] Ajouter dépendances (gorilla/mux, prometheus)
  - [ ] Créer internal/observability/metrics.go
  - [ ] Compteurs devis_generes_total, devis_expires_total
  - [ ] Histogramme devis_generation_duration_seconds
  - [ ] Créer cmd/quotation/handlers.go
  - [ ] Handler POST /api/devis
  - [ ] Handler GET /api/devis/{id}
  - [ ] Handler GET /health
  - [ ] Handler GET /metrics
  - [ ] Mettre à jour cmd/quotation/main.go (init complète)
  - [ ] Créer docker/quotation/Dockerfile
  - [ ] Ajouter service quotation au docker-compose (port 8081)
  - [ ] Valider : API accessible, événements produits

- [ ] **2.1.6** Tests d'intégration Quotation
  - [ ] Créer tests/integration/quotation_test.go
  - [ ] Test flux complet génération
  - [ ] Test expiration
  - [ ] Créer tests/integration/helpers.go (consumer test)
  - [ ] Ajouter `make test-integration-quotation`
  - [ ] Valider : tous les tests passent

## Étape 2.2 : Service Souscription

- [ ] **2.2.1** Structure et modèles Souscription
  - [ ] Créer internal/models/souscription.go (struct Contrat)
  - [ ] Créer internal/models/events_souscription.go
  - [ ] Créer schemas/ContratEmis.avsc
  - [ ] Créer schemas/ContratModifie.avsc
  - [ ] Créer schemas/ContratResilie.avsc
  - [ ] Créer cmd/souscription/main.go minimal
  - [ ] Valider : `go build ./cmd/souscription`

- [ ] **2.2.2** Consumer Kafka Souscription
  - [ ] Créer internal/kafka/consumer.go
  - [ ] Consumer group souscription-service
  - [ ] Méthode Start, Stop, RegisterHandler
  - [ ] Créer internal/kafka/avro_deserialize.go
  - [ ] Créer internal/services/souscription_handlers.go
  - [ ] Handler DevisGenere
  - [ ] Handler SinistreDeclare
  - [ ] Handler IndemnisationEffectuee
  - [ ] Créer tests/integration/kafka_consumer_test.go
  - [ ] Valider : événements consommés (logs)

- [ ] **2.2.3** Logique métier et API Souscription
  - [ ] Créer internal/database/souscription_repository.go
  - [ ] Créer internal/services/souscription_service.go
  - [ ] Méthode ConvertirDevis
  - [ ] Méthode ModifierContrat
  - [ ] Méthode ResilierContrat
  - [ ] Créer cmd/souscription/handlers.go
  - [ ] POST /api/contrats/convertir
  - [ ] GET /api/contrats/{id}
  - [ ] PUT /api/contrats/{id}/modifier
  - [ ] POST /api/contrats/{id}/resilier
  - [ ] GET /health, /metrics
  - [ ] Créer internal/observability/souscription_metrics.go
  - [ ] Créer docker/souscription/Dockerfile
  - [ ] Ajouter service souscription au docker-compose (port 8082)
  - [ ] Valider : flux Devis -> Contrat fonctionne

## Étape 2.3 : Service Réclamation

- [ ] **2.3.1** Implémentation complète Réclamation
  - [ ] Créer internal/models/reclamation.go (Sinistre, Indemnisation)
  - [ ] Créer schemas/SinistreDeclare.avsc
  - [ ] Créer schemas/SinistreEvalue.avsc
  - [ ] Créer schemas/IndemnisationEffectuee.avsc
  - [ ] Créer internal/database/reclamation_repository.go
  - [ ] Créer internal/services/reclamation_service.go
  - [ ] Méthode DeclarerSinistre
  - [ ] Méthode EvaluerSinistre
  - [ ] Méthode Indemniser
  - [ ] Créer internal/services/reclamation_handlers.go
  - [ ] Handler ContratEmis
  - [ ] Handler ContratResilie
  - [ ] Créer cmd/reclamation/main.go et handlers.go
  - [ ] POST /api/sinistres
  - [ ] PUT /api/sinistres/{id}/evaluer
  - [ ] POST /api/sinistres/{id}/indemniser
  - [ ] GET /health, /metrics
  - [ ] Créer docker/reclamation/Dockerfile
  - [ ] Ajouter service reclamation au docker-compose (port 8083)
  - [ ] Valider : flux complet Devis -> Contrat -> Sinistre -> Indemnisation

## Étape 2.4 : Dashboard de Contrôle

- [ ] **2.4.1** Structure du Dashboard
  - [ ] Créer web/templates/layout.html
  - [ ] Inclure HTMX via CDN
  - [ ] CSS de base ou Tailwind CDN
  - [ ] Créer web/templates/index.html
  - [ ] Section contrôle (Start/Stop, vitesse, scénario)
  - [ ] Section statistiques (placeholder)
  - [ ] Section événements (placeholder)
  - [ ] Créer web/static/style.css
  - [ ] Créer web/handlers/dashboard.go
  - [ ] Handler GET /
  - [ ] Handler GET /health, /metrics
  - [ ] Mettre à jour cmd/dashboard/main.go
  - [ ] Créer docker/dashboard/Dockerfile
  - [ ] Ajouter service dashboard au docker-compose (port 8080)
  - [ ] Valider : http://localhost:8080 affiche la page

- [ ] **2.4.2** Événements temps réel (SSE)
  - [ ] Créer internal/services/event_aggregator.go
  - [ ] Consumer tous les topics
  - [ ] Channel pour distribution SSE
  - [ ] Struct EventDisplay
  - [ ] Créer web/handlers/events.go
  - [ ] Handler GET /api/events/stream (SSE)
  - [ ] Handler GET /api/events/recent
  - [ ] Mettre à jour index.html avec hx-sse
  - [ ] Créer web/templates/partials/event_row.html
  - [ ] CSS pour animations
  - [ ] Valider : événements apparaissent en temps réel

- [ ] **2.4.3** Diagramme de flux animé
  - [ ] Créer web/templates/partials/flow_diagram.html (SVG)
  - [ ] 3 boîtes : Quotation, Souscription, Réclamation
  - [ ] Flèches entre systèmes
  - [ ] Créer web/static/flow.js
  - [ ] Fonction animateEvent(from, to, eventType)
  - [ ] Animations CSS (pulse, couleur)
  - [ ] Intégrer dans index.html
  - [ ] Connecter aux événements SSE
  - [ ] Valider : animations visibles lors des événements

## Étape 2.5 : Simulateur d'Événements

- [ ] **2.5.1** Générateur d'événements
  - [ ] Créer cmd/simulator/main.go (port 8084)
  - [ ] Créer internal/services/simulator.go
  - [ ] Struct Simulator (Speed, Scenario, Running)
  - [ ] Méthode Start, Stop, SetSpeed, SetScenario
  - [ ] Créer internal/services/scenarios.go
  - [ ] Scénario NORMAL
  - [ ] Scénario PIC_CHARGE
  - [ ] Scénario ERREURS
  - [ ] Scénario CONSOMMATEUR_LENT
  - [ ] Scénario SERVICE_EN_PANNE
  - [ ] Créer internal/services/data_generator.go
  - [ ] Génération noms clients
  - [ ] Génération types de biens
  - [ ] Génération valeurs réalistes
  - [ ] Génération types de sinistres
  - [ ] Créer cmd/simulator/handlers.go
  - [ ] POST /api/simulator/start
  - [ ] POST /api/simulator/stop
  - [ ] PUT /api/simulator/speed
  - [ ] PUT /api/simulator/scenario
  - [ ] GET /api/simulator/status
  - [ ] Créer docker/simulator/Dockerfile
  - [ ] Ajouter service simulator au docker-compose (port 8084)
  - [ ] Valider : simulation génère des événements

- [ ] **2.5.2** Intégration Dashboard-Simulateur
  - [ ] Mettre à jour web/handlers/dashboard.go (proxy simulateur)
  - [ ] Mettre à jour index.html
  - [ ] Bouton Start avec hx-post
  - [ ] Bouton Stop avec hx-post
  - [ ] Sélecteur vitesse avec hx-put
  - [ ] Sélecteur scénario avec hx-put
  - [ ] Indicateur statut avec hx-get polling
  - [ ] Créer web/templates/partials/simulation_status.html
  - [ ] CSS boutons actifs/inactifs
  - [ ] Valider : contrôle complet depuis le dashboard

- [ ] **2.5.3** Dashboards Grafana complets
  - [ ] Mettre à jour kafka-overview.json (métriques réelles)
  - [ ] Créer services-health.json
  - [ ] Row Quotation (status, req/sec, latence, erreurs)
  - [ ] Row Souscription
  - [ ] Row Réclamation
  - [ ] Créer business-events.json
  - [ ] Pie chart par type d'événement
  - [ ] Graphe évolution temporelle
  - [ ] Table derniers événements
  - [ ] Créer simulation-control.json
  - [ ] Panel statut simulateur
  - [ ] Panel vitesse
  - [ ] Panel scénario
  - [ ] Panel total événements
  - [ ] Graphe événements/sec
  - [ ] Valider : dashboards avec données réelles

- [ ] **2.5.4** Tests d'intégration Phase 2
  - [ ] Créer tests/integration/phase2_test.go
  - [ ] Test flux complet sans simulateur
  - [ ] Créer tests/integration/simulator_test.go
  - [ ] Test start/stop
  - [ ] Test changement vitesse
  - [ ] Test scénario NORMAL
  - [ ] Créer tests/integration/dashboard_test.go
  - [ ] Test page accessible
  - [ ] Test SSE
  - [ ] Test contrôles
  - [ ] Ajouter `make test-integration-phase2`
  - [ ] Créer scripts/e2e-phase2.sh
  - [ ] Valider : tous les tests passent

- [ ] **2.5.5** Documentation et Tag Phase 2
  - [ ] Compléter docs/01-producteur-consommateur/README.md
  - [ ] Théorie du patron
  - [ ] Schéma architecture
  - [ ] Description services
  - [ ] Guide utilisation dashboard
  - [ ] Exercices de compréhension
  - [ ] Mettre à jour README.md
  - [ ] Mettre à jour CHANGELOG.md
  - [ ] Vérification finale (make reset, up, health)
  - [ ] Créer tag v2.0-pubsub
  - [ ] Valider : tag fonctionnel

---

# PHASE 3 : Event Sourcing

## Étape 3.1 : Refactoring pour Event Store

- [ ] **3.1.1** Abstraction Event Store
  - [ ] Créer internal/eventsourcing/event_store.go
  - [ ] Interface EventStore (Append, Load, LoadFromVersion)
  - [ ] Struct Event (ID, AggregateID, Type, Version, Timestamp, Data)
  - [ ] Créer internal/eventsourcing/kafka_event_store.go
  - [ ] Implémentation avec Kafka
  - [ ] Créer internal/eventsourcing/aggregate.go
  - [ ] Interface Aggregate
  - [ ] BaseAggregate
  - [ ] Tests unitaires
  - [ ] Tests intégration avec Kafka
  - [ ] Valider : Event Store fonctionnel

- [ ] **3.1.2** Agrégat Contrat avec Event Sourcing
  - [ ] Créer internal/aggregates/contrat.go
  - [ ] Struct ContratAggregate
  - [ ] Méthodes Emettre, Modifier, Resilier, EnregistrerSinistre
  - [ ] Méthode ApplyEvent
  - [ ] Mettre à jour souscription_service.go
  - [ ] Créer internal/projections/contrat_projection.go
  - [ ] Consumer pour mise à jour SQLite
  - [ ] Tests unitaires
  - [ ] Tests intégration
  - [ ] Valider : état reconstruit depuis Kafka

## Étape 3.2 : Reconstruction d'état

- [ ] **3.2.1** Rebuild depuis Kafka
  - [ ] Créer internal/eventsourcing/rebuilder.go
  - [ ] Fonction RebuildProjection
  - [ ] Support rebuild partiel
  - [ ] Affichage progression
  - [ ] Mettre à jour cmd/souscription/main.go (flag --rebuild)
  - [ ] Métrique rebuild_events_processed_total
  - [ ] Endpoint GET /api/debug/state
  - [ ] Endpoint GET /api/debug/events
  - [ ] Mettre à jour dashboard (indicateur rebuild)
  - [ ] Documentation reconstruction
  - [ ] Valider : reconstruction fonctionne après suppression DB

## Étape 3.3 : Snapshots

- [ ] **3.3.1** Mécanisme de snapshots
  - [ ] Créer internal/eventsourcing/snapshot_store.go
  - [ ] Interface SnapshotStore
  - [ ] Struct Snapshot
  - [ ] Créer internal/eventsourcing/sqlite_snapshot_store.go
  - [ ] Implémentation SQLite
  - [ ] Mettre à jour ContratAggregate (ToSnapshot, FromSnapshot)
  - [ ] Création snapshot tous les N événements
  - [ ] Mettre à jour rebuilder (depuis snapshot)
  - [ ] Métriques snapshots_created_total
  - [ ] Documentation snapshots
  - [ ] Valider : reconstruction plus rapide avec snapshot

- [ ] **3.3.2** Tests et Tag Phase 3
  - [ ] Créer tests/integration/phase3_test.go
  - [ ] Test reconstruction complète
  - [ ] Test reconstruction depuis snapshot
  - [ ] Test cohérence
  - [ ] Test performance
  - [ ] Compléter docs/02-event-sourcing/README.md
  - [ ] Théorie Event Sourcing
  - [ ] Avantages/Inconvénients
  - [ ] Schéma implémentation
  - [ ] Exercices
  - [ ] Créer tag v3.0-eventsourcing
  - [ ] Valider : tag fonctionnel

---

# PHASE 4 : CQRS

## Étape 4.1 : Séparation Command/Query

- [ ] **4.1.1** Séparation des modèles
  - [ ] Créer internal/cqrs/commands.go
  - [ ] Struct pour chaque commande (EmettreContratCommand, etc.)
  - [ ] Créer internal/cqrs/queries.go
  - [ ] Struct pour chaque query (GetContratQuery, ListContratsQuery)
  - [ ] Séparer les modèles write/read
  - [ ] Valider : compilation OK

- [ ] **4.1.2** Command handlers
  - [ ] Créer internal/cqrs/command_handlers.go
  - [ ] Handler pour chaque commande
  - [ ] Validation des commandes
  - [ ] Exécution via agrégat
  - [ ] Créer internal/cqrs/command_bus.go
  - [ ] Dispatch des commandes
  - [ ] Tests unitaires
  - [ ] Valider : commandes exécutées correctement

## Étape 4.2 : Vues matérialisées

- [ ] **4.2.1** Vues optimisées
  - [ ] Créer internal/cqrs/read_models.go
  - [ ] ContratListItem (vue liste)
  - [ ] ContratDetail (vue détail)
  - [ ] ContratStatistics (agrégations)
  - [ ] Créer internal/cqrs/projections/
  - [ ] Projection liste
  - [ ] Projection détail
  - [ ] Projection statistiques
  - [ ] Valider : vues créées

- [ ] **4.2.2** Query handlers
  - [ ] Créer internal/cqrs/query_handlers.go
  - [ ] Handler GetContrat
  - [ ] Handler ListContrats
  - [ ] Handler GetStatistics
  - [ ] Créer internal/cqrs/query_bus.go
  - [ ] Tests unitaires
  - [ ] Valider : queries fonctionnelles

## Étape 4.3 : Synchronisation

- [ ] **4.3.1** Eventual consistency
  - [ ] Documenter le délai de synchronisation
  - [ ] Ajouter métriques de lag
  - [ ] Indicateur dans le dashboard
  - [ ] Gestion des lectures stale
  - [ ] Tests de cohérence
  - [ ] Valider : comportement documenté

- [ ] **4.3.2** Tests et Tag Phase 4
  - [ ] Créer tests/integration/phase4_test.go
  - [ ] Test séparation command/query
  - [ ] Test vues matérialisées
  - [ ] Test eventual consistency
  - [ ] Compléter docs/03-cqrs/README.md
  - [ ] Créer tag v4.0-cqrs
  - [ ] Valider : tag fonctionnel

---

# PHASE 5 : Saga Choreography

## Étape 5.1 : Transactions distribuées

- [ ] **5.1.1** Processus de souscription complète
  - [ ] Définir le flux Saga complet
  - [ ] Créer internal/saga/souscription_saga.go
  - [ ] États du processus
  - [ ] Transitions
  - [ ] Créer internal/saga/saga_state.go
  - [ ] Persistance de l'état
  - [ ] Valider : flux défini

- [ ] **5.1.2** Coordination par événements
  - [ ] Créer les événements de processus
  - [ ] SouscriptionDemarree
  - [ ] VerificationTerminee
  - [ ] PaiementEffectue
  - [ ] SouscriptionFinalisee
  - [ ] Implémenter les handlers
  - [ ] Tests unitaires
  - [ ] Valider : coordination fonctionne

## Étape 5.2 : Compensation

- [ ] **5.2.1** Événements de compensation
  - [ ] Créer SouscriptionAnnulee
  - [ ] Créer PaiementRembourse
  - [ ] Créer ContratAnnule
  - [ ] Implémenter la logique de compensation
  - [ ] Valider : compensation définie

- [ ] **5.2.2** Rollback automatique
  - [ ] Détecter les échecs
  - [ ] Déclencher la compensation
  - [ ] Logger les étapes
  - [ ] Tests d'intégration
  - [ ] Valider : rollback automatique fonctionne

## Étape 5.3 : Scénarios complexes

- [ ] **5.3.1** Scénarios de test
  - [ ] Scénario succès complet
  - [ ] Scénario échec vérification
  - [ ] Scénario échec paiement
  - [ ] Scénario timeout
  - [ ] Implémenter dans le simulateur
  - [ ] Valider : scénarios exécutables

- [ ] **5.3.2** Tests et Tag Phase 5
  - [ ] Créer tests/integration/phase5_test.go
  - [ ] Tests tous scénarios
  - [ ] Compléter docs/04-saga-choreography/README.md
  - [ ] Créer tag v5.0-saga
  - [ ] Valider : tag fonctionnel

---

# PHASE 6 : Dead Letter Queue

## Étape 6.1 : Gestion des erreurs

- [ ] **6.1.1** Erreurs de traitement
  - [ ] Créer internal/dlq/error_handler.go
  - [ ] Classification des erreurs (retryable, non-retryable)
  - [ ] Logging détaillé
  - [ ] Métriques d'erreurs
  - [ ] Valider : erreurs classifiées

- [ ] **6.1.2** Retry avec backoff
  - [ ] Créer internal/dlq/retry.go
  - [ ] Backoff exponentiel
  - [ ] Nombre max de retries
  - [ ] Configuration par topic
  - [ ] Tests unitaires
  - [ ] Valider : retry fonctionne

## Étape 6.2 : Dead Letter Queue

- [ ] **6.2.1** Configuration DLQ
  - [ ] Topic dlq.errors configuré
  - [ ] Créer internal/dlq/dlq_producer.go
  - [ ] Envoi vers DLQ après max retries
  - [ ] Métadonnées (erreur originale, tentatives)
  - [ ] Valider : messages en DLQ

- [ ] **6.2.2** Interface de visualisation
  - [ ] Ajouter page DLQ dans dashboard
  - [ ] Liste des messages en erreur
  - [ ] Détail de l'erreur
  - [ ] Bouton replay
  - [ ] Bouton supprimer
  - [ ] Valider : interface fonctionnelle

## Étape 6.3 : Scénarios de panne

- [ ] **6.3.1** Simulation de pannes
  - [ ] Scénario service indisponible
  - [ ] Scénario message invalide
  - [ ] Scénario timeout
  - [ ] Ajouter au simulateur
  - [ ] Valider : pannes simulées

- [ ] **6.3.2** Tests et Tag Phase 6
  - [ ] Créer tests/integration/phase6_test.go
  - [ ] Tests scénarios de panne
  - [ ] Tests DLQ et replay
  - [ ] Compléter docs/05-dead-letter-queue/README.md
  - [ ] Créer tag v6.0-dlq
  - [ ] Valider : tag fonctionnel

---

# PHASE 7 : Finalisation

## Étape 7.1 : Documentation complète

- [ ] **7.1.1** Documentation tous les patrons
  - [ ] Revoir docs/00-introduction.md
  - [ ] Revoir docs/01-producteur-consommateur/
  - [ ] Revoir docs/02-event-sourcing/
  - [ ] Revoir docs/03-cqrs/
  - [ ] Revoir docs/04-saga-choreography/
  - [ ] Revoir docs/05-dead-letter-queue/
  - [ ] Ajouter schémas manquants
  - [ ] Ajouter exercices
  - [ ] Valider : documentation complète

- [ ] **7.1.2** Guide d'utilisation final
  - [ ] Mettre à jour README.md
  - [ ] Guide de démarrage rapide
  - [ ] Guide avancé
  - [ ] FAQ
  - [ ] Troubleshooting complet
  - [ ] Valider : guide utilisable

## Étape 7.2 : Tests de charge

- [ ] **7.2.1** Tests avec k6
  - [ ] Installer k6
  - [ ] Créer tests/load/quotation.js
  - [ ] Créer tests/load/souscription.js
  - [ ] Créer tests/load/reclamation.js
  - [ ] Créer tests/load/full_scenario.js
  - [ ] Scénario charge normale (100 evt/sec, 5 min)
  - [ ] Scénario pic (500 evt/sec)
  - [ ] Scénario endurance (50 evt/sec, 30 min)
  - [ ] Ajouter `make test-load`
  - [ ] Valider : tests exécutables

- [ ] **7.2.2** Rapport de performance
  - [ ] Exécuter tous les tests de charge
  - [ ] Collecter les métriques
  - [ ] Créer docs/performance-report.md
  - [ ] Graphiques de performance
  - [ ] Recommandations
  - [ ] Valider : rapport complet

## Étape 7.3 : Polish et Release

- [ ] **7.3.1** Polish UI et UX
  - [ ] Revoir le design du dashboard
  - [ ] Améliorer les animations
  - [ ] Tester sur différentes résolutions
  - [ ] Corriger les bugs UI
  - [ ] Optimiser les performances front
  - [ ] Valider : UI propre

- [ ] **7.3.2** Tag final v7.0-final
  - [ ] Revue complète du code
  - [ ] Nettoyage (code mort, TODOs)
  - [ ] Vérification tous les tests
  - [ ] Vérification documentation
  - [ ] Mettre à jour CHANGELOG.md
  - [ ] Créer tag v7.0-final
  - [ ] Préparer release notes
  - [ ] Valider : projet finalisé

---

# Checklist Globale de Validation

## Infrastructure
- [ ] `docker-compose up` démarre sans erreur
- [ ] Tous les services sont healthy
- [ ] Kafka UI accessible et fonctionnel
- [ ] Grafana accessible avec dashboards
- [ ] Jaeger accessible
- [ ] Prometheus collecte les métriques

## Services
- [ ] Quotation : API fonctionnelle, événements produits
- [ ] Souscription : Consumer + API fonctionnels
- [ ] Réclamation : Consumer + API fonctionnels
- [ ] Dashboard : Interface accessible, SSE fonctionne
- [ ] Simulateur : Génération d'événements fonctionne

## Patrons d'architecture
- [ ] Pub/Sub : Flux complet fonctionne
- [ ] Event Sourcing : Reconstruction d'état fonctionne
- [ ] CQRS : Séparation lecture/écriture fonctionne
- [ ] Saga : Transactions distribuées fonctionnent
- [ ] DLQ : Gestion des erreurs fonctionne

## Tests
- [ ] Tests unitaires : couverture > 80%
- [ ] Tests d'intégration : tous passent
- [ ] Tests de charge : rapport généré

## Documentation
- [ ] README complet
- [ ] Documentation par patron
- [ ] Exercices définis
- [ ] CHANGELOG à jour

---

*Dernière mise à jour : 18 janvier 2026*
