# Authentification API

## Résumé

L'**authentification** vérifie l'identité de l'appelant. Plusieurs mécanismes existent selon le contexte : API Key pour les partenaires simples, OAuth/JWT pour les applications modernes, mTLS pour le service-to-service.

## Points clés

- **API Key** : Simple, pour partenaires B2B de confiance
- **OAuth 2.0** : Standard pour la délégation d'autorisation
- **JWT** : Token auto-contenu pour les architectures distribuées
- **mTLS** : Authentification mutuelle par certificats

## Comparatif des mécanismes

| Mécanisme | Complexité | Cas d'usage | Sécurité |
|-----------|------------|-------------|----------|
| API Key | Faible | B2B, partenaires | Moyenne |
| Basic Auth | Faible | Tests, interne | Faible |
| OAuth 2.0 | Élevée | Apps tierces, SSO | Élevée |
| JWT | Moyenne | Microservices | Élevée |
| mTLS | Élevée | Service-to-service | Très élevée |

## API Key

### Principe
```
Client → Gateway
Headers:
  X-API-Key: sk_live_abc123def456

Gateway:
  1. Extraire la clé
  2. Vérifier dans le registre
  3. Associer au client/permissions
```

### Avantages
- Simple à implémenter
- Simple pour les partenaires
- Pas d'expiration (sauf révocation)

### Inconvénients
- Secret partagé
- Difficile à révoquer si compromis
- Pas de contexte utilisateur

### Bonnes pratiques
```
# Générer des clés sécurisées
key = "sk_" + environment + "_" + random(32)
# Exemple: sk_live_7f4a8b2c9d1e0f3a4b5c6d7e8f9a0b1c

# Hasher avant stockage
stored_key = hash(api_key)

# Rotation périodique
validité = 1 an max
```

## OAuth 2.0

### Flux Authorization Code (applications web)

```
┌────────┐     ┌─────────────┐     ┌────────────┐
│  User  │     │  Client App │     │  Auth      │
│Browser │     │             │     │  Server    │
└───┬────┘     └──────┬──────┘     └─────┬──────┘
    │                 │                   │
    │  1. Accéder à   │                   │
    │     l'app       │                   │
    │────────────────▶│                   │
    │                 │                   │
    │  2. Redirect    │                   │
    │◀────────────────│                   │
    │  /authorize?    │                   │
    │  client_id=...  │                   │
    │                 │                   │
    │  3. Login       │                   │
    │─────────────────────────────────────▶
    │                 │                   │
    │  4. Consent     │                   │
    │◀─────────────────────────────────────
    │  (autoriser?)   │                   │
    │                 │                   │
    │  5. Redirect    │                   │
    │  avec code      │                   │
    │────────────────▶│                   │
    │                 │                   │
    │                 │  6. Échanger code │
    │                 │  contre token     │
    │                 │──────────────────▶│
    │                 │                   │
    │                 │  7. Access Token  │
    │                 │◀──────────────────│
    │                 │                   │
```

### Flux Client Credentials (machine-to-machine)

```
┌──────────┐                    ┌────────────┐
│  Client  │                    │  Auth      │
│ (Service)│                    │  Server    │
└────┬─────┘                    └─────┬──────┘
     │                                │
     │  POST /token                   │
     │  grant_type=client_credentials │
     │  client_id=...                 │
     │  client_secret=...             │
     │───────────────────────────────▶│
     │                                │
     │  { access_token: "..." }       │
     │◀───────────────────────────────│
     │                                │
```

## JWT (JSON Web Token)

### Structure

```
Header.Payload.Signature

eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJzdWIiOiJ1c2VyMTIzIiwicm9sZXMiOlsiYWdlbnQiXSwiZXhwIjoxNzEwNTAwMDAwfQ.
K8ZlN3PJn2vR5xQ9sT7mUdY6wH1fAqBcDeF
```

### Header
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

### Payload (Claims)
```json
{
  "sub": "user123",           // Subject (user ID)
  "iss": "auth.insurance.com", // Issuer
  "aud": "api.insurance.com",  // Audience
  "exp": 1710500000,          // Expiration
  "iat": 1710496400,          // Issued at
  "roles": ["agent"],         // Custom claim
  "permissions": ["quote:create", "policy:read"]
}
```

### Signature
```
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret
)
```

### Validation

```python
def validate_jwt(token, secret):
    header, payload, signature = token.split('.')

    # 1. Vérifier la signature
    expected_sig = hmac_sha256(f"{header}.{payload}", secret)
    if signature != expected_sig:
        raise InvalidSignature()

    # 2. Décoder le payload
    claims = json.loads(base64_decode(payload))

    # 3. Vérifier l'expiration
    if claims['exp'] < time.now():
        raise TokenExpired()

    # 4. Vérifier l'issuer
    if claims['iss'] != EXPECTED_ISSUER:
        raise InvalidIssuer()

    return claims
```

## mTLS (Mutual TLS)

### Principe
```
Client                          Server
  │                               │
  │  1. ClientHello               │
  │───────────────────────────────▶
  │                               │
  │  2. ServerHello +             │
  │     ServerCertificate +       │
  │     CertificateRequest        │
  │◀───────────────────────────────
  │                               │
  │  3. ClientCertificate +       │
  │     ClientKeyExchange         │
  │───────────────────────────────▶
  │                               │
  │  4. Connexion établie         │
  │◀─────────────────────────────▶│
```

### Usage
- Communication service-to-service
- APIs critiques (paiement, données sensibles)
- Zero-trust architecture

## Cas d'usage assurance

### Partenaires B2B (courtiers)
```
Courtier → Gateway
Headers:
  X-API-Key: sk_live_courtier42_abc123

Gateway:
  - Vérifier la clé
  - Identifier le courtier
  - Appliquer son rate limit
  - Logger l'accès
```

### Application mobile client
```
1. Utilisateur se connecte via OAuth
2. App reçoit un access_token JWT
3. Chaque requête inclut le token:
   Authorization: Bearer eyJhbG...

4. Gateway valide le JWT et extrait:
   - user_id
   - roles
   - permissions
```

### Communication inter-services
```
Quote Engine → Policy Admin
  - mTLS pour l'authentification
  - JWT pour le contexte utilisateur propagé

Headers:
  Authorization: Bearer <user_jwt>
  X-Service-Name: quote-engine
  X-Request-ID: abc123
```

## Bonnes pratiques

### Rotation des secrets
```
# API Keys
- Rotation annuelle minimum
- Double-clé pendant la transition
- Notification avant expiration

# JWT Secrets
- Rotation trimestrielle
- Supporter plusieurs clés actives
```

### Stockage sécurisé
```
# Mauvais
config.py:
  API_KEY = "sk_live_secret123"

# Bon
- Variables d'environnement
- Vault/Secret Manager
- Jamais dans le code
```

### Validation stricte
```python
# Toujours vérifier:
- Signature
- Expiration
- Issuer
- Audience
- Permissions requises
```

## Anti-patterns

1. **Secrets dans le code** : Compromis si code leaké
2. **Tokens sans expiration** : Risque si compromis
3. **Pas de rotation** : Accumulation de risque
4. **Validation partielle** : Vérifier signature mais pas expiration
5. **Logs avec tokens** : Exposition des secrets
