# ADR 003: Choix de Go pour le backend

## Statut

Accepté

## Contexte

EDA-Lab nécessite un langage pour implémenter les microservices backend. Les candidats:

- Go
- Java (Spring Boot)
- Python (FastAPI)
- Node.js (TypeScript)
- Rust

## Décision

Nous avons choisi **Go** (Golang) version 1.21+.

## Justification

### Avantages

1. **Performance**: Compilation native, faible empreinte mémoire
2. **Concurrence**: Goroutines et channels natifs
3. **Simplicité**: Langage minimaliste, courbe d'apprentissage rapide
4. **Tooling**: go test, go fmt, go vet intégrés
5. **Conteneurisation**: Binaires statiques, images Docker légères
6. **Kafka client**: confluent-kafka-go mature et performant
7. **Écosystème**: chi, pgx, prometheus/client_golang

### Inconvénients

1. **Verbosité**: Gestion d'erreurs répétitive
2. **Généricité**: Support des génériques récent (1.18+)
3. **ORM**: Moins mature que Java/Python
4. **Réflexion**: Plus limitée que Java

### Alternatives rejetées

| Alternative | Raison du rejet |
|-------------|-----------------|
| Java | Plus lourd, temps de démarrage, JVM overhead |
| Python | Performance, GIL, typage dynamique |
| Node.js | Mono-thread, callback hell, moins adapté aux systèmes distribués |
| Rust | Courbe d'apprentissage, compile times |

## Conséquences

- Go modules pour la gestion des dépendances
- Workspace Go (`go.work`) pour le monorepo
- Structure de projet standard (`cmd/`, `internal/`, `pkg/`)
- Tests avec `go test` et testify

## Structure de projet

```
services/<service>/
├── cmd/<service>/main.go    # Point d'entrée
├── internal/                 # Code privé au service
│   ├── domain/              # Modèles métier
│   ├── repository/          # Accès données
│   ├── handler/             # Handlers d'événements
│   └── api/                 # API REST
├── go.mod                   # Dépendances
└── Dockerfile               # Build multi-stage
```

## Bibliothèques principales

| Bibliothèque | Usage |
|--------------|-------|
| chi | Routeur HTTP |
| pgx | Client PostgreSQL |
| confluent-kafka-go | Client Kafka |
| hamba/avro | Sérialisation Avro |
| prometheus/client_golang | Métriques |
| testify | Assertions de test |
