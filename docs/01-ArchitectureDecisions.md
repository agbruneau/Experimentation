# Architecture Decision Records (ADRs) - Projet AgentMeshKafka

> **Version :** 1.1.0 | **Statut :** Approuvé | **Dernière révision :** Janvier 2026

Ce document consigne les **décisions architecturales structurantes** pour le projet de Maillage Agentique. Chaque décision suit le format standard **ADR (Architecture Decision Record)** et explique le contexte, le choix effectué et ses conséquences (positives et négatives).

> 💡 **Pourquoi ce document ?** Il démontre la capacité à justifier des choix techniques complexes en les reliant aux contraintes du système (Découplage, Résilience, Non-déterminisme).

## Table des Matières

- [ADR-001 : Adoption d'une Architecture Événementielle (Event-Driven) via Kafka](#adr-001--adoption-dune-architecture-événementielle-event-driven-via-kafka)
- [ADR-002 : Gouvernance des Données via Avro et Schema Registry](#adr-002--gouvernance-des-données-via-avro-et-schema-registry)
- [ADR-003 : Architecture Cognitive des Agents (Pattern ReAct)](#adr-003--architecture-cognitive-des-agents-pattern-react)
- [ADR-004 : Stratégie de Résilience (Event Sourcing & Outbox)](#adr-004--stratégie-de-résilience-event-sourcing--outbox)
- [ADR-005 : Cadre d'Évaluation Agentique (Le Diamant)](#adr-005--cadre-dévaluation-agentique-le-diamant)
- [ADR-006 : Configuration Externalisée (YAML + Environment Override)](#adr-006--configuration-externalisée-yaml--environment-override)
- [ADR-007 : Stratégie de Sélection des Modèles LLM par Agent](#adr-007--stratégie-de-sélection-des-modèles-llm-par-agent)

---

**Documents connexes :** [02-DataContracts.md](./02-DataContracts.md) | [03-AgentSpecs.md](./03-AgentSpecs.md) | [04-EvaluationStrategie.md](./04-EvaluationStrategie.md)

---

## ADR-001 : Adoption d'une Architecture Événementielle (Event-Driven) via Kafka

- **Statut :** Accepté
- **Date :** 2026-01-09
- **Contexte :**
  Dans une architecture traditionnelle orientée services (microservices), la communication se fait souvent via HTTP (REST/gRPC). Pour des agents autonomes IA, ce couplage synchrone pose problème : la latence des LLM est élevée et imprévisible, et un agent indisponible ne doit pas bloquer toute la chaîne.
- **Décision :**
  Nous utiliserons **Apache Kafka** comme épine dorsale (backbone) de communication asynchrone. Les agents ne s'appellent jamais directement.
- **Producteurs :** Les agents publient leurs résultats (faits) dans des _Topics_.
- **Consommateurs :** Les agents s'abonnent aux topics pertinents pour réagir aux événements.

- **Conséquences :**
- ✅ **Découplage Temporel :** Un agent peut être hors ligne, les messages l'attendront.
- ✅ **Scalabilité :** Possibilité d'ajouter plusieurs instances d'un même agent (Consumer Group) pour paralléliser le traitement.
- ✅ **Observabilité :** Le journal (Log) Kafka sert de source de vérité immuable pour déboguer les décisions des agents.
- ⚠️ **Complexité :** Nécessite la gestion d'un cluster Kafka et Zookeeper (ou KRaft).

---

## ADR-002 : Gouvernance des Données via Avro et Schema Registry

- **Statut :** Accepté
- **Contexte :**
  Les agents IA sont par nature non déterministes et peuvent produire des sorties variables. Si un agent en amont change le format de ses données (ex: renomme un champ JSON), cela peut briser les agents en aval ("Schema Drift").
- **Décision :**
  Nous imposons l'utilisation de **Apache Avro** pour la sérialisation et d'un **Schema Registry** pour valider les messages avant publication.
- Tout événement doit correspondre à un schéma `.avsc` validé.
- La politique de compatibilité sera réglée sur `FORWARD` (les anciennes données peuvent être lues par les nouveaux schémas).

- **Conséquences :**
- ✅ **Contrats Explicites :** Les agents ont une interface claire et typée.
- ✅ **Prévention d'Erreurs :** Un agent "halluciné" produisant un JSON malformé sera bloqué au niveau du Producer, protégeant le reste du système.
- ⚠️ **Overhead :** Nécessite une étape de compilation/validation des schémas avant le déploiement.

---

## ADR-003 : Architecture Cognitive des Agents (Pattern ReAct)

- **Statut :** Accepté
- **Contexte :**
  Un agent ne doit pas seulement "parler", il doit "agir". Un simple appel LLM (Zero-shot) est insuffisant pour des tâches complexes nécessitant des calculs ou des vérifications externes.
- **Décision :**
  Chaque agent implémentera le pattern **ReAct (Reason + Act)**.

1. **Thought :** L'agent analyse la situation.
2. **Action :** L'agent sélectionne un outil (Tool Use) parmi une liste définie (ex: `calculer_score`, `chercher_base_vectorielle`).
3. **Observation :** L'agent reçoit le résultat de l'outil.
4. **Final Answer :** L'agent synthétise la réponse.
   Le framework technique retenu pour orchestrer cette boucle est **LangChain / LangGraph**.

- **Conséquences :**
- ✅ **Capacité d'Action :** Permet aux agents d'interagir avec le monde réel (API, DB).
- ✅ **Explicabilité :** La chaîne de pensée (Chain of Thought) est enregistrée, permettant de comprendre _pourquoi_ une décision a été prise.
- ⚠️ **Coût et Latence :** Augmente le nombre de tokens et le temps de réponse global.

---

## ADR-004 : Stratégie de Résilience (Event Sourcing & Outbox)

- **Statut :** Accepté
- **Contexte :**
  L'état interne d'un agent (sa "mémoire") doit être cohérent avec les événements qu'il publie. Le risque de "Dual Write" (écrire en DB mais échouer à publier dans Kafka) est critique.
- **Décision :**

1. **Event Sourcing :** L'état de l'agent n'est pas stocké dans une table CRUD classique, mais reconstruit en relisant son journal d'événements.
2. **Idempotence :** Les consommateurs doivent gérer les doublons potentiels (At-least-once delivery).

- **Conséquences :**
- ✅ **Auditabilité Totale :** On peut "rembobiner" le système pour voir l'état exact lors d'une décision passée.
- ✅ **Robustesse :** En cas de crash, un agent peut reconstruire son contexte en relisant le topic.
- ⚠️ **Courbe d'apprentissage :** Le paradigme Event Sourcing est plus complexe à implémenter que le CRUD standard.

---

## ADR-005 : Cadre d'Évaluation Agentique (Le Diamant)

- **Statut :** Accepté
- **Contexte :**
  Les tests unitaires classiques (assert x == y) ne fonctionnent pas bien avec les LLM dont les réponses varient sémantiquement mais restent correctes. Nous devons valider la "compétence" et la "sécurité".
- **Décision :**
  Adoption du **"Diamant de l'Évaluation"** décrit dans la littérature du projet.

1. **Tests Unitaires :** Pour le code déterministe (outils, connecteurs).
2. **Tests Cognitifs (Model-based Evaluation) :** Utilisation d'un "LLM Juge" pour scorer la qualité des réponses sur des critères (Factualité, Pertinence).
3. **Tests d'Adversité (Red Teaming) :** Scénarios d'attaque spécifiques (injections de prompt).
4. **Simulation d'Écosystème :** Tests d'intégration de bout en bout.

- **Conséquences :**
- ✅ **Assurance Qualité Adaptée :** Couvre les spécificités de l'IA Générative.
- ✅ **Confiance :** Permet de déployer en production avec des métriques de fiabilité.

---

## ADR-006 : Configuration Externalisée (YAML + Environment Override)

- **Statut :** Accepté
- **Date :** 2026-01-09
- **Contexte :**
  Les agents IA nécessitent une configuration flexible pour s'adapter à différents environnements (développement, staging, production) sans modification du code source. Les paramètres critiques incluent : modèles LLM, températures, seuils de décision, et endpoints Kafka. Une approche de configuration "hardcodée" dans le code source crée des risques lors des déploiements et complique les tests.
- **Décision :**
  Nous utiliserons un fichier **`config.yaml`** centralisé avec support de **surcharge via variables d'environnement**.

  **Pattern de surcharge :** `SECTION__KEY__SUBKEY` (double underscore pour la hiérarchie)

  Exemple :

  ```yaml
  # config.yaml
  agents:
    risk_agent:
      model: "claude-sonnet-4-20250514"
  ```

  Surcharge :

  ```bash
  export AGENTS__RISK_AGENT__MODEL=claude-3-5-haiku-20241022
  ```

  **Implémentation :** Module `src/shared/config_loader.py` avec :

  - `load_config()` : Charge YAML + applique env overrides
  - `get_agent_config(agent_name)` : Configuration spécifique par agent
  - `get_thresholds()` : Seuils de décision
  - Conversion de types automatique (int, float, bool)

- **Conséquences :**
- ✅ **Flexibilité Opérationnelle :** Changement de configuration sans redéploiement.
- ✅ **Sécurité :** Les secrets (API keys) restent dans l'environnement, pas dans les fichiers.
- ✅ **Cohérence :** Un seul fichier de référence pour tous les agents.
- ⚠️ **Complexité :** Le pattern `__` peut être contre-intuitif pour certains développeurs.
- ⚠️ **Validation :** Pas de validation de schéma à ce stade (risque de typos).

---

## ADR-007 : Stratégie de Sélection des Modèles LLM par Agent

- **Statut :** Accepté
- **Date :** 2026-01-09
- **Contexte :**
  Anthropic propose une famille de modèles Claude avec des compromis coût/performance différents. Utiliser le même modèle pour tous les agents ignore ces différences et peut entraîner soit des coûts excessifs, soit une qualité insuffisante pour certaines tâches.

  | Modèle                   | Caractéristique     | Coût Relatif |
  | ------------------------ | ------------------- | ------------ |
  | Claude 3.5 Haiku         | Rapide, économique  | $            |
  | Claude 3.5 Sonnet        | Équilibré           | $$           |
  | Claude Sonnet 4 / Opus 4 | Raisonnement avancé | $$$          |

- **Décision :**
  Chaque agent utilisera le modèle **approprié à sa complexité cognitive** :

  1. **Intake Agent** → **Claude 3.5 Haiku**

     - Tâche : Validation structurelle simple
     - Justification : Pas de raisonnement complexe, volume élevé
     - Température : 0.0 (déterministe)

  2. **Risk Agent** → **Claude Sonnet 4** (ou Opus 4.5 pour cas critiques)

     - Tâche : Analyse de risque multi-factorielle, RAG
     - Justification : Nécessite un raisonnement sur des politiques complexes
     - Température : 0.2 (légère créativité pour interprétation)

  3. **Decision Agent** → **Claude 3.5 Sonnet**
     - Tâche : Décision finale, analyse zone grise
     - Justification : Équilibre qualité/coût pour décisions structurées
     - Température : 0.1 (conservateur pour cohérence)

- **Conséquences :**
- ✅ **Optimisation des Coûts :** Réduction estimée de 40-60% vs utilisation uniforme d'Opus.
- ✅ **Performance Adaptée :** Chaque agent a le "cerveau" approprié à sa mission.
- ✅ **Flexibilité :** Configurable via `config.yaml` ou env pour tests A/B.
- ⚠️ **Maintenance :** Nécessite une réévaluation lors des sorties de nouveaux modèles.
- ⚠️ **Latence :** Les modèles supérieurs (Opus) ont une latence plus élevée.

---

## 📚 Navigation

| ⬅️ Précédent                   |                        🏠 Index                         |                                   ➡️ Suivant |
| :----------------------------- | :-----------------------------------------------------: | -------------------------------------------: |
| [00-Readme.md](./00-Readme.md) | [Documentation](./00-Readme.md#-documentation-complète) | [02-DataContracts.md](./02-DataContracts.md) |
