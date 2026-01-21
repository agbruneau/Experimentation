"""
Scénario CROSS-03 : Sécuriser le Gateway.

Ce scénario démontre comment implémenter l'authentification JWT
et l'autorisation RBAC sur un API Gateway.

Objectif pédagogique:
- Générer et valider des tokens JWT
- Implémenter le contrôle d'accès basé sur les rôles (RBAC)
- Gérer les cas d'erreur (token invalide, permissions manquantes)
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
    user_id: str = ""
    user_role: str = ""
    jwt_token: str = ""
    token_valid: bool = False
    permissions: List[str] = field(default_factory=list)
    api_calls: List[Dict] = field(default_factory=list)
    access_denied_count: int = 0
    events: List[Dict] = field(default_factory=list)


scenario = {
    "id": "CROSS-03",
    "title": "Sécuriser le Gateway",
    "description": "Implémentez l'authentification JWT et l'autorisation RBAC",
    "pillar": "cross_cutting",
    "complexity": 2,
    "learning_objectives": [
        "Comprendre la structure d'un token JWT",
        "Valider un token (signature, expiration, issuer)",
        "Extraire les rôles et permissions du token",
        "Appliquer le contrôle d'accès RBAC"
    ],
    "steps": [
        {
            "id": 1,
            "title": "Configurer OAuth",
            "instruction": "Un Identity Provider (IdP) est configuré. Observez les endpoints disponibles.",
            "expected_result": "L'IdP expose /token pour l'authentification.",
            "action": "configure_oauth"
        },
        {
            "id": 2,
            "title": "Générer un JWT",
            "instruction": "Un agent se connecte. L'IdP génère un JWT avec ses rôles (agent).",
            "expected_result": "Un token JWT est retourné avec les claims user_id, roles, exp.",
            "action": "generate_jwt",
            "params": {"user_id": "agent42", "role": "agent"}
        },
        {
            "id": 3,
            "title": "Valider le token",
            "instruction": "Le Gateway reçoit une requête avec le token. Il vérifie la signature et l'expiration.",
            "expected_result": "Le token est validé. La signature est correcte et le token n'est pas expiré.",
            "action": "validate_token"
        },
        {
            "id": 4,
            "title": "Extraire les droits",
            "instruction": "Le Gateway extrait les rôles et permissions du token.",
            "expected_result": "L'agent a les permissions: quote:create, quote:read, policy:read, customer:read.",
            "action": "extract_permissions"
        },
        {
            "id": 5,
            "title": "Appliquer RBAC",
            "instruction": "L'agent demande à créer un devis (POST /quotes). Vérifiez s'il a la permission.",
            "expected_result": "Permission quote:create accordée. La requête est autorisée.",
            "action": "check_permission",
            "params": {"endpoint": "POST /quotes", "required_permission": "quote:create"}
        },
        {
            "id": 6,
            "title": "Refuser un accès",
            "instruction": "L'agent tente de supprimer une police (DELETE /policies/123). Vérifiez les permissions.",
            "expected_result": "Permission policy:delete refusée. Erreur 403 Forbidden retournée.",
            "action": "check_permission",
            "params": {"endpoint": "DELETE /policies/123", "required_permission": "policy:delete"}
        },
        {
            "id": 7,
            "title": "Audit de sécurité",
            "instruction": "Consultez le log d'audit des accès autorisés et refusés.",
            "expected_result": "L'audit montre toutes les tentatives d'accès avec leur résultat.",
            "action": "show_audit_log"
        }
    ],
    "initial_state": ScenarioState().__dict__,
    "config": {
        "jwt": {
            "issuer": "interop-learning",
            "audience": "api.insurance.com",
            "expiration_minutes": 30,
            "algorithm": "HS256"
        },
        "roles": {
            "viewer": ["quote:read", "policy:read", "claim:read", "customer:read"],
            "agent": ["quote:create", "quote:read", "quote:update",
                     "policy:read", "customer:read", "customer:create"],
            "underwriter": ["quote:*", "policy:create", "policy:read",
                          "policy:update", "policy:cancel", "customer:read"],
            "admin": ["*"]
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
        # Configuration OAuth
        event["action"] = "configure_oauth"
        event["oauth_config"] = {
            "issuer": "https://auth.insurance.com",
            "authorization_endpoint": "/oauth/authorize",
            "token_endpoint": "/oauth/token",
            "jwks_uri": "/.well-known/jwks.json"
        }
        event["message"] = "OAuth provider configured"

    elif step_id == 2:
        # Générer JWT
        user_id = params.get("user_id", "agent42")
        role = params.get("role", "agent")

        # Simuler un token JWT (header.payload.signature)
        token_id = uuid.uuid4().hex[:16]
        jwt_token = f"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.{token_id}.signature"

        new_state.user_id = user_id
        new_state.user_role = role
        new_state.jwt_token = jwt_token

        event["action"] = "generate_jwt"
        event["token"] = jwt_token[:50] + "..."
        event["claims"] = {
            "sub": user_id,
            "roles": [role],
            "iss": "interop-learning",
            "aud": "api.insurance.com",
            "exp": "2024-03-15T15:00:00Z",
            "iat": "2024-03-15T14:30:00Z"
        }
        event["message"] = f"JWT generated for user {user_id} with role {role}"

    elif step_id == 3:
        # Valider le token
        new_state.token_valid = True

        event["action"] = "validate_token"
        event["validations"] = [
            {"check": "signature", "result": "VALID", "message": "HMAC-SHA256 signature verified"},
            {"check": "expiration", "result": "VALID", "message": "Token not expired"},
            {"check": "issuer", "result": "VALID", "message": "Issuer matches expected"},
            {"check": "audience", "result": "VALID", "message": "Audience matches expected"}
        ]
        event["message"] = "Token validation successful"

    elif step_id == 4:
        # Extraire les permissions
        role = new_state.user_role
        roles_config = scenario["config"]["roles"]

        permissions = roles_config.get(role, [])
        new_state.permissions = permissions

        event["action"] = "extract_permissions"
        event["user_id"] = new_state.user_id
        event["role"] = role
        event["permissions"] = permissions
        event["message"] = f"Extracted {len(permissions)} permissions for role {role}"

    elif step_id == 5:
        # Vérifier permission accordée
        endpoint = params.get("endpoint", "POST /quotes")
        required = params.get("required_permission", "quote:create")

        has_permission = required in new_state.permissions or "*" in new_state.permissions

        api_call = {
            "timestamp": datetime.now().isoformat(),
            "endpoint": endpoint,
            "user": new_state.user_id,
            "role": new_state.user_role,
            "required_permission": required,
            "granted": has_permission,
            "status": 200 if has_permission else 403
        }
        new_state.api_calls.append(api_call)

        event["action"] = "check_permission"
        event["endpoint"] = endpoint
        event["required_permission"] = required
        event["granted"] = has_permission
        event["status_code"] = 200
        event["message"] = f"Access GRANTED to {endpoint}"

    elif step_id == 6:
        # Vérifier permission refusée
        endpoint = params.get("endpoint", "DELETE /policies/123")
        required = params.get("required_permission", "policy:delete")

        has_permission = required in new_state.permissions or "*" in new_state.permissions

        api_call = {
            "timestamp": datetime.now().isoformat(),
            "endpoint": endpoint,
            "user": new_state.user_id,
            "role": new_state.user_role,
            "required_permission": required,
            "granted": has_permission,
            "status": 200 if has_permission else 403
        }
        new_state.api_calls.append(api_call)

        if not has_permission:
            new_state.access_denied_count += 1

        event["action"] = "check_permission"
        event["endpoint"] = endpoint
        event["required_permission"] = required
        event["granted"] = has_permission
        event["status_code"] = 403
        event["error"] = {
            "code": "FORBIDDEN",
            "message": f"User {new_state.user_id} does not have permission {required}"
        }
        event["message"] = f"Access DENIED to {endpoint}"

    elif step_id == 7:
        # Afficher le log d'audit
        event["action"] = "show_audit_log"
        event["audit_entries"] = new_state.api_calls
        event["summary"] = {
            "total_requests": len(new_state.api_calls),
            "granted": len([c for c in new_state.api_calls if c["granted"]]),
            "denied": len([c for c in new_state.api_calls if not c["granted"]])
        }
        event["message"] = f"Audit log contains {len(new_state.api_calls)} entries"

    new_state.current_step = step_id
    new_state.events.append(event)

    return new_state.__dict__


def get_visualization_data(state: Dict) -> Dict:
    """
    Génère les données pour la visualisation.
    """
    return {
        "nodes": [
            {"id": "client", "label": "Client\n(Agent)", "type": "client", "status": "authenticated" if state.get("token_valid") else "anonymous"},
            {"id": "gateway", "label": "API Gateway\n+ JWT Validation", "type": "gateway", "status": "healthy"},
            {"id": "idp", "label": "Identity\nProvider", "type": "service", "status": "healthy"},
            {"id": "rbac", "label": "RBAC\nEngine", "type": "pattern", "status": "healthy"},
            {"id": "quote_engine", "label": "Quote\nEngine", "type": "service", "status": "healthy"},
            {"id": "policy_admin", "label": "Policy\nAdmin", "type": "service", "status": "healthy"}
        ],
        "links": [
            {"source": "client", "target": "gateway", "type": "auth", "label": "JWT Token"},
            {"source": "gateway", "target": "idp", "type": "sync", "label": "Validate"},
            {"source": "gateway", "target": "rbac", "type": "sync", "label": "Check Permission"},
            {"source": "gateway", "target": "quote_engine", "type": "sync", "label": "Authorized"},
            {"source": "gateway", "target": "policy_admin", "type": "blocked", "label": "Denied"}
        ],
        "security_context": {
            "user_id": state.get("user_id", ""),
            "role": state.get("user_role", ""),
            "token_valid": state.get("token_valid", False),
            "permissions": state.get("permissions", [])
        },
        "api_calls": state.get("api_calls", []),
        "access_denied_count": state.get("access_denied_count", 0)
    }
