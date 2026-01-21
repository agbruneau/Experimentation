"""
Scénario CROSS-01 : Panne du tarificateur externe avec Circuit Breaker.

Ce scénario démontre comment le pattern Circuit Breaker protège le système
contre les pannes en cascade quand un service externe devient indisponible.

Objectif pédagogique:
- Comprendre les trois états du Circuit Breaker
- Observer la protection automatique
- Utiliser un fallback quand le circuit est ouvert
"""
import asyncio
from typing import Dict, Any, List
from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class ScenarioStep:
    """Étape d'un scénario."""
    id: int
    title: str
    instruction: str
    expected_result: str
    validation_fn: str = ""


@dataclass
class ScenarioState:
    """État du scénario en cours."""
    current_step: int = 1
    circuit_state: str = "CLOSED"
    failure_count: int = 0
    success_count: int = 0
    rating_service_healthy: bool = True
    fallback_used: bool = False
    calls_made: int = 0
    calls_rejected: int = 0
    events: List[Dict] = field(default_factory=list)


scenario = {
    "id": "CROSS-01",
    "title": "Panne tarificateur externe",
    "description": "Gérez la panne du service de tarification externe avec Circuit Breaker",
    "pillar": "cross_cutting",
    "complexity": 2,
    "learning_objectives": [
        "Comprendre les trois états du Circuit Breaker (CLOSED, OPEN, HALF_OPEN)",
        "Observer comment le circuit s'ouvre après N échecs",
        "Utiliser un fallback quand le circuit est ouvert",
        "Observer la récupération automatique"
    ],
    "steps": [
        {
            "id": 1,
            "title": "Appel normal",
            "instruction": "Le tarificateur externe est disponible. Effectuez un appel de tarification pour un devis AUTO.",
            "expected_result": "Le circuit est CLOSED, l'appel réussit, la prime est calculée.",
            "action": "call_rating",
            "params": {"product": "AUTO", "risk_data": {"age": 35, "experience": 10}}
        },
        {
            "id": 2,
            "title": "Simuler une lenteur",
            "instruction": "Activez la simulation de latence sur le tarificateur (5 secondes de délai).",
            "expected_result": "Les appels commencent à timeout (seuil: 2s).",
            "action": "inject_latency",
            "params": {"latency_seconds": 5}
        },
        {
            "id": 3,
            "title": "Observer les échecs",
            "instruction": "Effectuez 5 appels de tarification. Observez les timeouts et le compteur d'échecs.",
            "expected_result": "Les 5 appels échouent par timeout. Le compteur atteint le seuil (5).",
            "action": "call_rating_multiple",
            "params": {"count": 5}
        },
        {
            "id": 4,
            "title": "Circuit ouvert",
            "instruction": "Le circuit breaker s'ouvre automatiquement après 5 échecs. Observez l'état.",
            "expected_result": "Le circuit passe en état OPEN. Les appels suivants sont rejetés immédiatement.",
            "action": "check_circuit_state"
        },
        {
            "id": 5,
            "title": "Utiliser le fallback",
            "instruction": "Effectuez un nouvel appel. Comme le circuit est ouvert, utilisez le tarif par défaut.",
            "expected_result": "L'appel est rejeté par le circuit breaker. Le fallback retourne un tarif estimatif.",
            "action": "call_with_fallback"
        },
        {
            "id": 6,
            "title": "Restaurer le service",
            "instruction": "Désactivez la latence pour simuler la récupération du tarificateur.",
            "expected_result": "Le service externe est de nouveau disponible.",
            "action": "restore_service"
        },
        {
            "id": 7,
            "title": "Test de récupération",
            "instruction": "Après le délai de reset (30s simulé), le circuit passe en HALF_OPEN. Un appel test est autorisé.",
            "expected_result": "Le circuit passe en HALF_OPEN, l'appel test réussit.",
            "action": "wait_and_test",
            "params": {"wait_seconds": 30}
        },
        {
            "id": 8,
            "title": "Circuit fermé",
            "instruction": "Après 2 succès en HALF_OPEN, le circuit se referme. Le fonctionnement normal reprend.",
            "expected_result": "Le circuit repasse en CLOSED. Les appels fonctionnent normalement.",
            "action": "verify_recovery"
        }
    ],
    "initial_state": ScenarioState().__dict__,
    "config": {
        "circuit_breaker": {
            "failure_threshold": 5,
            "success_threshold": 2,
            "reset_timeout_seconds": 30,
            "timeout_seconds": 2
        },
        "fallback": {
            "default_rates": {
                "AUTO": 500,
                "HOME": 350
            }
        }
    }
}


async def execute_step(step_id: int, state: Dict, params: Dict = None) -> Dict:
    """
    Exécute une étape du scénario.

    Args:
        step_id: Numéro de l'étape
        state: État actuel du scénario
        params: Paramètres optionnels

    Returns:
        Nouvel état après exécution
    """
    params = params or {}
    new_state = ScenarioState(**state)
    event = {"timestamp": datetime.now().isoformat(), "step": step_id}

    if step_id == 1:
        # Appel normal réussi
        new_state.calls_made += 1
        new_state.success_count += 1
        event["action"] = "call_rating"
        event["result"] = "success"
        event["response"] = {"rate": 456.00, "source": "external_api"}
        event["circuit_state"] = "CLOSED"

    elif step_id == 2:
        # Injection de latence
        new_state.rating_service_healthy = False
        event["action"] = "inject_latency"
        event["latency"] = params.get("latency_seconds", 5)

    elif step_id == 3:
        # 5 appels qui échouent
        for i in range(5):
            new_state.calls_made += 1
            new_state.failure_count += 1
            new_state.events.append({
                "timestamp": datetime.now().isoformat(),
                "action": "call_rating",
                "result": "timeout",
                "failure_count": new_state.failure_count
            })
            await asyncio.sleep(0.1)  # Simulation

        event["action"] = "call_rating_multiple"
        event["failures"] = 5
        event["failure_count"] = new_state.failure_count

        # Le circuit s'ouvre après 5 échecs
        if new_state.failure_count >= 5:
            new_state.circuit_state = "OPEN"
            event["circuit_state_changed"] = "CLOSED → OPEN"

    elif step_id == 4:
        # Vérification de l'état
        event["action"] = "check_circuit_state"
        event["circuit_state"] = new_state.circuit_state
        event["message"] = "Circuit is OPEN - calls will be rejected immediately"

    elif step_id == 5:
        # Appel avec fallback
        new_state.calls_made += 1
        new_state.calls_rejected += 1
        new_state.fallback_used = True
        event["action"] = "call_with_fallback"
        event["rejected_by"] = "circuit_breaker"
        event["fallback_rate"] = 500
        event["message"] = "Using fallback rate due to circuit open"

    elif step_id == 6:
        # Restauration du service
        new_state.rating_service_healthy = True
        event["action"] = "restore_service"
        event["message"] = "External rating service is back online"

    elif step_id == 7:
        # Passage en HALF_OPEN et test
        new_state.circuit_state = "HALF_OPEN"
        new_state.calls_made += 1
        new_state.success_count += 1
        event["action"] = "wait_and_test"
        event["circuit_state_changed"] = "OPEN → HALF_OPEN"
        event["test_call"] = "success"
        event["message"] = "Test call succeeded in HALF_OPEN state"

    elif step_id == 8:
        # Fermeture du circuit
        new_state.circuit_state = "CLOSED"
        new_state.failure_count = 0
        event["action"] = "verify_recovery"
        event["circuit_state_changed"] = "HALF_OPEN → CLOSED"
        event["message"] = "Circuit is now CLOSED - normal operation resumed"

    new_state.current_step = step_id
    new_state.events.append(event)

    return new_state.__dict__


def get_visualization_data(state: Dict) -> Dict:
    """
    Génère les données pour la visualisation.

    Returns:
        Données pour D3.js
    """
    return {
        "nodes": [
            {
                "id": "gateway",
                "label": "API Gateway",
                "type": "gateway",
                "status": "healthy"
            },
            {
                "id": "quote_engine",
                "label": "Quote Engine",
                "type": "service",
                "status": "healthy"
            },
            {
                "id": "circuit_breaker",
                "label": f"Circuit Breaker\n[{state.get('circuit_state', 'CLOSED')}]",
                "type": "pattern",
                "status": state.get("circuit_state", "CLOSED").lower()
            },
            {
                "id": "external_rating",
                "label": "External Rating",
                "type": "external",
                "status": "healthy" if state.get("rating_service_healthy", True) else "error"
            },
            {
                "id": "fallback",
                "label": "Fallback\n(Default Rates)",
                "type": "fallback",
                "status": "active" if state.get("fallback_used") else "standby"
            }
        ],
        "links": [
            {"source": "gateway", "target": "quote_engine", "type": "sync"},
            {"source": "quote_engine", "target": "circuit_breaker", "type": "sync"},
            {
                "source": "circuit_breaker",
                "target": "external_rating",
                "type": "sync",
                "blocked": state.get("circuit_state") == "OPEN"
            },
            {
                "source": "circuit_breaker",
                "target": "fallback",
                "type": "fallback",
                "active": state.get("circuit_state") == "OPEN"
            }
        ],
        "metrics": {
            "calls_made": state.get("calls_made", 0),
            "calls_rejected": state.get("calls_rejected", 0),
            "failure_count": state.get("failure_count", 0),
            "success_count": state.get("success_count", 0),
            "circuit_state": state.get("circuit_state", "CLOSED")
        }
    }
