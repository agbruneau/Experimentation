# 8.1 Transactions Distribuées

## Résumé

Dans une architecture microservices, une opération métier peut impliquer plusieurs services. Les **transactions distribuées** coordonnent ces opérations pour maintenir la cohérence, même en cas de panne partielle.

### Points clés

- Les transactions ACID traditionnelles ne fonctionnent pas entre services
- Les patterns Saga et Outbox sont les solutions principales
- Le choix dépend du besoin de cohérence
- La compensation remplace le rollback classique

## Le problème

### Scénario : Souscription d'assurance

```
┌─────────────────────────────────────────────────────────────┐
│              SOUSCRIPTION MULTI-SERVICES                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Quote Engine     ──► Valider le devis                   │
│  2. Policy Admin     ──► Créer la police                    │
│  3. Billing          ──► Créer la facture                   │
│  4. Document Service ──► Générer les documents              │
│  5. Notification     ──► Envoyer confirmation               │
│                                                             │
│  Si l'étape 3 échoue, que faire des étapes 1-2 ?           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Pourquoi les transactions ACID ne marchent pas

| Aspect | Mono-service | Microservices |
|--------|--------------|---------------|
| Scope | Une seule DB | Plusieurs DBs |
| Rollback | Automatique | Impossible |
| Verrou | Court | Long (bloquant) |
| Disponibilité | OK | Dégradée par 2PC |

## Les approches possibles

### 1. Two-Phase Commit (2PC)

```
Coordinateur : "Préparez-vous à commit"
Service A : "Prêt"
Service B : "Prêt"
Service C : "Prêt"

Coordinateur : "Commit !"
Tous : "Fait"
```

**Problèmes** :
- Point unique de défaillance (coordinateur)
- Verrous maintenus pendant la coordination
- Mauvaise disponibilité
- Ne scale pas

### 2. Saga Pattern

```
                      SUCCÈS
    ──────────────────────────────────►
    T1 ──► T2 ──► T3 ──► T4 ──► T5

    ÉCHEC → COMPENSATION
    ◄──────────────────────────────────
    C1 ◄── C2 ◄── C3    T4 échoue
```

**Avantages** :
- Pas de verrous maintenus
- Chaque service gère sa DB
- Scalable
- Résilient

### 3. Outbox Pattern

```
┌────────────────────────────────────────┐
│           MÊME TRANSACTION              │
│                                        │
│  1. UPDATE policies SET ...            │
│  2. INSERT INTO outbox VALUES (...)    │
│  3. COMMIT                             │
│                                        │
└────────────────────────────────────────┘
            │
            │ Polling asynchrone
            ▼
     Publication garantie
```

**Avantages** :
- Atomicité garantie
- Pas de perte d'événement
- Découplage producteur/consommateur

## Cohérence éventuelle

### Définition

> Le système atteindra un état cohérent, mais pas immédiatement.

### Implications

```
T0: Commande reçue
T1: Service A modifié
T2: Service B modifié
T3: Service C modifié     ← Cohérent à partir d'ici

Entre T0 et T3 : État intermédiaire
Après T3 : État cohérent
```

### Gestion des états intermédiaires

```python
# État d'une souscription
status = "IN_PROGRESS"  # Pendant la saga
# -> "COMPLETED" si succès
# -> "CANCELLED" si compensation
```

## Recommandations

| Situation | Pattern recommandé |
|-----------|-------------------|
| Opération multi-services synchrone | Saga Orchestration |
| Publication d'événements fiable | Outbox |
| Intégration système legacy | 2PC (si nécessaire) |
| Opérations compensables | Saga Chorégraphie |

## Exemple complet : Souscription

```python
# 1. Démarrer la saga
saga = SubscriptionSaga()
result = await saga.execute({
    "quote_id": "Q001",
    "customer_id": "C001",
    "product": "AUTO"
})

# 2. Résultat possible: COMPLETED
{
    "status": "COMPLETED",
    "policy_id": "POL-001",
    "invoice_id": "INV-001",
    "documents": ["DOC-001", "DOC-002"]
}

# 3. Ou si échec à l'étape 3 (billing)
{
    "status": "COMPENSATED",
    "error": "Payment system unavailable",
    "failed_step": "create_invoice",
    "compensated_steps": ["create_policy"]
}
```

La police créée a été annulée automatiquement !
