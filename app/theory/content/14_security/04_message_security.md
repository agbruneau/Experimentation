# Sécurité des Messages et Événements

## Résumé

Dans une architecture événementielle, les messages transitent par des brokers et peuvent être consommés par de multiples services. La sécurité des messages inclut l'authentification des producteurs/consommateurs, le chiffrement des payloads et la validation des données.

## Points clés

- **Authentification** : Qui peut publier/consommer ?
- **Autorisation** : Quels topics/queues accessibles ?
- **Chiffrement** : Protection du contenu des messages
- **Validation** : Schémas et intégrité des données

## Authentification sur le broker

### Mécanismes courants

| Mécanisme | Description | Usage |
|-----------|-------------|-------|
| SASL/PLAIN | Username/password | Simple, interne |
| SASL/SCRAM | Challenge-response | Plus sécurisé |
| mTLS | Certificats | Production |
| OAuth | Tokens JWT | Cloud, fédération |

### Configuration type

```yaml
# Producteur
broker:
  bootstrap_servers: "kafka:9092"
  security_protocol: "SASL_SSL"
  sasl_mechanism: "SCRAM-SHA-256"
  sasl_username: "quote-engine"
  sasl_password: "${KAFKA_PASSWORD}"
  ssl_ca_location: "/certs/ca.pem"
```

## Autorisation par topic

### ACLs (Access Control Lists)

```
# quote-engine peut publier sur policies.*
User:quote-engine → ALLOW → WRITE → Topic:policies.*

# billing peut consommer policies.created
User:billing → ALLOW → READ → Topic:policies.created

# claims ne peut pas accéder aux topics billing
User:claims → DENY → ALL → Topic:billing.*
```

### Schéma de permissions

```
                    Broker
                      │
        ┌─────────────┼─────────────┐
        │             │             │
   Topic:policies  Topic:claims  Topic:billing
        │             │             │
        │             │             │
┌───────┴───────┐     │       ┌─────┴─────┐
│   Producteurs │     │       │Consommateurs│
│ quote-engine  │     │       │  billing   │
│ policy-admin  │     │       │  notif     │
└───────────────┘     │       └────────────┘
                      │
              ┌───────┴───────┐
              │   Producteurs │
              │ claims-mgmt   │
              │               │
              │ Consommateurs │
              │ policy-admin  │
              │ billing       │
              └───────────────┘
```

## Chiffrement des messages

### Chiffrement en transit

```
Producer ═══TLS═══▶ Broker ═══TLS═══▶ Consumer

Le broker voit le contenu en clair.
Protection contre l'interception réseau.
```

### Chiffrement End-to-End (E2E)

```
Producer                 Broker                Consumer
    │                      │                      │
    │ encrypt(payload)     │                      │
    │────────────────────▶ │                      │
    │                      │ message chiffré      │
    │                      │─────────────────────▶│
    │                      │                      │ decrypt(payload)
```

Le broker ne peut pas lire le contenu.

### Implémentation E2E

```python
class SecureProducer:
    def __init__(self, producer, encryption_key):
        self.producer = producer
        self.cipher = Fernet(encryption_key)

    async def publish(self, topic, event):
        # Sérialiser
        payload = json.dumps(event).encode()

        # Chiffrer
        encrypted = self.cipher.encrypt(payload)

        # Publier
        await self.producer.publish(topic, encrypted)


class SecureConsumer:
    def __init__(self, consumer, decryption_key):
        self.consumer = consumer
        self.cipher = Fernet(decryption_key)

    async def consume(self, topic):
        async for message in self.consumer.consume(topic):
            # Déchiffrer
            decrypted = self.cipher.decrypt(message.value)

            # Désérialiser
            event = json.loads(decrypted.decode())
            yield event
```

## Validation des messages

### Schéma de validation

```json
// Schema: PolicyCreated
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["event_id", "policy_number", "customer_id", "created_at"],
  "properties": {
    "event_id": {
      "type": "string",
      "format": "uuid"
    },
    "policy_number": {
      "type": "string",
      "pattern": "^POL-[0-9]{4}-[0-9]{6}$"
    },
    "customer_id": {
      "type": "string"
    },
    "premium": {
      "type": "number",
      "minimum": 0
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    }
  }
}
```

### Validation à la production

```python
class ValidatingProducer:
    def __init__(self, producer, schema_registry):
        self.producer = producer
        self.registry = schema_registry

    async def publish(self, topic, event):
        # Récupérer le schéma
        schema = await self.registry.get_schema(topic)

        # Valider
        try:
            validate(event, schema)
        except ValidationError as e:
            raise InvalidEventError(f"Event validation failed: {e}")

        # Publier
        await self.producer.publish(topic, event)
```

### Validation à la consommation

```python
async def consume_with_validation(topic, handler):
    async for message in consumer.consume(topic):
        try:
            # Valider le schéma
            schema = await schema_registry.get_schema(topic)
            validate(message.value, schema)

            # Traiter
            await handler(message.value)

        except ValidationError as e:
            # Message malformé → DLQ
            log.error("Invalid message", error=str(e))
            await dlq.send(message)
```

## Signature des messages

### Pour garantir l'intégrité et l'origine

```python
import hmac
import hashlib

def sign_message(message, secret_key):
    """Signe un message avec HMAC-SHA256."""
    payload = json.dumps(message).encode()
    signature = hmac.new(
        secret_key.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    return {
        "payload": message,
        "signature": signature,
        "signed_at": datetime.now().isoformat()
    }

def verify_signature(signed_message, secret_key):
    """Vérifie la signature d'un message."""
    payload = json.dumps(signed_message["payload"]).encode()
    expected = hmac.new(
        secret_key.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(
        signed_message["signature"],
        expected
    )
```

## Cas d'usage assurance

### Événement PolicyCreated sécurisé

```python
# Production
async def publish_policy_created(policy):
    event = {
        "event_id": str(uuid4()),
        "event_type": "PolicyCreated",
        "policy_number": policy.number,
        "customer_id": policy.customer_id,
        "premium": policy.premium,
        "created_at": datetime.now().isoformat(),
        # Données sensibles chiffrées
        "customer_details_encrypted": encrypt(
            json.dumps({
                "name": policy.customer_name,
                "email": policy.customer_email
            })
        )
    }

    # Signer
    signed_event = sign_message(event, SIGNING_KEY)

    # Publier
    await producer.publish("policies.created", signed_event)


# Consommation
async def handle_policy_created(signed_event):
    # Vérifier la signature
    if not verify_signature(signed_event, SIGNING_KEY):
        raise SecurityError("Invalid signature")

    event = signed_event["payload"]

    # Valider le schéma
    validate(event, POLICY_CREATED_SCHEMA)

    # Déchiffrer si autorisé
    if has_permission("customer:read"):
        customer_details = json.loads(
            decrypt(event["customer_details_encrypted"])
        )
    else:
        customer_details = None

    # Traiter
    await process_policy_created(event, customer_details)
```

### Isolation des topics par sensibilité

```
Topics publics (données agrégées):
- analytics.quotes.daily
- analytics.claims.summary

Topics internes (données opérationnelles):
- policies.created
- claims.submitted

Topics sensibles (données PII):
- customers.updated [chiffrement E2E]
- payments.processed [chiffrement E2E]
```

## Bonnes pratiques

1. **Authentification obligatoire** sur le broker
2. **ACLs granulaires** par service et topic
3. **Chiffrement E2E** pour les données sensibles
4. **Validation de schéma** à la production ET consommation
5. **Signature** pour les événements critiques
6. **Audit** des accès aux topics sensibles

## Anti-patterns

1. **Broker sans authentification** : Tout le monde peut publier
2. **Pas d'ACLs** : Tout le monde voit tout
3. **Données sensibles en clair** : Visibles si broker compromis
4. **Pas de validation** : Messages malformés causent des erreurs
5. **Signature facultative** : Messages falsifiables
