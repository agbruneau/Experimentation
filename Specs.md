# Cahier des Charges - kafka-eda-lab

## Statut : VALIDÉ

**Date de validation :** 18 janvier 2026

---

## Résumé Exécutif

| Élément | Description |
|---------|-------------|
| **Projet** | `kafka-eda-lab` — Simulation pédagogique EDA |
| **Objectif** | Former des architectes de domaine aux patrons d'architecture événementielle avec Apache Kafka |
| **Domaine métier** | Assurance Dommages (Quotation, Souscription, Réclamation) |
| **Stack technique** | Go, Kafka KRaft, Avro, Docker Compose, HTMX |
| **Observabilité** | Prometheus, Grafana, Loki, Jaeger |
| **Patrons couverts** | Pub/Sub, Event Sourcing, CQRS, Saga, Dead Letter Queue |
| **Développement** | 7 phases incrémentales, génération intégrale via Claude Code |
| **Plateforme** | Windows uniquement |

---

## 1. Présentation du Projet

**Nom du projet :** `kafka-eda-lab`

**Répertoire de développement :** `C:\Users\agbru\OneDrive\Documents\GitHub\Experimentation`

**Description :** Simulation académique dans le domaine de l'interopérabilité en écosystèmes d'entreprise, fondée sur les patrons d'architecture EDA (Event Driven Architecture) avec Apache Kafka et des mécanismes d'observabilité.

**Outils de développement :** Claude Code et Cursor (génération intégrale du code)

---

## 2. Contexte et Public Cible

**Public cible :** Architectes de domaine en interopérabilité des écosystèmes en entreprise

**Profil :** Professionnels seniors responsables de la conception et de l'intégration des systèmes d'information dans un contexte d'entreprise étendue.

**Niveau technique préalable :**
- Apache Kafka : Aucune expérience
- Programmation : Débutant, lecture de code uniquement
- Langage préféré : Go
- Conteneurisation (Docker/Kubernetes) : Aucune expérience

**Mode de développement :**
La solution sera entièrement conçue, développée, testée et exécutée via Claude Code. L'utilisateur final n'aura pas à coder lui-même.

---

## 3. Objectifs Pédagogiques

À l'issue de la simulation, l'architecte sera capable de :

1. **Maîtriser les concepts fondamentaux de Kafka**
   - Topics, partitions, offsets, consumer groups
   - Producteurs et consommateurs
   - Garanties de livraison

2. **Choisir le bon patron EDA selon le contexte**
   - Pub/Sub, Event Sourcing, CQRS, Saga, etc.
   - Critères de décision et cas d'usage appropriés

3. **Concevoir une architecture d'interopérabilité événementielle**
   - Modélisation des flux entre systèmes
   - Intégration dans un écosystème d'entreprise

4. **Exploiter l'observabilité pour le diagnostic**
   - Lecture et interprétation des métriques
   - Identification et résolution de problèmes

---

## 4. Domaine Métier

**Secteur :** Services financiers (écosystème complet)

**Sous-domaines couverts :**
- Banque (comptes, transactions, virements)
- Finance (investissements, marchés, portefeuilles)
- Assurance de personnes (vie, santé, prévoyance)
- Assurance dommages (auto, habitation, responsabilité civile)

**Pertinence :** Ce secteur présente une forte exigence d'interopérabilité entre acteurs multiples, des contraintes réglementaires strictes et des flux événementiels complexes.

**Périmètre retenu pour la simulation :** Assurance Dommages

**Systèmes simulés (3) :**

| Système | Rôle métier |
|---------|-------------|
| **Quotation** | Calcul de prime, génération de devis |
| **Souscription** | Émission et gestion des contrats |
| **Réclamation** | Déclaration et traitement des sinistres |

Ces 3 systèmes couvrent le cycle de vie complet d'un contrat d'assurance dommages.

**Événements métier simulés :**

| Système source | Événement | Description |
|----------------|-----------|-------------|
| Quotation | `DevisGenere` | Devis créé pour un prospect |
| Quotation | `DevisExpire` | Devis non converti après délai |
| Souscription | `ContratEmis` | Nouveau contrat actif |
| Souscription | `ContratModifie` | Avenant au contrat |
| Souscription | `ContratResilie` | Fin de contrat |
| Réclamation | `SinistreDeclare` | Nouvelle déclaration de sinistre |
| Réclamation | `SinistreEvalue` | Expertise terminée |
| Réclamation | `IndemnisationEffectuee` | Paiement au client |

**Matrice des flux inter-systèmes :**

```
┌─────────────┐      DevisGenere       ┌──────────────┐
│  QUOTATION  │ ────────────────────►  │ SOUSCRIPTION │
└─────────────┘                        └──────┬───────┘
                                              │
                         ContratEmis          │
                         ContratResilie       │
                              │               │
                              ▼               │
                       ┌──────────────┐       │
                       │ RÉCLAMATION  │ ◄─────┘
                       └──────┬───────┘
                              │
            SinistreDeclare   │
            IndemnisationEffectuee
                              │
                              ▼
                       ┌──────────────┐
                       │ SOUSCRIPTION │ (mise à jour historique/risque)
                       └──────────────┘
```

| Événement | Producteur | Consommateur(s) | Objectif |
|-----------|------------|-----------------|----------|
| `DevisGenere` | Quotation | Souscription | Conversion en contrat |
| `ContratEmis` | Souscription | Réclamation | Autoriser les déclarations |
| `ContratResilie` | Souscription | Réclamation | Bloquer nouvelles déclarations |
| `SinistreDeclare` | Réclamation | Souscription | Historique du contrat |
| `IndemnisationEffectuee` | Réclamation | Souscription | Mise à jour du risque |

---

## 5. Patrons d'Architecture EDA

**Approche pédagogique :** Progression graduelle du simple au complexe

| Ordre | Patron | Description | Objectif pédagogique |
|-------|--------|-------------|---------------------|
| 1 | **Producteur/Consommateur (Pub/Sub)** | Base de Kafka | Comprendre les fondamentaux |
| 2 | **Event Sourcing** | État = séquence d'événements | Traçabilité et reconstruction d'état |
| 3 | **CQRS** | Séparation lecture/écriture | Optimisation et scalabilité |
| 4 | **Saga (Choreography)** | Transactions distribuées par événements | Cohérence sans coordinateur central |
| 5 | **Dead Letter Queue** | Gestion des erreurs | Résilience et reprise sur erreur |

**Structure de la simulation :** Scénario continu avec documentation pédagogique intégrée

- **Application évolutive** : Une seule base de code qui s'enrichit à chaque étape
- **Tags Git par patron** : Chaque patron correspond à un tag versionné (ex: `v2.0-pubsub`)
- **Documentation intégrée** : Chaque étape inclut :
  - Explication théorique du patron
  - Schéma d'architecture
  - Code commenté
  - Points d'attention et bonnes pratiques
  - Exercices de compréhension

---

## 6. Architecture Technique

### 6.1 Environnement d'exécution

**Choix :** Docker Compose (priorité à la facilité d'utilisation)

**Principe :** Une seule commande `docker-compose up` démarre l'ensemble de l'écosystème :
- Apache Kafka en mode KRaft (sans Zookeeper)
- Services Go (Quotation, Souscription, Réclamation)
- Stack d'observabilité (Prometheus, Grafana, Loki, Jaeger)
- Dashboard de contrôle et visualisation

**Avantages :**
- Aucune installation manuelle requise
- Environnement reproductible et isolé
- Démarrage/arrêt simplifié
- Configuration versionnée dans Git

**Prérequis utilisateur :**
- Docker Desktop installé
- Ressources minimales : RAM 8 Go recommandé

### 6.2 Observabilité

**Couverture :** Les 3 piliers complets

| Pilier | Outil | Port | Fonction |
|--------|-------|------|----------|
| **Métriques** | Prometheus + Grafana | 3000 | Dashboards, alertes, santé système |
| **Logs** | Grafana Loki | (via Grafana) | Logs centralisés, recherche, corrélation |
| **Tracing** | Jaeger | 16686 | Parcours des événements, latence par service |

**Avantages de cette stack :**
- Interface unifiée via Grafana (métriques + logs)
- Légèreté (Loki moins gourmand qu'ELK)
- Standards CNCF (Cloud Native Computing Foundation)
- Intégration native avec les applications Go via OpenTelemetry

### 6.3 Interface Utilisateur

**Type :** Tableau de bord de contrôle et visualisation

**Objectif :** Piloter et observer la simulation sans aucune saisie manuelle

**Principe clé :** Génération automatique des événements — la simulation s'exécute de manière autonome

**Section Contrôle :**
- Démarrer / Arrêter la simulation
- Sélectionner le patron d'architecture actif
- Réinitialiser l'environnement

**Paramètres de simulation (simplifiés) :**

| Paramètre | Options |
|-----------|---------|
| **Vitesse** | Lente (1 evt/sec) · Normale (5 evt/sec) · Rapide (20 evt/sec) |
| **Scénario** | Voir tableau ci-dessous |

**Scénarios disponibles :**

| Scénario | Comportement | Objectif pédagogique |
|----------|--------------|---------------------|
| **Normal** | Flux régulier sans erreur | Observer le fonctionnement nominal |
| **Pic de charge** | Volume élevé soudain | Observer le comportement sous stress |
| **Erreurs réseau** | Pertes de connexion simulées | Observer la résilience et les retry |
| **Consommateur lent** | Un service traite lentement | Observer le lag et le backpressure |
| **Service en panne** | Un service devient indisponible | Observer la Dead Letter Queue |

**Section Visualisation (double vue) :**

| Vue | Description |
|-----|-------------|
| **Timeline d'événements** | Liste chronologique des événements avec statut (succès/échec/en cours) |
| **Flux temps réel** | Diagramme animé montrant les événements circuler entre les 3 systèmes |

**Accès rapide :**
- Liens vers Grafana (métriques + logs)
- Lien vers Jaeger (tracing)

### 6.4 Technologie Front-end

**Choix :** HTMX + Go Templates

**Justification (simplicité maximale) :**
- Pas de framework JavaScript complexe
- Rendu côté serveur avec Go (stack unifiée)
- HTMX pour l'interactivité (mises à jour partielles, temps réel via SSE)
- Aucune étape de build front-end (pas de npm/webpack)
- Code lisible et maintenable

**Composants :**
- Templates HTML avec Go `html/template`
- HTMX pour les interactions dynamiques
- Server-Sent Events (SSE) pour le flux temps réel
- CSS simple (ou Tailwind CSS pour le style)

### 6.5 Format des Événements

**Choix :** Apache Avro avec Schema Registry

**Justification :**
- Standard de l'industrie pour Kafka en entreprise
- Schémas évolutifs (ajout de champs sans casser les consommateurs)
- Validation automatique des messages
- Compression efficace (messages compacts)
- Gouvernance des données via Schema Registry

**Composants :**
- **Confluent Schema Registry** (conteneur Docker)
- Schémas Avro versionnés pour chaque type d'événement
- Bibliothèque Go : `github.com/linkedin/goavro` ou `github.com/riferrei/srclient`

**Schémas à définir :**
- `DevisGenere.avsc`
- `DevisExpire.avsc`
- `ContratEmis.avsc`
- `ContratModifie.avsc`
- `ContratResilie.avsc`
- `SinistreDeclare.avsc`
- `SinistreEvalue.avsc`
- `IndemnisationEffectuee.avsc`

### 6.6 Persistance des Données

**Choix :** SQLite (base embarquée)

**Justification :**
- Aucun serveur de base de données à gérer
- Fichier unique par service, facilement réinitialisable
- Léger et rapide pour une simulation
- Parfait pour le contexte pédagogique

**Organisation :**
| Service | Fichier | Données stockées |
|---------|---------|------------------|
| Quotation | `quotation.db` | Devis générés |
| Souscription | `souscription.db` | Contrats, historique |
| Réclamation | `reclamation.db` | Sinistres, indemnisations |

**Note Event Sourcing :** À l'étape 2 (Event Sourcing), l'état sera reconstruit depuis Kafka, SQLite servira de vue matérialisée pour les requêtes.

### 6.7 Distribution Kafka

**Choix :** Apache Kafka en mode KRaft (sans Zookeeper)

**Justification :**
- Architecture simplifiée (pas de Zookeeper à gérer)
- Moins de conteneurs à déployer
- Démarrage plus rapide
- Futur standard officiel d'Apache Kafka
- Configuration réduite

**Version cible :** Kafka 3.6+ (KRaft production-ready)

**Configuration Docker :**
```yaml
# Un seul conteneur Kafka en mode KRaft
kafka:
  image: apache/kafka:3.7.0
  environment:
    KAFKA_NODE_ID: 1
    KAFKA_PROCESS_ROLES: broker,controller
    KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
```

---

## 7. Documentation Pédagogique

### 7.1 Formats

**Double approche :**
- **Markdown dans le repository** — Documentation complète, versionnée avec le code
- **Intégrée dans l'interface Web** — Résumés et guides accessibles pendant l'utilisation

### 7.2 Structure par Patron

Chaque patron d'architecture inclura :

| Section | Description | Format |
|---------|-------------|--------|
| **Théorie** | Explication du patron, principes fondamentaux | Markdown + Web |
| **Cas d'usage** | Exemples concrets en contexte entreprise | Markdown + Web |
| **Schéma d'architecture** | Diagramme visuel du patron appliqué | Image + Web |
| **Points d'attention** | Pièges courants, bonnes pratiques | Markdown + Web |
| **Code source** | Liens directs vers les fichiers pertinents | Markdown |

### 7.3 Organisation des Fichiers

```
docs/
├── 00-introduction.md
├── 01-producteur-consommateur/
│   ├── README.md
│   ├── theorie.md
│   ├── schema.png
│   └── exercices.md
├── 02-event-sourcing/
├── 03-cqrs/
├── 04-saga-choreography/
└── 05-dead-letter-queue/
```

---

## 8. Qualité et Tests

### 8.1 Stratégie de Tests

**Niveau :** Complet (3 niveaux de tests)

| Type | Portée | Outils |
|------|--------|--------|
| **Tests unitaires** | Logique métier, validations, transformations | Go testing + testify |
| **Tests d'intégration** | Flux Kafka bout-en-bout, interactions services | testcontainers-go |
| **Tests de charge** | Performance sous stress, limites du système | k6 ou vegeta |

### 8.2 Couverture Cible

| Composant | Couverture minimale |
|-----------|---------------------|
| Services métier (Quotation, Souscription, Réclamation) | 80% |
| Producteurs/Consommateurs Kafka | 70% |
| Interface Web | Tests fonctionnels des endpoints |

### 8.3 Scénarios de Tests de Charge

Alignés avec les scénarios de simulation :
- **Charge normale** : 100 evt/sec pendant 5 min
- **Pic de charge** : Montée à 500 evt/sec
- **Endurance** : 50 evt/sec pendant 30 min

## 9. Structure du Projet

**Organisation :** Monorepo

**Justification :**
- Un seul `git clone` pour tout le projet
- Versionnement cohérent entre tous les composants
- Navigation simplifiée pour l'apprentissage
- Refactoring facilité

**Arborescence :**

```
kafka-eda-lab/
├── cmd/                        # Points d'entrée des services
│   ├── quotation/
│   │   └── main.go
│   ├── souscription/
│   │   └── main.go
│   ├── reclamation/
│   │   └── main.go
│   ├── dashboard/
│   │   └── main.go
│   └── simulator/              # Générateur d'événements
│       └── main.go
├── internal/                   # Code interne partagé
│   ├── kafka/                  # Client Kafka, producteurs, consommateurs
│   ├── models/                 # Structures de données
│   ├── observability/          # Métriques, logs, tracing
│   └── database/               # Accès SQLite
├── pkg/                        # Code réutilisable exportable
├── schemas/                    # Schémas Avro (.avsc)
├── web/                        # Interface Web
│   ├── templates/              # Go templates HTML
│   ├── static/                 # CSS, JS (HTMX), images
│   └── handlers/               # Handlers HTTP
├── docker/                     # Dockerfiles par service
├── docs/                       # Documentation pédagogique
├── tests/                      # Tests d'intégration et de charge
│   ├── integration/
│   └── load/
├── docker-compose.yml          # Orchestration complète
├── Makefile                    # Commandes utilitaires
└── README.md
```

## 10. Exigences Non Fonctionnelles

### 10.1 Performance

| Métrique | Cible |
|----------|-------|
| Latence par événement | < 2 secondes |
| Débit minimal | 20 événements/seconde |
| Temps de démarrage (docker-compose up) | < 2 minutes |

**Note :** Objectifs adaptés au contexte pédagogique, non optimisés pour la production.

### 10.2 Sécurité

**Niveau :** Aucun (simulation locale uniquement)

- Pas d'authentification sur le dashboard
- Pas de chiffrement TLS sur Kafka
- Pas de gestion des secrets

**Justification :** Projet académique destiné à tourner en local, la sécurité ajouterait une complexité non pertinente pour l'objectif pédagogique.

### 10.3 Compatibilité

| Critère | Exigence |
|---------|----------|
| **Système d'exploitation** | Windows uniquement |
| **Docker** | Docker Desktop for Windows |
| **Architecture** | x64 (AMD64) |

### 10.4 Ressources Minimales

| Ressource | Minimum recommandé |
|-----------|-------------------|
| RAM | 8 Go |
| CPU | 4 cœurs |
| Disque | 10 Go disponibles |

## 11. Phases de Développement

**Approche :** Développement incrémental par patron d'architecture

| Phase | Contenu | Livrable | Critère de validation |
|-------|---------|----------|----------------------|
| **Phase 1** | Infrastructure de base | Docker Compose fonctionnel (Kafka KRaft, Prometheus, Grafana, Loki, Jaeger) | `docker-compose up` démarre sans erreur |
| **Phase 2** | Producteur/Consommateur | 3 services Go + événements + dashboard minimal | Événements visibles dans le dashboard |
| **Phase 3** | Event Sourcing | Reconstruction d'état depuis Kafka | État reconstruit après redémarrage d'un service |
| **Phase 4** | CQRS | Séparation lecture/écriture | Requêtes de lecture sur vue matérialisée |
| **Phase 5** | Saga Choreography | Transactions distribuées | Scénario de compensation exécuté |
| **Phase 6** | Dead Letter Queue | Gestion des erreurs + scénarios de panne | Messages en erreur visibles dans DLQ |
| **Phase 7** | Finalisation | Documentation complète, tests, polish | Tous les tests passent, doc complète |

### 11.1 Stratégie Git

**Approche :** Développement linéaire sur `main` avec tags

**Organisation :**
- Tout le développement se fait sur la branche `main`
- Un tag Git marque la fin de chaque phase

**Tags prévus :**
| Tag | Description |
|-----|-------------|
| `v1.0-infra` | Phase 1 — Infrastructure de base |
| `v2.0-pubsub` | Phase 2 — Producteur/Consommateur |
| `v3.0-eventsourcing` | Phase 3 — Event Sourcing |
| `v4.0-cqrs` | Phase 4 — CQRS |
| `v5.0-saga` | Phase 5 — Saga Choreography |
| `v6.0-dlq` | Phase 6 — Dead Letter Queue |
| `v7.0-final` | Phase 7 — Version finale |

**Avantages :**
- Historique simple et lisible
- Navigation facile via `git checkout v2.0-pubsub`
- Pas de complexité de merge

## 12. Dashboards Grafana

**Dashboards pré-configurés :**

| Dashboard | Métriques affichées | Usage |
|-----------|---------------------|-------|
| **Kafka Overview** | Topics, partitions, messages/sec, lag consommateurs, réplication | Santé de l'infrastructure Kafka |
| **Services Health** | Statut UP/DOWN, latence p50/p95/p99, taux d'erreur, requêtes/sec | Monitoring des 3 services métier |
| **Business Events** | Compteurs par type d'événement, tendances, répartition | Vision métier des flux |
| **Simulation Control** | Événements générés, vitesse actuelle, scénario actif, durée | Pilotage de la simulation |

**Format :** Fichiers JSON provisionnés automatiquement au démarrage de Grafana

**Emplacement :**
```
docker/grafana/
├── provisioning/
│   ├── dashboards/
│   │   ├── kafka-overview.json
│   │   ├── services-health.json
│   │   ├── business-events.json
│   │   └── simulation-control.json
│   └── datasources/
│       ├── prometheus.yml
│       └── loki.yml
```

## 13. Commandes Utilitaires (Makefile)

**Objectif :** Simplifier l'utilisation du projet avec des commandes mémorisables

| Commande | Action | Détail |
|----------|--------|--------|
| `make up` | Démarre l'environnement | `docker-compose up -d` |
| `make down` | Arrête l'environnement | `docker-compose down` |
| `make reset` | Réinitialisation complète | Stop + supprime volumes + redémarre |
| `make logs` | Affiche les logs | `docker-compose logs -f` |
| `make test` | Lance les tests | Tests unitaires + intégration |
| `make test-load` | Tests de charge | Exécute k6/vegeta |
| `make build` | Compile les services | Build des binaires Go |
| `make dashboard` | Ouvre le dashboard | Lance le navigateur sur localhost |

**Commandes bonus :**
| Commande | Action |
|----------|--------|
| `make status` | Affiche l'état de tous les conteneurs |
| `make kafka-ui` | Ouvre l'interface Kafka dans le navigateur |
| `make grafana` | Ouvre Grafana dans le navigateur |
| `make jaeger` | Ouvre Jaeger dans le navigateur |

---

## 14. Annexes

### 14.1 Ports Exposés

| Service | Port | URL |
|---------|------|-----|
| Dashboard | 8080 | http://localhost:8080 |
| Grafana | 3000 | http://localhost:3000 |
| Jaeger UI | 16686 | http://localhost:16686 |
| Prometheus | 9090 | http://localhost:9090 |
| Kafka | 9092 | localhost:9092 |
| Schema Registry | 8081 | http://localhost:8081 |

### 14.2 Glossaire

| Terme | Définition |
|-------|------------|
| **EDA** | Event Driven Architecture — Architecture pilotée par les événements |
| **Kafka** | Plateforme de streaming d'événements distribuée |
| **KRaft** | Mode Kafka sans Zookeeper (Kafka Raft) |
| **Topic** | Canal de communication Kafka pour un type d'événement |
| **Consumer Group** | Groupe de consommateurs partageant la charge de lecture |
| **Avro** | Format de sérialisation binaire avec schéma |
| **Schema Registry** | Service de gestion des schémas Avro |
| **DLQ** | Dead Letter Queue — File pour les messages en erreur |
| **CQRS** | Command Query Responsibility Segregation |
| **Saga** | Patron de gestion des transactions distribuées |

---

**Document élaboré via entretien structuré (30 questions)**

**Prêt pour développement avec Claude Code**
