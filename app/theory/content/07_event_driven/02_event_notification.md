# 7.2 Event Notification vs Event-Carried State Transfer

## Résumé

Il existe deux approches pour structurer le contenu d'un événement : **Event Notification** (signal léger) et **Event-Carried State Transfer** (données complètes). Chaque approche a ses avantages selon le contexte.

### Points clés

- Event Notification : signal minimal, les consommateurs doivent rappeler
- Event-Carried State Transfer : données complètes dans l'événement
- Le choix impacte le couplage et la performance
- On peut combiner les deux approches

## Event Notification

### Principe

L'événement contient juste l'information qu'il s'est passé quelque chose, pas les détails.

```
┌────────────┐     ┌─────────────────┐     ┌────────────┐
│  Service A │     │    Événement    │     │  Service B │
│            │────►│  "PolicyCreated"│────►│            │
│            │     │  policy_id: P001│     │            │
└────────────┘     └─────────────────┘     └──────┬─────┘
                                                  │
                          ◄──────────────────────┘
                          API call pour détails
```

### Structure

```json
{
  "event_type": "PolicyCreated",
  "policy_id": "POL-2024-001",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Avantages

| Avantage | Description |
|----------|-------------|
| **Messages légers** | Peu de bande passante |
| **Données fraîches** | Toujours l'état actuel |
| **Schéma simple** | Moins de maintenance |
| **Flexibilité** | Chaque consommateur prend ce dont il a besoin |

### Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Couplage temporel** | Le service source doit être disponible |
| **Charge API** | N consommateurs = N appels |
| **Latence** | Callback ajoute du délai |
| **Indisponibilité** | Échec si API source down |

### Cas d'usage appropriés

- Notifications où le détail n'est pas critique
- Quand les données changent fréquemment
- Pour déclencher une action qui ira chercher l'état complet
- Systèmes avec forte disponibilité des APIs

## Event-Carried State Transfer

### Principe

L'événement contient toutes les données nécessaires pour que les consommateurs travaillent de manière autonome.

```
┌────────────┐     ┌─────────────────────────┐     ┌────────────┐
│  Service A │     │      Événement          │     │  Service B │
│            │────►│  "PolicyCreated"        │────►│            │
│            │     │  policy_id: P001        │     │  (autonome)│
│            │     │  customer_id: C001      │     │            │
│            │     │  customer_name: Dupont  │     │            │
│            │     │  product: AUTO          │     │            │
│            │     │  premium: 850           │     │            │
└────────────┘     └─────────────────────────┘     └────────────┘
                   Pas de callback nécessaire
```

### Structure

```json
{
  "event_type": "PolicyCreated",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "policy_id": "POL-2024-001",
    "policy_number": "POL-2024-001",
    "customer": {
      "id": "C001",
      "name": "Jean Dupont",
      "email": "jean.dupont@email.com"
    },
    "product": "AUTO",
    "premium": 850.00,
    "coverages": ["RC", "VOL", "BRIS_GLACE"],
    "start_date": "2024-02-01",
    "end_date": "2025-01-31"
  }
}
```

### Avantages

| Avantage | Description |
|----------|-------------|
| **Autonomie** | Pas besoin de rappeler le service source |
| **Résilience** | Fonctionne même si source down |
| **Performance** | Pas d'appels supplémentaires |
| **Découplage** | Services indépendants |

### Inconvénients

| Inconvénient | Description |
|--------------|-------------|
| **Messages volumineux** | Plus de bande passante |
| **Données obsolètes** | Snapshot au moment de l'émission |
| **Schéma complexe** | Plus de maintenance |
| **Duplication** | Données copiées partout |

### Cas d'usage appropriés

- Quand l'autonomie des consommateurs est critique
- Systèmes distribués géographiquement
- Haute disponibilité requise
- Reporting et analytics

## Comparaison

| Critère | Notification | State Transfer |
|---------|--------------|----------------|
| Taille message | Petite | Grande |
| Fraîcheur données | Actuelle | Au moment de l'événement |
| Couplage temporel | Fort | Faible |
| Charge réseau | Événement + callbacks | Événement seul |
| Autonomie consommateur | Faible | Forte |
| Complexité schéma | Faible | Haute |

## Approche hybride

Dans la pratique, on combine souvent les deux :

```json
{
  "event_type": "PolicyCreated",
  "timestamp": "2024-01-15T10:30:00Z",

  "summary": {
    "policy_id": "POL-2024-001",
    "customer_id": "C001",
    "product": "AUTO",
    "premium": 850.00
  },

  "links": {
    "policy_details": "/api/policies/POL-2024-001",
    "customer": "/api/customers/C001"
  }
}
```

### Avantages de l'hybride

- Données essentielles disponibles immédiatement
- Liens pour les détails complets si nécessaire
- Flexibilité pour les consommateurs

## Exemple Assurance

### Scénario : Création de police

**Event Notification** :
```json
{"event": "PolicyCreated", "policy_id": "POL-001"}
```

→ Billing doit appeler l'API pour obtenir le montant de la prime

**Event-Carried State Transfer** :
```json
{
  "event": "PolicyCreated",
  "policy_id": "POL-001",
  "premium": 850,
  "customer_email": "jean@email.com"
}
```

→ Billing et Notifications ont tout ce qu'il faut

## Recommandations

| Contexte | Recommandation |
|----------|----------------|
| Notifications email/SMS | State Transfer (évite callbacks) |
| Déclencheur de workflow | Notification (workflow ira chercher) |
| Analytics/Reporting | State Transfer (autonomie) |
| UI temps réel | Notification (rafraîchir à la demande) |
| Facturation | State Transfer (données critiques) |
