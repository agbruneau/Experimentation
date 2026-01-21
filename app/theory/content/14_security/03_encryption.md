# Chiffrement et Protection des Données

## Résumé

Le **chiffrement** protège les données sensibles contre les accès non autorisés. On distingue le chiffrement **en transit** (données en mouvement) et **au repos** (données stockées).

## Points clés

- **En transit** : TLS/HTTPS entre les services
- **Au repos** : Chiffrement des bases de données et fichiers
- **Gestion des clés** : Stockage sécurisé, rotation régulière
- **Données sensibles** : Identifier et protéger (PII, financier)

## Chiffrement en transit

### TLS (Transport Layer Security)

```
┌────────┐          TLS           ┌────────┐
│ Client │ ◄═══════════════════▶  │ Server │
└────────┘    Connexion chiffrée  └────────┘
```

### Configuration recommandée

```
# Versions
TLS 1.2 minimum
TLS 1.3 recommandé

# Cipher suites (TLS 1.3)
TLS_AES_256_GCM_SHA384
TLS_CHACHA20_POLY1305_SHA256
TLS_AES_128_GCM_SHA256

# À désactiver
TLS 1.0, 1.1, SSL (obsolètes)
Cipher suites faibles (RC4, DES, MD5)
```

### HTTPS partout

```
# Mauvais
http://api.insurance.com/policies

# Bon
https://api.insurance.com/policies

# Forcer HTTPS (header)
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### mTLS pour service-to-service

```
Quote Engine ◄═══mTLS═══▶ Policy Admin

Les deux parties présentent leur certificat.
Authentification mutuelle + chiffrement.
```

## Chiffrement au repos

### Niveaux de chiffrement

| Niveau | Description | Usage |
|--------|-------------|-------|
| Full Disk | Disque entier chiffré | VM, serveurs |
| Database | Chiffrement transparent (TDE) | Bases de données |
| Column | Colonnes spécifiques | Données sensibles |
| Application | Chiffré par l'application | Contrôle fin |

### Chiffrement applicatif

```python
from cryptography.fernet import Fernet

# Générer une clé
key = Fernet.generate_key()
cipher = Fernet(key)

# Chiffrer
sensitive_data = "FR7612345678901234567890123"
encrypted = cipher.encrypt(sensitive_data.encode())
# b'gAAAAABl...'

# Déchiffrer
decrypted = cipher.decrypt(encrypted).decode()
# "FR7612345678901234567890123"
```

### Stockage en base de données

```sql
-- Colonne chiffrée
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    name VARCHAR(100),
    email_encrypted BYTEA,  -- Email chiffré
    ssn_encrypted BYTEA,    -- Numéro sécu chiffré
    email_hash VARCHAR(64)  -- Hash pour recherche
);

-- Recherche par hash
SELECT * FROM customers
WHERE email_hash = sha256('client@email.com');
```

## Gestion des clés

### Hiérarchie des clés

```
┌─────────────────────────────────────────┐
│       Master Key (KEK)                   │
│       (Key Encryption Key)               │
│       Stockée dans HSM/Vault             │
└────────────────┬────────────────────────┘
                 │
     ┌───────────┼───────────┐
     ▼           ▼           ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│ DEK 1   │ │ DEK 2   │ │ DEK 3   │
│(Data Key)│ │(Data Key)│ │(Data Key)│
│ Clients │ │ Polices │ │ Sinistres│
└─────────┘ └─────────┘ └─────────┘
```

### Rotation des clés

```python
# Processus de rotation
async def rotate_encryption_key():
    # 1. Générer nouvelle clé
    new_key = generate_key()
    store_key(new_key, version="v2")

    # 2. Rechiffrer les données (progressif)
    async for record in get_all_encrypted_records():
        # Déchiffrer avec ancienne clé
        plain = decrypt(record.data, key_version="v1")
        # Rechiffrer avec nouvelle clé
        new_encrypted = encrypt(plain, key_version="v2")
        await update_record(record.id, new_encrypted, key_version="v2")

    # 3. Marquer ancienne clé comme obsolète
    mark_key_deprecated("v1")
```

### Stockage des clés

```
# Mauvais
config.py:
    ENCRYPTION_KEY = "super_secret_key"

# Bon
- AWS KMS / Azure Key Vault / HashiCorp Vault
- HSM (Hardware Security Module)
- Variables d'environnement (minimum)
```

## Données sensibles en assurance

### Classification

| Type | Exemples | Protection requise |
|------|----------|-------------------|
| PII | Nom, adresse, téléphone | Chiffrement colonne |
| Financier | IBAN, carte bancaire | Chiffrement + tokenization |
| Santé | Dossier médical | Chiffrement + accès restreint |
| Identité | SSN, permis, passeport | Chiffrement + hash |

### Masquage pour les logs

```python
def mask_sensitive(data):
    """Masque les données sensibles pour les logs."""
    masked = data.copy()

    # Email
    if "email" in masked:
        email = masked["email"]
        masked["email"] = email[0] + "***@***" + email.split("@")[1][-4:]

    # IBAN
    if "iban" in masked:
        iban = masked["iban"]
        masked["iban"] = iban[:4] + "****" + iban[-4:]

    # Numéro de téléphone
    if "phone" in masked:
        masked["phone"] = "****" + masked["phone"][-4:]

    return masked

# Log sécurisé
log.info("Customer created", customer=mask_sensitive(customer))
# {"email": "j***@***.com", "iban": "FR76****7890", "phone": "****1234"}
```

### Tokenization

```
Données réelles                    Tokens
┌──────────────────┐              ┌──────────────────┐
│ IBAN: FR7612345  │  ◄──────▶   │ IBAN: tok_abc123 │
│ CC: 4111...1111  │              │ CC: tok_xyz789   │
└──────────────────┘              └──────────────────┘
        │                                  │
        ▼                                  ▼
   Vault sécurisé                   Application
   (accès restreint)               (manipulation OK)
```

## Cas d'usage assurance

### Stockage des données client

```python
class CustomerRepository:
    def __init__(self, vault):
        self.vault = vault

    async def create(self, customer):
        # Chiffrer les données sensibles
        encrypted_data = {
            "name": customer.name,  # Non chiffré
            "email_encrypted": self.vault.encrypt(customer.email),
            "email_hash": hash(customer.email),  # Pour recherche
            "ssn_encrypted": self.vault.encrypt(customer.ssn),
            "address_encrypted": self.vault.encrypt(
                json.dumps(customer.address)
            )
        }
        return await db.customers.insert(encrypted_data)

    async def get(self, customer_id):
        record = await db.customers.get(customer_id)
        # Déchiffrer
        return Customer(
            name=record["name"],
            email=self.vault.decrypt(record["email_encrypted"]),
            ssn=self.vault.decrypt(record["ssn_encrypted"]),
            address=json.loads(
                self.vault.decrypt(record["address_encrypted"])
            )
        )
```

### Communication avec partenaires

```
┌──────────────┐       TLS        ┌──────────────┐
│  Assureur    │◄═══════════════▶ │  Courtier    │
│              │                  │              │
│  Données     │                  │  Données     │
│  chiffrées   │   Payload JSON   │  déchiffrées │
│  au repos    │   chiffré E2E    │  en mémoire  │
└──────────────┘                  └──────────────┘
```

## Conformité

### RGPD (Europe)
- Droit à l'oubli : Suppression possible
- Minimisation : Ne collecter que le nécessaire
- Chiffrement : Mesure technique appropriée

### PCI-DSS (Cartes bancaires)
- Chiffrement des numéros de carte
- Pas de stockage du CVV
- Tokenization recommandée

## Bonnes pratiques

1. **Chiffrer par défaut** les données sensibles
2. **TLS partout** même en interne
3. **Rotation régulière** des clés
4. **Séparation** des clés et des données
5. **Audit** des accès aux clés
6. **Test de récupération** en cas de perte de clé

## Anti-patterns

1. **Clés dans le code** : Compromis si code leaké
2. **Algorithmes obsolètes** : MD5, SHA1, DES
3. **Même clé partout** : Compromis total si volée
4. **Pas de rotation** : Accumulation de risque
5. **Logs non masqués** : Exposition des secrets
