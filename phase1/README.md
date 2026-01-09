# Phase 1 : Communication Événementielle avec Kafka

**Complexité :** ⭐⭐ Simple | **Temps setup :** ~15 minutes

La Phase 1 ajoute **Apache Kafka** pour découpler les agents temporellement et spatialement. Les agents communiquent maintenant via des événements asynchrones au lieu d'appels directs.

---

## 🎯 Objectif

Comprendre l'architecture événementielle :

- Communication asynchrone via Kafka
- Producers et Consumers
- Découplage temporel et spatial
- Scalabilité horizontale

---

## 📋 Prérequis

- Docker & Docker Compose
- Python 3.10+
- Clé API Anthropic
- Phase 0 maîtrisée (recommandé)

---

## ⚡ Installation Rapide

```bash
# 1. Démarrer Kafka (mode KRaft, sans Zookeeper)
docker-compose up -d

# Attendre que Kafka soit prêt (~30 secondes)
# Vérifier: docker-compose logs kafka | grep "started"

# 2. Installer les dépendances Python
pip install -r requirements.txt

# 3. Créer les topics Kafka
python scripts/init_kafka.py

# 4. Lancer les agents (dans des terminaux séparés)
# Terminal 1
python src/agents/intake_agent/main.py

# Terminal 2
python src/agents/risk_agent/main.py

# Terminal 3
python src/agents/decision_agent/main.py
```

---

## ⚙️ Configuration

La Phase 1 utilise le fichier `config.yaml` à la racine du projet pour centraliser toute la configuration. Les valeurs peuvent être **surchargées** via des variables d'environnement.

### Fichier config.yaml

```yaml
# config.yaml - Configuration centralisée
agents:
  risk_agent:
    model: "claude-sonnet-4-20250514" # Modèle LLM pour analyse de risque
    temperature: 0.2 # Créativité (0.0 = déterministe)
    consumer_group: "agent-risk-analyst"

  decision_agent:
    model: "claude-3-5-sonnet-20241022" # Modèle LLM pour décision finale
    temperature: 0.1 # Plus conservateur pour décisions
    consumer_group: "agent-loan-officer"

  intake_agent:
    model: "claude-3-5-haiku-20241022" # Modèle rapide pour validation
    temperature: 0.0 # Totalement déterministe

# Seuils de décision automatique
thresholds:
  auto_approve_score: 20 # Score < 20 = approbation auto
  auto_reject_score: 80 # Score > 80 = rejet auto
  high_value_amount: 100000 # Montant nécessitant revue humaine

# Configuration Kafka
kafka:
  bootstrap_servers: "localhost:9092"
  topics:
    applications: "finance.loan.application.v1"
    risk_results: "risk.scoring.result.v1"
    decisions: "finance.loan.decision.v1"
```

### Override via Variables d'Environnement

Utilisez le pattern `SECTION__KEY__SUBKEY` pour surcharger n'importe quelle valeur :

```bash
# Changer le modèle du Risk Agent
export AGENTS__RISK_AGENT__MODEL=claude-3-5-haiku-20241022

# Ajuster la température pour plus de créativité
export AGENTS__DECISION_AGENT__TEMPERATURE=0.3

# Modifier les seuils de décision
export THRESHOLDS__AUTO_APPROVE_SCORE=15
export THRESHOLDS__AUTO_REJECT_SCORE=85

# Configuration Kafka pour environnement distant
export KAFKA__BOOTSTRAP_SERVERS=kafka.production.example.com:9092
```

### Exemples d'Usage

**Développement local avec modèle économique :**

```bash
# Utiliser Haiku (moins cher) pour tous les agents en dev
AGENTS__RISK_AGENT__MODEL=claude-3-5-haiku-20241022 \
AGENTS__DECISION_AGENT__MODEL=claude-3-5-haiku-20241022 \
python src/agents/risk_agent/main.py
```

**Production avec seuils stricts :**

```bash
# Seuils plus conservateurs pour la production
THRESHOLDS__AUTO_APPROVE_SCORE=10 \
THRESHOLDS__AUTO_REJECT_SCORE=90 \
python src/agents/decision_agent/main.py
```

**Fichier .env pour persistance :**

```bash
# .env - Variables pour votre environnement
ANTHROPIC_API_KEY=sk-ant-your-key-here
AGENTS__RISK_AGENT__MODEL=claude-sonnet-4-20250514
KAFKA__BOOTSTRAP_SERVERS=localhost:9092
```

### Priorité de Configuration

1. **Variables d'environnement** (priorité haute)
2. **config.yaml** (priorité basse)

> 💡 **Tip:** Utilisez `python -c "from src.shared.config_loader import load_config; print(load_config())"` pour afficher la configuration résolue.

## 📂 Structure

```
phase1/
├── README.md              # Ce fichier
├── docker-compose.yml     # Kafka KRaft uniquement
├── requirements.txt       # + confluent-kafka
├── scripts/
│   └── init_kafka.py     # Création des topics
└── src/
    ├── agents/           # Agents adaptés pour Kafka
    └── shared/
        ├── kafka_client.py  # Wrappers Producer/Consumer
        ├── models.py       # Modèles de données
        └── prompts.py      # System prompts
```

---

## 🔍 Différences avec Phase 0

| Aspect         | Phase 0        | Phase 1             |
| -------------- | -------------- | ------------------- |
| Communication  | Appels directs | Événements Kafka    |
| Déploiement    | Script unique  | 3 processus séparés |
| Infrastructure | Aucune         | Kafka Docker        |
| Scalabilité    | Limitée        | Horizontale         |
| Découplage     | Temporel       | Temporel + Spatial  |

---

## 🏗️ Architecture Kafka

### Topics

- `finance.loan.application.v1` : Demandes validées (Intake → Risk)
- `risk.scoring.result.v1` : Évaluations de risque (Risk → Decision)
- `finance.loan.decision.v1` : Décisions finales (Decision → External)

### Flux de Données

```
[Intake Agent] --produit--> [finance.loan.application.v1]
                                    |
                                    | consomme
                                    v
[Risk Agent] --produit--> [risk.scoring.result.v1]
                                |
                                | consomme
                                v
[Decision Agent] --produit--> [finance.loan.decision.v1]
```

---

## 🧪 Tester le Système

### 1. Envoyer une demande

Dans un terminal séparé :

```bash
python scripts/send_test_request.py
```

### 2. Observer les logs

Les agents affichent les messages qu'ils consomment et produisent.

### 3. Vérifier les topics

```bash
# Lister les topics
docker exec agentmesh-kafka kafka-topics --bootstrap-server localhost:9092 --list

# Consulter les messages d'un topic
docker exec agentmesh-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic finance.loan.application.v1 \
  --from-beginning
```

---

## 🐛 Dépannage

**Kafka ne démarre pas**

- Vérifiez les logs: `docker-compose logs kafka`
- Assurez-vous que le port 9092 n'est pas utilisé

**Agents ne reçoivent pas de messages**

- Vérifiez que les topics existent: `python scripts/init_kafka.py`
- Vérifiez les logs des agents pour les erreurs de connexion

**Messages dupliqués**

- Normal si vous relancez les agents (auto.offset.reset=earliest)
- Pour repartir de zéro: supprimez les topics et recréez-les

---

## 📚 Prochaines étapes

Une fois que vous maîtrisez la Phase 1 :

1. **Phase 2** : Ajouter RAG avec ChromaDB

   - Consultez [../PHASES.md](../PHASES.md)
   - Naviguez vers `phase2/`

2. **Explorer les notebooks** :
   - `../notebooks/03-kafka-flow.ipynb`

---

## 💡 Concepts Clés

- **Producer** : Publie des événements dans un topic
- **Consumer** : Lit des événements depuis un topic
- **Consumer Group** : Permet la parallélisation (plusieurs instances)
- **Offset** : Position de lecture dans le topic

---

**Besoin d'aide ?** Consultez [../PHASES.md](../PHASES.md) ou les logs Docker.
