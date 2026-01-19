# ADR 001: Choix d'Apache Kafka comme Message Broker

## Statut

Accepté

## Contexte

EDA-Lab nécessite un message broker pour implémenter les patterns d'architecture événementielle. Les candidats considérés étaient:

- Apache Kafka
- RabbitMQ
- Apache Pulsar
- NATS
- Amazon SQS/SNS

## Décision

Nous avons choisi **Apache Kafka** (Confluent Platform Community Edition) comme message broker.

## Justification

### Avantages

1. **Écosystème riche**: Confluent Platform offre Schema Registry, Kafka Connect, ksqlDB
2. **Persistance des messages**: Log distribué permettant le replay
3. **Haute performance**: Throughput élevé avec faible latence
4. **Partitionnement**: Scalabilité horizontale native
5. **Consumer Groups**: Parallélisation du traitement
6. **Mode KRaft**: Élimine la dépendance ZooKeeper (simplification)
7. **Adoption industrielle**: Standard de facto pour l'event streaming

### Inconvénients

1. **Complexité opérationnelle**: Plus complexe que RabbitMQ
2. **Ressources**: Consommation mémoire plus élevée
3. **Courbe d'apprentissage**: Concepts spécifiques (offsets, partitions)

### Alternatives rejetées

| Alternative | Raison du rejet |
|-------------|-----------------|
| RabbitMQ | Moins adapté à l'event sourcing, pas de replay natif |
| Apache Pulsar | Moins mature, communauté plus petite |
| NATS | Simplicité mais fonctionnalités limitées pour EDA |
| Amazon SQS | Vendor lock-in, coût, pas adapté à un lab local |

## Conséquences

- Configuration Docker Compose avec Confluent Platform
- Utilisation de confluent-kafka-go comme client Go
- Intégration avec Schema Registry pour Avro
- Topics avec rétention configurable pour le replay

## Notes

Pour le MVP, nous utilisons un seul broker en mode KRaft. En production, un cluster de 3+ brokers serait recommandé.
