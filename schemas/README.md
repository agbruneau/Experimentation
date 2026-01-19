# Schémas Avro - EDA-Lab

Ce répertoire contient les schémas Avro pour tous les événements du système EDA-Lab.

## Structure

```
schemas/
├── bancaire/           # Domaine bancaire
│   ├── compte-ouvert.avsc
│   ├── compte-ferme.avsc
│   ├── depot-effectue.avsc
│   ├── retrait-effectue.avsc
│   ├── virement-emis.avsc
│   ├── virement-recu.avsc
│   └── paiement-prime-effectue.avsc
└── README.md
```

## Convention de nommage

- **Fichiers**: `<entité>-<action>.avsc` (kebab-case)
- **Sujets Schema Registry**: `<domaine>.<entité>.<action>-value`
- **Namespace**: `com.edalab.<domaine>.events`

## Types communs

### TypeCompte (enum)
- `COURANT` - Compte courant
- `EPARGNE` - Compte épargne
- `JOINT` - Compte joint

### Canal (enum)
- `GUICHET` - Opération au guichet
- `VIREMENT` - Par virement bancaire
- `CHEQUE` - Par chèque
- `CARTE` - Par carte bancaire

### StatutVirement (enum)
- `INITIE` - Virement initié
- `EN_COURS` - En cours de traitement
- `COMPLETE` - Virement complété
- `REJETE` - Virement rejeté

## Enregistrement des schémas

Pour enregistrer les schémas dans le Schema Registry :

```bash
./scripts/register-schemas.sh
```

Variables d'environnement :
- `SCHEMA_REGISTRY_URL` - URL du Schema Registry (défaut: http://localhost:8081)
- `SCHEMAS_DIR` - Répertoire des schémas (défaut: ./schemas)
