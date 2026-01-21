"""
Scénario CROSS-02 : Tracing distribué.

Ce scénario démontre comment suivre une requête à travers tous les services
grâce au distributed tracing.

Objectif pédagogique:
- Comprendre la propagation du contexte de trace
- Utiliser le trace_id pour corréler les logs
- Visualiser le waterfall des appels
- Identifier les goulots d'étranglement
"""
import asyncio
from typing import Dict, Any, List
from dataclasses import dataclass, field
from datetime import datetime
import uuid


@dataclass
class ScenarioState:
    """État du scénario en cours."""
    current_step: int = 1
    trace_id: str = ""
    spans: List[Dict] = field(default_factory=list)
    logs: List[Dict] = field(default_factory=list)
    total_duration_ms: int = 0
    bottleneck_service: str = ""
    events: List[Dict] = field(default_factory=list)


scenario = {
    "id": "CROSS-02",
    "title": "Tracing distribué",
    "description": "Suivez une requête à travers tous les services",
    "pillar": "cross_cutting",
    "complexity": 2,
    "learning_objectives": [
        "Comprendre les concepts de Trace et Span",
        "Observer la propagation du trace_id via les headers",
        "Corréler les logs de différents services",
        "Identifier le service le plus lent dans la chaîne"
    ],
    "steps": [
        {
            "id": 1,
            "title": "Générer un trace ID",
            "instruction": "Une requête arrive au Gateway. Un identifiant unique (trace_id) est généré pour suivre cette requête.",
            "expected_result": "Un trace_id est créé et attaché à la requête.",
            "action": "generate_trace_id"
        },
        {
            "id": 2,
            "title": "Propager le contexte",
            "instruction": "Le Gateway appelle Quote Engine. Observez comment le trace_id est transmis via les headers HTTP.",
            "expected_result": "Le header X-Trace-ID est inclus dans l'appel vers Quote Engine.",
            "action": "propagate_context"
        },
        {
            "id": 3,
            "title": "Créer des spans enfants",
            "instruction": "Quote Engine appelle plusieurs services. Chaque appel crée un span enfant avec le même trace_id.",
            "expected_result": "Des spans sont créés pour Customer Hub et External Rating, tous liés au même trace_id.",
            "action": "create_child_spans"
        },
        {
            "id": 4,
            "title": "Instrumenter les appels",
            "instruction": "Chaque service log ses actions avec le trace_id. Observez les logs générés.",
            "expected_result": "Les logs de tous les services contiennent le même trace_id.",
            "action": "instrument_calls"
        },
        {
            "id": 5,
            "title": "Visualiser la trace",
            "instruction": "La requête est terminée. Visualisez le waterfall complet des appels.",
            "expected_result": "Un diagramme montre la cascade des appels avec leurs durées.",
            "action": "visualize_trace"
        },
        {
            "id": 6,
            "title": "Identifier un goulot",
            "instruction": "Analysez la trace pour identifier le service le plus lent.",
            "expected_result": "External Rating est identifié comme le goulot (70% du temps total).",
            "action": "find_bottleneck"
        },
        {
            "id": 7,
            "title": "Corréler les logs",
            "instruction": "Recherchez tous les logs avec ce trace_id pour avoir la vue complète de la requête.",
            "expected_result": "Tous les logs pertinents sont regroupés par trace_id.",
            "action": "correlate_logs"
        }
    ],
    "initial_state": ScenarioState().__dict__,
    "config": {
        "services": ["gateway", "quote_engine", "customer_hub", "external_rating"],
        "latencies": {
            "gateway": 10,
            "quote_engine": 50,
            "customer_hub": 30,
            "external_rating": 200
        }
    }
}


async def execute_step(step_id: int, state: Dict, params: Dict = None) -> Dict:
    """
    Exécute une étape du scénario.
    """
    params = params or {}
    new_state = ScenarioState(**state)
    event = {"timestamp": datetime.now().isoformat(), "step": step_id}

    if step_id == 1:
        # Générer un trace ID
        trace_id = f"trace-{uuid.uuid4().hex[:16]}"
        new_state.trace_id = trace_id
        event["action"] = "generate_trace_id"
        event["trace_id"] = trace_id
        event["message"] = f"Generated trace ID: {trace_id}"

        # Premier span (root)
        new_state.spans.append({
            "span_id": f"span-{uuid.uuid4().hex[:8]}",
            "trace_id": trace_id,
            "parent_span_id": None,
            "operation": "POST /quotes",
            "service": "gateway",
            "start_time": 0,
            "duration_ms": 0
        })

    elif step_id == 2:
        # Propagation du contexte
        event["action"] = "propagate_context"
        event["headers"] = {
            "X-Trace-ID": new_state.trace_id,
            "X-Span-ID": new_state.spans[0]["span_id"],
            "X-Parent-Span-ID": ""
        }
        event["message"] = "Context propagated via HTTP headers"

        # Log du Gateway
        new_state.logs.append({
            "timestamp": datetime.now().isoformat(),
            "service": "gateway",
            "level": "INFO",
            "message": "Routing request to quote-engine",
            "trace_id": new_state.trace_id,
            "span_id": new_state.spans[0]["span_id"]
        })

    elif step_id == 3:
        # Créer des spans enfants
        parent_span = new_state.spans[0]

        # Span Quote Engine
        quote_span = {
            "span_id": f"span-{uuid.uuid4().hex[:8]}",
            "trace_id": new_state.trace_id,
            "parent_span_id": parent_span["span_id"],
            "operation": "create_quote",
            "service": "quote_engine",
            "start_time": 10,
            "duration_ms": 280
        }
        new_state.spans.append(quote_span)

        # Span Customer Hub
        new_state.spans.append({
            "span_id": f"span-{uuid.uuid4().hex[:8]}",
            "trace_id": new_state.trace_id,
            "parent_span_id": quote_span["span_id"],
            "operation": "get_customer",
            "service": "customer_hub",
            "start_time": 20,
            "duration_ms": 30
        })

        # Span External Rating
        new_state.spans.append({
            "span_id": f"span-{uuid.uuid4().hex[:8]}",
            "trace_id": new_state.trace_id,
            "parent_span_id": quote_span["span_id"],
            "operation": "calculate_rate",
            "service": "external_rating",
            "start_time": 60,
            "duration_ms": 200
        })

        event["action"] = "create_child_spans"
        event["spans_created"] = 3
        event["message"] = "Child spans created for quote_engine, customer_hub, external_rating"

    elif step_id == 4:
        # Instrumenter les appels (logs)
        trace_id = new_state.trace_id

        logs = [
            {"service": "gateway", "message": "Request received", "duration_ms": 0},
            {"service": "gateway", "message": "Auth validated", "duration_ms": 5},
            {"service": "quote_engine", "message": "Processing quote request", "duration_ms": 10},
            {"service": "customer_hub", "message": "Customer C001 fetched", "duration_ms": 30},
            {"service": "quote_engine", "message": "Calling external rating", "duration_ms": 55},
            {"service": "external_rating", "message": "Calculating rate for AUTO", "duration_ms": 60},
            {"service": "external_rating", "message": "Rate calculated: 456.00", "duration_ms": 260},
            {"service": "quote_engine", "message": "Quote Q001 created", "duration_ms": 280},
            {"service": "gateway", "message": "Response sent", "duration_ms": 290}
        ]

        for log in logs:
            new_state.logs.append({
                "timestamp": datetime.now().isoformat(),
                "service": log["service"],
                "level": "INFO",
                "message": log["message"],
                "trace_id": trace_id,
                "duration_ms": log["duration_ms"]
            })

        event["action"] = "instrument_calls"
        event["logs_generated"] = len(logs)
        event["message"] = f"All services logged with trace_id: {trace_id}"

    elif step_id == 5:
        # Visualiser la trace
        # Calculer la durée totale
        new_state.total_duration_ms = 290
        new_state.spans[0]["duration_ms"] = 290

        event["action"] = "visualize_trace"
        event["trace_id"] = new_state.trace_id
        event["total_duration_ms"] = 290
        event["waterfall"] = [
            {"service": "gateway", "start": 0, "end": 290, "duration": 290},
            {"service": "quote_engine", "start": 10, "end": 290, "duration": 280},
            {"service": "customer_hub", "start": 20, "end": 50, "duration": 30},
            {"service": "external_rating", "start": 60, "end": 260, "duration": 200}
        ]

    elif step_id == 6:
        # Identifier le goulot
        # External Rating = 200ms / 290ms total = 69%
        new_state.bottleneck_service = "external_rating"

        event["action"] = "find_bottleneck"
        event["bottleneck"] = {
            "service": "external_rating",
            "duration_ms": 200,
            "percentage": 69
        }
        event["analysis"] = [
            {"service": "gateway", "duration_ms": 10, "percentage": 3},
            {"service": "quote_engine", "duration_ms": 70, "percentage": 24},
            {"service": "customer_hub", "duration_ms": 30, "percentage": 10},
            {"service": "external_rating", "duration_ms": 200, "percentage": 69}
        ]
        event["recommendation"] = "Consider caching external rating results or using async pattern"

    elif step_id == 7:
        # Corréler les logs
        event["action"] = "correlate_logs"
        event["trace_id"] = new_state.trace_id
        event["correlated_logs"] = len(new_state.logs)
        event["services_involved"] = ["gateway", "quote_engine", "customer_hub", "external_rating"]
        event["message"] = f"Found {len(new_state.logs)} log entries for trace {new_state.trace_id}"

    new_state.current_step = step_id
    new_state.events.append(event)

    return new_state.__dict__


def get_visualization_data(state: Dict) -> Dict:
    """
    Génère les données pour la visualisation.
    """
    spans = state.get("spans", [])

    return {
        "trace_id": state.get("trace_id", ""),
        "nodes": [
            {"id": "gateway", "label": "Gateway", "type": "gateway", "status": "healthy"},
            {"id": "quote_engine", "label": "Quote Engine", "type": "service", "status": "healthy"},
            {"id": "customer_hub", "label": "Customer Hub", "type": "service", "status": "healthy"},
            {"id": "external_rating", "label": "External Rating", "type": "external",
             "status": "slow" if state.get("bottleneck_service") == "external_rating" else "healthy"}
        ],
        "links": [
            {"source": "gateway", "target": "quote_engine", "type": "sync", "duration_ms": 10},
            {"source": "quote_engine", "target": "customer_hub", "type": "sync", "duration_ms": 30},
            {"source": "quote_engine", "target": "external_rating", "type": "sync", "duration_ms": 200}
        ],
        "spans": spans,
        "waterfall": {
            "total_duration_ms": state.get("total_duration_ms", 0),
            "spans": [
                {"service": "gateway", "operation": "POST /quotes", "start": 0, "duration": 290},
                {"service": "quote_engine", "operation": "create_quote", "start": 10, "duration": 280},
                {"service": "customer_hub", "operation": "get_customer", "start": 20, "duration": 30},
                {"service": "external_rating", "operation": "calculate_rate", "start": 60, "duration": 200}
            ]
        },
        "logs": state.get("logs", []),
        "bottleneck": state.get("bottleneck_service", "")
    }
