# 6.2 Message Queue (Point-à-Point)

## Résumé

Une **Message Queue** est un mécanisme de communication point-à-point où chaque message est consommé par **un seul destinataire**. C'est le pattern fondamental pour le traitement asynchrone garanti.

### Points clés

- Un producteur envoie des messages dans la queue
- Un seul consommateur traite chaque message
- Les messages sont persistants et ordonnés (FIFO)
- Idéal pour le traitement de tâches en arrière-plan

## Fonctionnement

```
              ┌──────────────────────────┐
              │      MESSAGE QUEUE       │
              │                          │
Producteur ──►│ [Msg1] [Msg2] [Msg3] ──►│──► Consommateur
              │                          │
              └──────────────────────────┘
                      FIFO
```

### Cycle de vie d'un message

1. **Envoi** : Le producteur place le message dans la queue
2. **Stockage** : Le message est persisté (durable)
3. **Livraison** : Le broker délivre au consommateur
4. **Traitement** : Le consommateur traite le message
5. **Acquittement** : Le consommateur confirme le traitement
6. **Suppression** : Le broker supprime le message acquitté

## Competing Consumers

Plusieurs consommateurs peuvent écouter la même queue pour paralléliser le traitement :

```
              ┌──────────────────────────┐
              │      MESSAGE QUEUE       │
              │                          │     ┌─────────────┐
Producteur ──►│ [Msg1] [Msg2] [Msg3] ──►│────►│Consommateur1│
              │                          │     └─────────────┘
              │                          │     ┌─────────────┐
              │                      ──►│────►│Consommateur2│
              │                          │     └─────────────┘
              └──────────────────────────┘
```

### Avantages

- **Scalabilité** : Ajoutez des consommateurs pour traiter plus vite
- **Load balancing** : Distribution automatique de la charge
- **Résilience** : Si un consommateur tombe, les autres prennent le relais

### Attention

- L'ordre de traitement n'est plus garanti globalement
- Un message n'est traité que par UN consommateur

## Cas d'usage en Assurance

### Traitement des réclamations

```
┌────────────┐     ┌─────────────────┐     ┌────────────────┐
│  Portail   │────►│ queue.claims    │────►│ Claims Workers │
│  Client    │     │                 │     │  (x3 instances)│
└────────────┘     └─────────────────┘     └────────────────┘
```

Avantages :
- Les clients n'attendent pas le traitement complet
- Les pics de charge sont absorbés par la queue
- Le traitement peut être parallélisé

### Génération de documents

```
┌────────────┐     ┌─────────────────┐     ┌────────────────┐
│  Policy    │────►│ queue.documents │────►│ Document Gen   │
│  Service   │     │                 │     │                │
└────────────┘     └─────────────────┘     └────────────────┘
```

## Garanties de livraison

### At-Most-Once

- Le message est délivré au plus une fois
- Possible perte de messages
- Plus rapide (pas d'acquittement)

### At-Least-Once

- Le message est délivré au moins une fois
- Pas de perte, mais possibles doublons
- Nécessite l'**idempotence** du consommateur

### Exactly-Once (difficile)

- Le message est délivré exactement une fois
- Complexe à implémenter (transactions distribuées)
- Souvent simulé avec at-least-once + déduplication

## Implémentation pseudo-code

```python
# Producteur
async def submit_claim(claim_data):
    # Enregistrer en base
    claim = await db.insert_claim(claim_data)

    # Envoyer dans la queue pour traitement
    await broker.send_to_queue(
        "claims.processing",
        {
            "claim_id": claim.id,
            "type": claim.type,
            "timestamp": now()
        }
    )

    return {"status": "submitted", "claim_id": claim.id}

# Consommateur
async def process_claims():
    while True:
        message = await broker.receive_from_queue("claims.processing")

        try:
            claim = await db.get_claim(message["claim_id"])
            await evaluate_claim(claim)
            await notify_customer(claim)

            # Acquitter le message
            await message.ack()
        except Exception as e:
            # Rejeter pour retry ou DLQ
            await message.nack()
```

## Bonnes pratiques

| Pratique | Raison |
|----------|--------|
| Messages petits | Éviter les timeouts et problèmes mémoire |
| Idempotence | Gérer les retraitements sans effet de bord |
| Timeouts appropriés | Éviter les blocages |
| Monitoring | Surveiller la taille des queues |
| Dead Letter Queue | Isoler les messages problématiques |

## Sandbox : Scénario EVT-02

Dans ce scénario, vous allez créer une queue pour le traitement des réclamations et observer le pattern Competing Consumers en action.

**Objectif** : Implémenter un traitement de queue avec plusieurs consommateurs.
