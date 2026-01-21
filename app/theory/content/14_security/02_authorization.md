# Autorisation et RBAC

## Résumé

L'**autorisation** détermine ce qu'un utilisateur authentifié peut faire. Le **RBAC** (Role-Based Access Control) est le modèle le plus courant : les permissions sont attribuées à des rôles, et les rôles sont attribués aux utilisateurs.

## Points clés

- **Authentification** = Qui êtes-vous ?
- **Autorisation** = Que pouvez-vous faire ?
- **RBAC** = Permissions via rôles
- **Principe du moindre privilège** = Donner uniquement les droits nécessaires

## Modèles d'autorisation

### ACL (Access Control List)
Permissions directement sur les ressources.

```
Document "POL-001":
  - user:alice → read, write
  - user:bob → read
  - group:underwriters → read, write, delete
```

### RBAC (Role-Based Access Control)
Permissions attribuées à des rôles.

```
Rôles:
  agent → [quote:create, policy:read]
  underwriter → [quote:*, policy:*, claim:read]
  admin → [*]

Utilisateurs:
  alice → [agent]
  bob → [underwriter]
  charlie → [admin]
```

### ABAC (Attribute-Based Access Control)
Décisions basées sur des attributs contextuels.

```
Règle: "Un agent peut modifier une police
        SI la police appartient à son portefeuille
        ET la police est active
        ET c'est un jour ouvré"

policy.portfolio == user.portfolio
AND policy.status == "ACTIVE"
AND isBusinessDay()
```

## RBAC en détail

### Hiérarchie des rôles

```
                    admin
                      │
          ┌───────────┼───────────┐
          │           │           │
     underwriter  claims_mgr   finance
          │           │
      ┌───┴───┐       │
      │       │       │
    agent  broker  handler
      │
    viewer
```

Les rôles héritent des permissions de leurs parents.

### Permissions typiques assurance

| Permission | Description |
|------------|-------------|
| quote:read | Consulter les devis |
| quote:create | Créer un devis |
| quote:update | Modifier un devis |
| quote:delete | Supprimer un devis |
| policy:read | Consulter les polices |
| policy:create | Émettre une police |
| policy:update | Modifier une police |
| policy:cancel | Annuler une police |
| claim:read | Consulter les sinistres |
| claim:create | Déclarer un sinistre |
| claim:approve | Approuver un règlement |
| claim:reject | Refuser un sinistre |
| customer:read | Consulter les clients |
| customer:update | Modifier les clients |
| admin:* | Administration système |

### Rôles prédéfinis

```python
ROLES = {
    "viewer": {
        "description": "Lecture seule",
        "permissions": [
            "quote:read",
            "policy:read",
            "claim:read",
            "customer:read"
        ]
    },
    "agent": {
        "description": "Agent commercial",
        "inherits": ["viewer"],
        "permissions": [
            "quote:create",
            "quote:update",
            "customer:create",
            "customer:update"
        ]
    },
    "underwriter": {
        "description": "Souscripteur",
        "inherits": ["agent"],
        "permissions": [
            "quote:delete",
            "policy:create",
            "policy:update",
            "policy:cancel"
        ]
    },
    "claims_handler": {
        "description": "Gestionnaire sinistres",
        "inherits": ["viewer"],
        "permissions": [
            "claim:create",
            "claim:update",
            "claim:approve",
            "claim:reject"
        ]
    },
    "admin": {
        "description": "Administrateur",
        "permissions": ["*"]  # Toutes les permissions
    }
}
```

## Implémentation

### Middleware d'autorisation

```python
def require_permission(permission):
    def decorator(func):
        async def wrapper(request, *args, **kwargs):
            # 1. Extraire l'utilisateur du token
            user = request.state.user

            # 2. Vérifier la permission
            if not user.has_permission(permission):
                raise HTTPException(
                    status_code=403,
                    detail=f"Permission required: {permission}"
                )

            # 3. Exécuter la fonction
            return await func(request, *args, **kwargs)
        return wrapper
    return decorator

# Usage
@router.post("/policies")
@require_permission("policy:create")
async def create_policy(policy: PolicyCreate):
    ...
```

### Vérification des permissions

```python
class User:
    def __init__(self, id, roles):
        self.id = id
        self.roles = roles
        self._permissions = None

    @property
    def permissions(self):
        if self._permissions is None:
            self._permissions = set()
            for role_name in self.roles:
                role = ROLES.get(role_name)
                if role:
                    self._permissions.update(role["permissions"])
                    # Héritage
                    for parent in role.get("inherits", []):
                        self._permissions.update(
                            ROLES[parent]["permissions"]
                        )
        return self._permissions

    def has_permission(self, permission):
        # Admin a toutes les permissions
        if "*" in self.permissions:
            return True
        # Permission exacte
        if permission in self.permissions:
            return True
        # Permission wildcard (ex: "quote:*" couvre "quote:read")
        resource = permission.split(":")[0]
        if f"{resource}:*" in self.permissions:
            return True
        return False
```

## Cas d'usage assurance

### Contrôle d'accès API Gateway

```
POST /api/quotes
Authorization: Bearer <jwt>

JWT Claims:
{
  "sub": "agent42",
  "roles": ["agent"],
  "portfolio": "IDF-NORD"
}

Gateway:
1. Valider le JWT
2. Vérifier permission "quote:create" ✓
3. Router vers Quote Engine
```

### Filtrage des données par rôle

```python
@router.get("/policies")
@require_permission("policy:read")
async def list_policies(request):
    user = request.state.user

    if user.has_permission("admin:*"):
        # Admin voit tout
        return await get_all_policies()

    elif "underwriter" in user.roles:
        # Underwriter voit son département
        return await get_policies(department=user.department)

    else:
        # Agent voit son portefeuille
        return await get_policies(portfolio=user.portfolio)
```

### Audit des actions

```python
async def create_policy(policy, user):
    # Vérifier la permission
    if not user.has_permission("policy:create"):
        raise PermissionDenied()

    # Créer la police
    created = await db.policies.insert(policy)

    # Audit log
    await audit_log.write({
        "action": "policy:create",
        "resource": f"policy:{created.id}",
        "user": user.id,
        "roles": user.roles,
        "timestamp": datetime.now(),
        "details": {
            "policy_number": created.number,
            "customer": policy.customer_id
        }
    })

    return created
```

## Bonnes pratiques

### 1. Principe du moindre privilège
```
# Mauvais
Tous les agents ont accès à toutes les polices

# Bon
Les agents n'ont accès qu'à leur portefeuille
```

### 2. Séparation des responsabilités
```
# Un même utilisateur ne devrait pas pouvoir:
- Créer ET approuver un sinistre
- Modifier ET valider un paiement
```

### 3. Révision régulière des accès
```python
# Rapport des permissions par utilisateur
async def audit_permissions():
    for user in await get_all_users():
        print(f"User: {user.id}")
        print(f"Roles: {user.roles}")
        print(f"Permissions: {user.permissions}")
        print(f"Last activity: {user.last_login}")
        print("---")
```

### 4. Logging des accès refusés
```python
def check_permission(user, permission, resource):
    allowed = user.has_permission(permission)

    # Log toujours, succès ou échec
    security_log.write({
        "user": user.id,
        "permission": permission,
        "resource": resource,
        "allowed": allowed,
        "timestamp": datetime.now()
    })

    if not allowed:
        raise PermissionDenied()
```

## Anti-patterns

1. **Vérification côté client uniquement** : Le serveur doit toujours vérifier
2. **Rôles trop larges** : "admin" pour tout le monde
3. **Hardcoder les permissions** : Difficile à maintenir
4. **Pas d'audit** : Impossible de tracer les accès
5. **Permissions dans l'URL** : `/admin/users` au lieu de vérification
