# 6.1 Communication Synchrone vs Asynchrone

## Résumé

La communication entre services peut être **synchrone** (requête/réponse immédiate) ou **asynchrone** (découplée dans le temps). Le choix entre ces deux approches est fondamental pour l'architecture de votre système.

### Points clés

- **Synchrone** : L'appelant attend la réponse avant de continuer
- **Asynchrone** : L'appelant continue sans attendre la réponse
- Chaque approche a ses cas d'usage appropriés
- Il est courant de combiner les deux dans un même système

## Communication Synchrone

### Caractéristiques

```
Client ──────► Service
   │              │
   │   Attente    │
   │              │
   ◄──────────────┘
       Réponse
```

| Aspect | Description |
|--------|-------------|
| **Couplage temporel** | Fort - les deux parties doivent être disponibles |
| **Latence** | Perçue directement par l'appelant |
| **Complexité** | Simple à implémenter et déboguer |
| **Résilience** | Faible - une panne bloque la chaîne |

### Cas d'usage appropriés

- **Lecture de données** : Obtenir l'état actuel d'une ressource
- **Validation immédiate** : Vérifier une règle métier avant de continuer
- **Transactions courtes** : Opérations qui doivent réussir ou échouer immédiatement
- **Interface utilisateur** : Réponse directe à une action utilisateur

### Exemple Assurance

```
# Vérification de la validité d'une police
GET /api/policies/POL-2024-001/status

Réponse immédiate: { "status": "ACTIVE", "valid_until": "2024-12-31" }
```

## Communication Asynchrone

### Caractéristiques

```
Producteur ──────► Message Broker
                        │
                        │ (découplé)
                        │
                        ▼
                   Consommateur
```

| Aspect | Description |
|--------|-------------|
| **Couplage temporel** | Faible - découplage dans le temps |
| **Latence** | Absorbée par le système de messages |
| **Complexité** | Plus élevée (idempotence, ordre, etc.) |
| **Résilience** | Forte - les messages persistent |

### Cas d'usage appropriés

- **Notifications** : Informer sans attendre de réponse
- **Traitements longs** : Opérations qui prennent du temps
- **Découplage** : Quand les systèmes évoluent indépendamment
- **Pics de charge** : Lisser la charge via des files d'attente

### Exemple Assurance

```
# Création d'une police avec notification asynchrone
POST /api/policies
  └─► Événement publié: PolicyCreated
        │
        ├─► Billing (crée facture)
        ├─► Notifications (envoie email)
        └─► Audit (enregistre trace)
```

## Critères de Choix

### Choisir Synchrone quand...

| Critère | Explication |
|---------|-------------|
| Réponse immédiate requise | L'utilisateur attend une confirmation |
| Données fraîches nécessaires | Lecture de l'état actuel |
| Transaction atomique | Tout réussit ou tout échoue |
| Chaîne d'appels courte | Peu d'intermédiaires |

### Choisir Asynchrone quand...

| Critère | Explication |
|---------|-------------|
| Fire-and-forget | Pas besoin de réponse |
| Plusieurs consommateurs | Un événement déclenche plusieurs actions |
| Résilience critique | Le système doit continuer malgré les pannes |
| Traitement différé acceptable | La latence n'est pas critique |

## Pattern Request-Reply Asynchrone

Un hybride combinant les avantages des deux approches :

```
Client ──► Queue Requêtes ──► Service
                                 │
Client ◄── Queue Réponses ◄──────┘
```

### Avantages

- Découplage tout en gardant la sémantique requête/réponse
- Résilience améliorée (messages persistants)
- Load balancing naturel

### Inconvénients

- Complexité accrue
- Latence potentiellement plus élevée
- Gestion des timeouts plus complexe

## Sandbox : Scénario EVT-01

Dans ce scénario, vous allez comparer les deux approches en publiant un événement `PolicyCreated` et observer comment les différents services réagissent de manière asynchrone.

**Objectif** : Comprendre la différence entre un appel REST synchrone et une publication d'événement.
