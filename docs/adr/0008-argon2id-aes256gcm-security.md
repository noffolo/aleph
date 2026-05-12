# ADR-0008: Argon2id + AES-256-GCM for Security

## Status

Accepted

## Context

Aleph Data OS manages several categories of sensitive credentials that require cryptographic protection:

1. **User passwords**: Stored for authentication against local accounts. Must use a memory-hard password hashing function resistant to GPU/ASIC brute-force attacks.
2. **API keys**: System-level and user-level API keys that authenticate programmatic access. Must be encrypted at rest with authenticated encryption and proper key management.
3. **Agent credentials**: Tool credentials (email passwords, third-party API tokens) used during sandbox execution. Must be encrypted at rest.

The previous implementation used SHA-256 for password hashing — insufficient for credential storage by modern standards. A memory-hard function was required.

Candidates for password hashing:

| Algorithm | Memory-hard | GPU Resistant | Go stdlib | Recommended |
|-----------|-------------|---------------|-----------|-------------|
| bcrypt | No | Weak | Yes (x/crypto) | Legacy only |
| scrypt | Yes | Moderate | Yes (x/crypto) | Acceptable |
| Argon2id | Yes (tunable) | Strong | Yes (x/crypto) | OWASP recommended |

For symmetric encryption, AES-256-GCM provides authenticated encryption with built-in integrity checking, preventing both decryption of ciphertext and undetected tampering.

## Decision

### Password Hashing: Argon2id

- All passwords hashed via `golang.org/x/crypto/argon2` using the Argon2id variant
- Parameters (OWASP recommended):
  - Memory: 64 MiB (64 * 1024 KiB)
  - Iterations: 3
  - Parallelism: 4 (matching CPU cores)
  - Salt length: 16 bytes (random, per-password)
  - Key length: 32 bytes
- Implementation at `internal/auth/` with `HashPassword()` and `VerifyPassword()` functions
- Legacy SHA-256 hashes rehashed on next successful login (downgrade-resistant upgrade path)

### Symmetric Encryption: AES-256-GCM

- All API keys encrypted via `internal/crypto/` using AES-256-GCM
- Key derived from `KEY_ENCRYPTION_KEY` environment variable (32 bytes, hex-encoded), never hardcoded
- Each encryption operation uses a unique 96-bit random nonce
- Nonces tracked to detect and prevent nonce reuse
- Implementation provides `Encrypt(plaintext)` and `Decrypt(ciphertext)` with automatic nonce management

### Storage
- Password hashes stored in PostgreSQL `users` table with `hash_algorithm` column (values: `argon2id`, `sha256_legacy`)
- Encrypted API keys stored in PostgreSQL `api_keys` table
- `KEY_ENCRYPTION_KEY` required at startup — application fatals if missing or malformed
- No plaintext credential storage anywhere in the database

## Consequences

### Positive
- Argon2id resists GPU and ASIC attacks significantly better than bcrypt or SHA-256
- AES-256-GCM provides authenticated encryption — tampering detected
- Unique nonce per encryption prevents reuse attacks and nonce collision
- Migration path from legacy SHA-256 hashes (rehash on login)
- All crypto primitives from `golang.org/x/crypto` — no external crypto dependencies

### Negative
- Argon2id memory/CPU cost higher than bcrypt — login authentication is slower (~100-200ms vs ~50-100ms)
- Key management (`KEY_ENCRYPTION_KEY` rotation) adds operational complexity — key change requires re-encrypting all stored secrets
- 96-bit nonce space is large but nonce reuse is catastrophic (GCM loses all security on nonce reuse)
- Legacy SHA-256 hashes in database are potential attack surface until all users have logged in post-migration

## Compliance

- All user passwords hashed via `internal/auth/` using Argon2id
- All API keys encrypted via `internal/crypto/` using AES-256-GCM
- No plaintext credential storage in database or log output
- `KEY_ENCRYPTION_KEY` environment variable required — verified at startup via config validation
- Legacy SHA-256 hashes marked with `hash_algorithm = 'sha256_legacy'` in database
- Logging must never emit credential values (checked via `internal/log/` sanitization)

## Notes

- OWASP Argon2id recommendations: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- AES-GCM nonce reuse catastrophically breaks security — nonce uniqueness enforced via tracked nonces
- Related ADRs: ADR-0010 (RBAC + JWT Bearer Authentication)
