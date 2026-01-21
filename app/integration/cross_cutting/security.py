"""
Sécurité des intégrations - JWT, OAuth et RBAC.

Ce module fournit une implémentation simple pour:
- Génération et validation de tokens JWT
- Gestion des rôles et permissions (RBAC)
- Contexte de sécurité
"""
import base64
import hashlib
import hmac
import json
import time
import uuid
from typing import Any, Dict, List, Optional, Set
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from enum import Enum


# ========== JWT MANAGER ==========

class JWTError(Exception):
    """Exception pour les erreurs JWT."""
    pass


class TokenExpiredError(JWTError):
    """Token expiré."""
    pass


class InvalidTokenError(JWTError):
    """Token invalide."""
    pass


@dataclass
class JWTConfig:
    """Configuration JWT."""
    secret_key: str = "super-secret-key-for-demo-only"
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 30
    refresh_token_expire_days: int = 7
    issuer: str = "interop-learning"


@dataclass
class TokenPayload:
    """Payload d'un token JWT."""
    sub: str                          # Subject (user ID)
    exp: int                          # Expiration timestamp
    iat: int                          # Issued at timestamp
    jti: str                          # JWT ID
    iss: str = "interop-learning"     # Issuer
    aud: str = "interop-api"          # Audience
    scope: List[str] = field(default_factory=list)  # Scopes
    roles: List[str] = field(default_factory=list)  # Roles
    claims: Dict[str, Any] = field(default_factory=dict)  # Custom claims

    def to_dict(self) -> Dict:
        return {
            "sub": self.sub,
            "exp": self.exp,
            "iat": self.iat,
            "jti": self.jti,
            "iss": self.iss,
            "aud": self.aud,
            "scope": " ".join(self.scope),
            "roles": self.roles,
            **self.claims
        }

    @classmethod
    def from_dict(cls, data: Dict) -> 'TokenPayload':
        scope = data.get("scope", "")
        if isinstance(scope, str):
            scope = scope.split() if scope else []

        return cls(
            sub=data.get("sub", ""),
            exp=data.get("exp", 0),
            iat=data.get("iat", 0),
            jti=data.get("jti", ""),
            iss=data.get("iss", ""),
            aud=data.get("aud", ""),
            scope=scope,
            roles=data.get("roles", []),
            claims={k: v for k, v in data.items()
                   if k not in ["sub", "exp", "iat", "jti", "iss", "aud", "scope", "roles"]}
        )


class JWTManager:
    """
    Gestionnaire de tokens JWT.

    Features:
    - Génération de tokens d'accès et de refresh
    - Validation et décodage
    - Blacklist de tokens révoqués
    """

    def __init__(self, config: JWTConfig = None):
        self.config = config or JWTConfig()
        self._blacklist: Set[str] = set()
        self._issued_tokens: List[Dict] = []

    def _base64url_encode(self, data: bytes) -> str:
        """Encode en base64url sans padding."""
        return base64.urlsafe_b64encode(data).rstrip(b'=').decode('utf-8')

    def _base64url_decode(self, data: str) -> bytes:
        """Decode depuis base64url."""
        padding = 4 - len(data) % 4
        if padding != 4:
            data += '=' * padding
        return base64.urlsafe_b64decode(data)

    def _sign(self, message: str) -> str:
        """Signe un message avec HMAC-SHA256."""
        signature = hmac.new(
            self.config.secret_key.encode('utf-8'),
            message.encode('utf-8'),
            hashlib.sha256
        ).digest()
        return self._base64url_encode(signature)

    def create_token(
        self,
        user_id: str,
        roles: List[str] = None,
        scope: List[str] = None,
        expires_in_minutes: int = None,
        **claims
    ) -> str:
        """
        Crée un token JWT.

        Args:
            user_id: Identifiant de l'utilisateur
            roles: Liste des rôles
            scope: Liste des scopes
            expires_in_minutes: Durée de validité
            **claims: Claims personnalisés

        Returns:
            Token JWT encodé
        """
        now = int(time.time())
        exp_minutes = expires_in_minutes or self.config.access_token_expire_minutes

        payload = TokenPayload(
            sub=user_id,
            exp=now + (exp_minutes * 60),
            iat=now,
            jti=str(uuid.uuid4()),
            iss=self.config.issuer,
            roles=roles or [],
            scope=scope or [],
            claims=claims
        )

        # Header
        header = {"alg": self.config.algorithm, "typ": "JWT"}
        header_b64 = self._base64url_encode(json.dumps(header).encode('utf-8'))

        # Payload
        payload_b64 = self._base64url_encode(json.dumps(payload.to_dict()).encode('utf-8'))

        # Signature
        message = f"{header_b64}.{payload_b64}"
        signature = self._sign(message)

        token = f"{message}.{signature}"

        # Stocker pour audit
        self._issued_tokens.append({
            "jti": payload.jti,
            "sub": user_id,
            "iat": datetime.fromtimestamp(now).isoformat(),
            "exp": datetime.fromtimestamp(payload.exp).isoformat(),
            "roles": roles or []
        })

        return token

    def create_refresh_token(self, user_id: str) -> str:
        """Crée un token de refresh."""
        return self.create_token(
            user_id=user_id,
            expires_in_minutes=self.config.refresh_token_expire_days * 24 * 60,
            type="refresh"
        )

    def decode_token(self, token: str, verify: bool = True) -> TokenPayload:
        """
        Décode et valide un token JWT.

        Args:
            token: Token JWT encodé
            verify: Si True, vérifie la signature et l'expiration

        Returns:
            Payload décodé

        Raises:
            InvalidTokenError: Si le token est malformé ou invalide
            TokenExpiredError: Si le token est expiré
        """
        try:
            parts = token.split('.')
            if len(parts) != 3:
                raise InvalidTokenError("Token mal formé")

            header_b64, payload_b64, signature = parts

            if verify:
                # Vérifier la signature
                message = f"{header_b64}.{payload_b64}"
                expected_signature = self._sign(message)
                if not hmac.compare_digest(signature, expected_signature):
                    raise InvalidTokenError("Signature invalide")

            # Décoder le payload
            payload_json = self._base64url_decode(payload_b64).decode('utf-8')
            payload_data = json.loads(payload_json)
            payload = TokenPayload.from_dict(payload_data)

            if verify:
                # Vérifier l'expiration
                if payload.exp < int(time.time()):
                    raise TokenExpiredError("Token expiré")

                # Vérifier la blacklist
                if payload.jti in self._blacklist:
                    raise InvalidTokenError("Token révoqué")

            return payload

        except (json.JSONDecodeError, KeyError) as e:
            raise InvalidTokenError(f"Token mal formé: {e}")

    def validate_token(self, token: str) -> bool:
        """Valide un token sans lever d'exception."""
        try:
            self.decode_token(token)
            return True
        except JWTError:
            return False

    def revoke_token(self, token: str):
        """Révoque un token en l'ajoutant à la blacklist."""
        try:
            payload = self.decode_token(token, verify=False)
            self._blacklist.add(payload.jti)
        except JWTError:
            pass

    def get_issued_tokens(self, limit: int = 50) -> List[Dict]:
        """Retourne les tokens émis récemment."""
        return self._issued_tokens[-limit:]

    def clear_blacklist(self):
        """Vide la blacklist."""
        self._blacklist.clear()


# ========== RBAC (Role-Based Access Control) ==========

class Permission(Enum):
    """Permissions disponibles."""
    # Quotes
    QUOTE_READ = "quote:read"
    QUOTE_CREATE = "quote:create"
    QUOTE_UPDATE = "quote:update"
    QUOTE_DELETE = "quote:delete"

    # Policies
    POLICY_READ = "policy:read"
    POLICY_CREATE = "policy:create"
    POLICY_UPDATE = "policy:update"
    POLICY_DELETE = "policy:delete"
    POLICY_CANCEL = "policy:cancel"

    # Claims
    CLAIM_READ = "claim:read"
    CLAIM_CREATE = "claim:create"
    CLAIM_UPDATE = "claim:update"
    CLAIM_APPROVE = "claim:approve"
    CLAIM_REJECT = "claim:reject"

    # Customers
    CUSTOMER_READ = "customer:read"
    CUSTOMER_CREATE = "customer:create"
    CUSTOMER_UPDATE = "customer:update"
    CUSTOMER_DELETE = "customer:delete"

    # Admin
    ADMIN_READ = "admin:read"
    ADMIN_WRITE = "admin:write"
    ADMIN_DELETE = "admin:delete"


@dataclass
class Role:
    """Définition d'un rôle."""
    name: str
    description: str
    permissions: Set[Permission]
    parent_role: Optional[str] = None

    def to_dict(self) -> Dict:
        return {
            "name": self.name,
            "description": self.description,
            "permissions": [p.value for p in self.permissions],
            "parent_role": self.parent_role
        }


# Rôles prédéfinis
PREDEFINED_ROLES = {
    "viewer": Role(
        name="viewer",
        description="Lecture seule",
        permissions={
            Permission.QUOTE_READ,
            Permission.POLICY_READ,
            Permission.CLAIM_READ,
            Permission.CUSTOMER_READ
        }
    ),
    "agent": Role(
        name="agent",
        description="Agent d'assurance",
        permissions={
            Permission.QUOTE_READ, Permission.QUOTE_CREATE, Permission.QUOTE_UPDATE,
            Permission.POLICY_READ, Permission.POLICY_CREATE,
            Permission.CLAIM_READ, Permission.CLAIM_CREATE,
            Permission.CUSTOMER_READ, Permission.CUSTOMER_CREATE, Permission.CUSTOMER_UPDATE
        },
        parent_role="viewer"
    ),
    "underwriter": Role(
        name="underwriter",
        description="Souscripteur",
        permissions={
            Permission.QUOTE_READ, Permission.QUOTE_CREATE, Permission.QUOTE_UPDATE, Permission.QUOTE_DELETE,
            Permission.POLICY_READ, Permission.POLICY_CREATE, Permission.POLICY_UPDATE, Permission.POLICY_CANCEL,
            Permission.CLAIM_READ,
            Permission.CUSTOMER_READ, Permission.CUSTOMER_UPDATE
        },
        parent_role="agent"
    ),
    "claims_handler": Role(
        name="claims_handler",
        description="Gestionnaire de sinistres",
        permissions={
            Permission.CLAIM_READ, Permission.CLAIM_CREATE, Permission.CLAIM_UPDATE,
            Permission.CLAIM_APPROVE, Permission.CLAIM_REJECT,
            Permission.POLICY_READ,
            Permission.CUSTOMER_READ
        }
    ),
    "admin": Role(
        name="admin",
        description="Administrateur",
        permissions=set(Permission),  # Toutes les permissions
        parent_role="underwriter"
    )
}


class RBACManager:
    """
    Gestionnaire de contrôle d'accès basé sur les rôles.

    Features:
    - Définition de rôles et permissions
    - Héritage de rôles
    - Vérification des accès
    """

    def __init__(self):
        self._roles: Dict[str, Role] = PREDEFINED_ROLES.copy()
        self._user_roles: Dict[str, Set[str]] = {}

    def add_role(self, role: Role):
        """Ajoute un rôle."""
        self._roles[role.name] = role

    def get_role(self, role_name: str) -> Optional[Role]:
        """Récupère un rôle par son nom."""
        return self._roles.get(role_name)

    def get_all_roles(self) -> List[Dict]:
        """Retourne tous les rôles."""
        return [r.to_dict() for r in self._roles.values()]

    def assign_role(self, user_id: str, role_name: str):
        """Assigne un rôle à un utilisateur."""
        if role_name not in self._roles:
            raise ValueError(f"Rôle inconnu: {role_name}")

        if user_id not in self._user_roles:
            self._user_roles[user_id] = set()
        self._user_roles[user_id].add(role_name)

    def remove_role(self, user_id: str, role_name: str):
        """Retire un rôle à un utilisateur."""
        if user_id in self._user_roles:
            self._user_roles[user_id].discard(role_name)

    def get_user_roles(self, user_id: str) -> List[str]:
        """Récupère les rôles d'un utilisateur."""
        return list(self._user_roles.get(user_id, set()))

    def _get_role_permissions(self, role_name: str, visited: Set[str] = None) -> Set[Permission]:
        """Récupère toutes les permissions d'un rôle (incluant héritage)."""
        if visited is None:
            visited = set()

        if role_name in visited:
            return set()
        visited.add(role_name)

        role = self._roles.get(role_name)
        if not role:
            return set()

        permissions = role.permissions.copy()

        # Héritage
        if role.parent_role:
            permissions.update(self._get_role_permissions(role.parent_role, visited))

        return permissions

    def get_user_permissions(self, user_id: str) -> Set[Permission]:
        """Récupère toutes les permissions d'un utilisateur."""
        permissions = set()
        for role_name in self._user_roles.get(user_id, set()):
            permissions.update(self._get_role_permissions(role_name))
        return permissions

    def has_permission(self, user_id: str, permission: Permission) -> bool:
        """Vérifie si un utilisateur a une permission."""
        return permission in self.get_user_permissions(user_id)

    def has_any_permission(self, user_id: str, permissions: List[Permission]) -> bool:
        """Vérifie si un utilisateur a au moins une des permissions."""
        user_permissions = self.get_user_permissions(user_id)
        return any(p in user_permissions for p in permissions)

    def has_all_permissions(self, user_id: str, permissions: List[Permission]) -> bool:
        """Vérifie si un utilisateur a toutes les permissions."""
        user_permissions = self.get_user_permissions(user_id)
        return all(p in user_permissions for p in permissions)

    def check_permission(self, user_id: str, permission: Permission):
        """Vérifie une permission et lève une exception si non autorisé."""
        if not self.has_permission(user_id, permission):
            raise PermissionError(
                f"Utilisateur {user_id} n'a pas la permission {permission.value}"
            )


# ========== SECURITY CONTEXT ==========

@dataclass
class SecurityContext:
    """
    Contexte de sécurité pour une requête.

    Combine l'identité (JWT) et les autorisations (RBAC).
    """
    user_id: str
    roles: List[str]
    permissions: Set[Permission]
    token_payload: Optional[TokenPayload] = None
    authenticated: bool = False
    authentication_method: str = "none"
    client_ip: Optional[str] = None
    request_id: Optional[str] = None

    def has_permission(self, permission: Permission) -> bool:
        """Vérifie si le contexte a une permission."""
        return permission in self.permissions

    def has_role(self, role: str) -> bool:
        """Vérifie si le contexte a un rôle."""
        return role in self.roles

    def require_permission(self, permission: Permission):
        """Exige une permission ou lève une exception."""
        if not self.has_permission(permission):
            raise PermissionError(
                f"Permission requise: {permission.value}"
            )

    def require_role(self, role: str):
        """Exige un rôle ou lève une exception."""
        if not self.has_role(role):
            raise PermissionError(f"Rôle requis: {role}")

    def to_dict(self) -> Dict:
        return {
            "user_id": self.user_id,
            "roles": self.roles,
            "permissions": [p.value for p in self.permissions],
            "authenticated": self.authenticated,
            "authentication_method": self.authentication_method,
            "client_ip": self.client_ip,
            "request_id": self.request_id
        }


class SecurityManager:
    """
    Gestionnaire de sécurité combinant JWT et RBAC.

    Usage:
        security = SecurityManager()

        # Créer un token
        token = security.create_token("user123", roles=["agent"])

        # Valider et créer un contexte
        context = security.authenticate(token)

        # Vérifier les permissions
        if context.has_permission(Permission.QUOTE_CREATE):
            ...
    """

    def __init__(self, jwt_config: JWTConfig = None):
        self.jwt_manager = JWTManager(jwt_config)
        self.rbac_manager = RBACManager()

    def create_token(
        self,
        user_id: str,
        roles: List[str] = None,
        **claims
    ) -> str:
        """Crée un token JWT avec les rôles."""
        return self.jwt_manager.create_token(
            user_id=user_id,
            roles=roles or [],
            **claims
        )

    def authenticate(
        self,
        token: str,
        client_ip: str = None
    ) -> SecurityContext:
        """
        Authentifie un token et crée un contexte de sécurité.

        Args:
            token: Token JWT
            client_ip: Adresse IP du client

        Returns:
            Contexte de sécurité

        Raises:
            JWTError: Si le token est invalide
        """
        payload = self.jwt_manager.decode_token(token)

        # Calculer les permissions depuis les rôles
        permissions = set()
        for role_name in payload.roles:
            role = self.rbac_manager.get_role(role_name)
            if role:
                permissions.update(role.permissions)

        return SecurityContext(
            user_id=payload.sub,
            roles=payload.roles,
            permissions=permissions,
            token_payload=payload,
            authenticated=True,
            authentication_method="jwt",
            client_ip=client_ip,
            request_id=str(uuid.uuid4())
        )

    def authenticate_or_anonymous(
        self,
        token: Optional[str],
        client_ip: str = None
    ) -> SecurityContext:
        """
        Authentifie ou retourne un contexte anonyme.

        Args:
            token: Token JWT ou None
            client_ip: Adresse IP du client

        Returns:
            Contexte de sécurité (authentifié ou anonyme)
        """
        if token:
            try:
                return self.authenticate(token, client_ip)
            except JWTError:
                pass

        return SecurityContext(
            user_id="anonymous",
            roles=[],
            permissions=set(),
            authenticated=False,
            authentication_method="none",
            client_ip=client_ip,
            request_id=str(uuid.uuid4())
        )

    def revoke_token(self, token: str):
        """Révoque un token."""
        self.jwt_manager.revoke_token(token)

    def assign_role(self, user_id: str, role_name: str):
        """Assigne un rôle à un utilisateur."""
        self.rbac_manager.assign_role(user_id, role_name)

    def get_roles(self) -> List[Dict]:
        """Retourne tous les rôles disponibles."""
        return self.rbac_manager.get_all_roles()


# Instance globale
_security_manager: Optional[SecurityManager] = None


def get_security_manager() -> SecurityManager:
    """Récupère ou crée le gestionnaire de sécurité global."""
    global _security_manager
    if _security_manager is None:
        _security_manager = SecurityManager()
    return _security_manager


def reset_security():
    """Réinitialise le gestionnaire de sécurité."""
    global _security_manager
    _security_manager = None
