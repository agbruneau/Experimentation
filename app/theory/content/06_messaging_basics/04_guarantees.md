# 6.4 Garanties de Livraison

## Résumé

Les systèmes de messaging offrent différentes **garanties de livraison** qui déterminent si et combien de fois un message sera délivré. Comprendre ces garanties est essentiel pour concevoir des systèmes fiables.

### Points clés

- **At-most-once** : Peut perdre des messages (rapide)
- **At-least-once** : Pas de perte, mais doublons possibles
- **Exactly-once** : Idéal mais difficile à atteindre
- Le choix dépend des contraintes métier

## Les trois garanties

### At-Most-Once (Fire and Forget)

```
Producteur ──► Broker ──► Consommateur
    │                         │
    │     Pas de retry        │
    │                         │
    └── Si échec = Message perdu
```

**Caractéristiques** :
- Le producteur envoie sans attendre de confirmation
- Pas de retry en cas d'échec
- Meilleure performance (pas d'overhead)
- Perte de messages acceptable

**Cas d'usage** :
- Métriques temps réel (perdre quelques points n'est pas grave)
- Logs de debug
- Données IoT à haute fréquence

### At-Least-Once (Standard pour la plupart des cas)

```
Producteur ──► Broker ──► Consommateur
    │             │           │
    │◄── ACK ─────┤           │
    │             │           │
    │◄──────────────── ACK ───┤
    │                         │
    └── Si échec = Retry jusqu'à ACK
```

**Caractéristiques** :
- Le producteur attend un ACK du broker
- Le consommateur doit ACK après traitement
- Retry automatique en cas d'échec
- **Doublons possibles** si ACK perdu après traitement

**Cas d'usage** :
- Transactions financières
- Notifications importantes
- Événements métier critiques

### Exactly-Once (Le Graal)

```
Producteur                    Consommateur
    │                              │
    │ ─── Transaction ID ─────────►│
    │                              │
    │      Déduplication +         │
    │      Traitement atomique     │
    │                              │
    └── Garanti 1 et 1 seul traitement
```

**Caractéristiques** :
- Combinaison de at-least-once + déduplication
- Nécessite coordination (transactions distribuées)
- Overhead significatif
- Complexe à implémenter correctement

**Implémentation** :
- Transactions Kafka avec idempotence
- Outbox pattern + déduplication
- Two-phase commit (2PC)

## Acknowledgments (ACK)

### Côté Producteur

```python
# Sans ACK (at-most-once)
broker.send(message)  # Fire and forget

# Avec ACK (at-least-once)
result = await broker.send_and_wait(message)
if result.success:
    # Message confirmé reçu par broker
else:
    # Retry ou erreur
```

### Côté Consommateur

```python
async def process_message(message):
    try:
        # Traitement métier
        await handle_business_logic(message)

        # ACK manuel après succès
        await message.ack()
    except Exception as e:
        # NACK pour retry
        await message.nack()
```

### Stratégies d'ACK

| Stratégie | Description | Risque |
|-----------|-------------|--------|
| Auto-ACK | ACK dès réception | Perte si crash avant traitement |
| ACK après traitement | ACK après succès | Doublons si crash après traitement |
| ACK en batch | ACK par lots | Compromis performance/risque |

## Gestion des échecs

### Dead Letter Queue (DLQ)

```
Queue principale ──► Consommateur
                          │
                          │ Échec répété
                          ▼
                    Dead Letter Queue ──► Traitement manuel
```

**Configuration typique** :
- Retry 3 fois avec backoff exponentiel
- Après 3 échecs → DLQ
- Alerting sur DLQ non vide
- Traitement manuel ou automatisé des DLQ

### Retry avec Backoff

```python
async def process_with_retry(message, max_retries=3):
    for attempt in range(max_retries):
        try:
            await process(message)
            return  # Succès
        except RetryableError:
            # Backoff exponentiel : 1s, 2s, 4s
            delay = (2 ** attempt) * 1000
            await asyncio.sleep(delay / 1000)

    # Toutes les tentatives épuisées
    await send_to_dlq(message)
```

## Tableau comparatif

| Garantie | Perte | Doublons | Complexité | Performance |
|----------|-------|----------|------------|-------------|
| At-most-once | Oui | Non | Faible | Excellente |
| At-least-once | Non | Oui | Moyenne | Bonne |
| Exactly-once | Non | Non | Haute | Moyenne |

## Choix selon le cas d'usage Assurance

| Cas d'usage | Garantie recommandée | Justification |
|-------------|---------------------|---------------|
| Création de police | At-least-once | Critique, pas de perte acceptable |
| Métriques dashboard | At-most-once | Données temps réel, perte acceptable |
| Facturation | At-least-once + idempotence | Financier, doit gérer doublons |
| Audit trail | At-least-once | Traçabilité requise |

## Implémentation at-least-once robuste

```python
class AtLeastOnceProcessor:
    def __init__(self, broker, dlq_name):
        self.broker = broker
        self.dlq_name = dlq_name
        self.max_retries = 3

    async def process_queue(self, queue_name, handler):
        while True:
            message = await self.broker.receive(queue_name)

            for attempt in range(self.max_retries):
                try:
                    await handler(message.payload)
                    await message.ack()
                    break
                except RetryableError as e:
                    delay = (2 ** attempt)
                    await asyncio.sleep(delay)
                except FatalError as e:
                    await self.broker.send_to_dlq(
                        self.dlq_name,
                        message,
                        error=str(e)
                    )
                    await message.ack()
                    break
            else:
                # Max retries atteint
                await self.broker.send_to_dlq(
                    self.dlq_name,
                    message,
                    error="Max retries exceeded"
                )
                await message.ack()
```

## Points d'attention

1. **Toujours concevoir pour les doublons** avec at-least-once
2. **Monitorer les DLQ** - ne jamais les ignorer
3. **Logger les retries** pour diagnostiquer les problèmes
4. **Tester les scénarios de panne** (chaos engineering)
