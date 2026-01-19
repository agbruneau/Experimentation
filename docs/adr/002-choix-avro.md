# ADR 002: Choix d'Apache Avro pour la sérialisation

## Statut

Accepté

## Contexte

Les événements circulant dans Kafka doivent être sérialisés. Les formats considérés:

- Apache Avro
- Protocol Buffers (Protobuf)
- JSON Schema
- MessagePack
- JSON brut

## Décision

Nous avons choisi **Apache Avro** avec Confluent Schema Registry.

## Justification

### Avantages

1. **Schema Registry**: Gouvernance centralisée des schémas
2. **Évolution de schémas**: Compatibilité forward/backward automatique
3. **Compacité**: Sérialisation binaire efficace
4. **Intégration Confluent**: Support natif dans l'écosystème
5. **Validation automatique**: Messages invalides rejetés
6. **Documentation intégrée**: Les schémas documentent les événements

### Inconvénients

1. **Complexité**: Plus complexe que JSON brut
2. **Tooling**: Moins d'outils que Protobuf
3. **Lisibilité**: Format binaire non lisible directement

### Alternatives rejetées

| Alternative | Raison du rejet |
|-------------|-----------------|
| Protobuf | Moins bien intégré à l'écosystème Confluent |
| JSON Schema | Moins performant, pas de format binaire |
| JSON brut | Pas de validation, pas d'évolution contrôlée |
| MessagePack | Pas de schema registry |

## Conséquences

- Schémas Avro dans `schemas/<domaine>/`
- Génération de types Go depuis les schémas
- Configuration du Schema Registry dans Docker Compose
- Sérialisation/désérialisation avec hamba/avro

## Exemple de schéma

```json
{
  "type": "record",
  "name": "CompteOuvert",
  "namespace": "com.edalab.bancaire.events",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "compte_id", "type": "string"}
  ]
}
```

## Règles d'évolution

1. Ajouter des champs avec valeur par défaut
2. Ne jamais supprimer de champs obligatoires
3. Utiliser les types `union` pour les champs optionnels
4. Versionner les schémas (v1, v2, etc.)
