# 🔐 Security Architecture

This document describes PassBook's cryptographic design, key derivation hierarchy, vault file layout, and automatic migration mechanisms.

---

## Table of Contents

- [Encryption](#encryption)
- [Key Derivation (HKDF-based hierarchy)](#key-derivation-hkdf-based-hierarchy)
  - [Step 1: Root Key](#step-1-root-key)
  - [Step 2: Master Key](#step-2-master-key)
  - [Step 3: Vault Key](#step-3-vault-key)
- [HKDF Purpose Strings](#hkdf-purpose-strings)
- [The `.secret` File](#the-secret-file)
  - [HMAC Commit Tag (Wrong-Password Detection)](#hmac-commit-tag-wrong-password-detection)
  - [GCM AAD Binding](#gcm-aad-binding)
  - [Tamper Detection (Defense in Depth)](#tamper-detection-defense-in-depth)
- [The `.vault_params` File](#the-vault_params-file)
- [Entries and Attachments](#entries-and-attachments)
- [Key Zeroization](#key-zeroization)
- [Automatic Migrations](#automatic-migrations)
  - [Legacy Scheme Migration](#legacy-scheme-migration)
  - [Automatic Parameter Rehash](#automatic-parameter-rehash)
  - [HKDF Purpose String Migration](#hkdf-purpose-string-migration)
  - [Commit Tag Migration](#commit-tag-migration)
- [Design Rationale](#design-rationale)

---

## Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode) providing authenticated encryption.
- **Key size**: 256-bit (32-byte) keys.
- **Nonce**: 12-byte (96-bit) random nonce generated per encryption via a dedicated `generateNonce` function that:
  1. Reads exactly 12 bytes from Go's `crypto/rand` using `io.ReadFull` — a short read returns an error, never a truncated nonce.
  2. Rejects all-zero output as a sign of a catastrophic CSPRNG failure.
- **Collision bound**: With 96-bit random nonces, collision probability stays below 2⁻³² for up to ~2³² encryptions per key. PassBook issues a fresh vault key on every password change, keeping the per-key encryption count far below that threshold.
- **Format**: `[12-byte nonce][ciphertext + GCM tag]` — nonce is prepended to ciphertext for storage.

---

## Key Derivation (HKDF-based hierarchy)

PassBook derives all encryption keys from a single master password using a three-step hierarchy:

```
root_key   = Argon2id(password, salt, time, memory, threads)
                ↓
           HKDF-SHA256 expand
           ┌────────┴────────┐
master_key = HKDF(root_key,  vault_key = HKDF(root_key,
             master_purpose)              vault_purpose)
     │                              │
     ▼                              ▼
  encrypts .secret           encrypts entries
                             & attachments
```

### Step 1: Root Key

- **KDF**: Argon2id (password-hashing function resistant to GPU and side-channel attacks).
- **Parameters** (stored per-vault in `.vault_params`):
  - Time (iterations): `6` (recommended default)
  - Memory: `256 MB` (recommended default)
  - Threads: `4` (recommended default)
  - Salt: cryptographically random 32-byte value (unique per vault)
- **Input**: master password + stored parameters.
- **Output**: 32-byte root key (ephemeral — never stored, wiped immediately after HKDF expansion).

### Step 2: Master Key

- **Purpose**: Encrypt/decrypt the vault-local `.secret` file.
- **KDF**: HKDF-SHA256.
- **Info string**: configurable per-vault (see [HKDF Purpose Strings](#hkdf-purpose-strings)).
- **Input**: root key.
- **Output**: 32-byte master key (ephemeral — wiped after `.secret` is verified).

### Step 3: Vault Key

- **Purpose**: Encrypt/decrypt all vault entry files (`*.pb`) and attachment blobs.
- **KDF**: HKDF-SHA256.
- **Info string**: configurable per-vault (see [HKDF Purpose Strings](#hkdf-purpose-strings)).
- **Input**: root key.
- **Output**: 32-byte vault key (held in memory only while the application is running).

---

## HKDF Purpose Strings

The HKDF info strings used for key derivation are stored in `.vault_params`:

| Field              | New vaults            | Legacy vaults (empty field) |
| ------------------ | --------------------- | --------------------------- |
| `master_key_purpose` | `passbook:master:v1`  | `master`                    |
| `vault_key_purpose`  | `passbook:vault:v1`   | `vault`                     |

Storing the purpose strings in `.vault_params` ensures:

- **Forward compatibility**: Future versions can introduce `v2` purpose strings without breaking existing vaults.
- **Explicit domain separation**: The namespaced strings (`passbook:master:v1`) prevent accidental key reuse across different applications or protocol versions.
- **Safe migration**: Changing a purpose string triggers a full re-encryption (see [HKDF Purpose String Migration](#hkdf-purpose-string-migration)).

---

## The `.secret` File

Located at `<dataDir>/.secret`. This is the central trust anchor of the vault.

The file contains a JSON document (versioned schema) encrypted with AES-256-GCM using the **master key**:

| Field              | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| `version`          | Schema version (currently `1`)                               |
| `salt`             | 16-byte random salt (internal KDF use)                       |
| `time`             | Argon2id time parameter                                      |
| `memory_kb`        | Argon2id memory in KiB                                       |
| `threads`          | Argon2id parallelism                                         |
| `key_len`          | Derived key length (`32`)                                    |
| `kdf`              | KDF identifier (`"argon2id"`)                                |
| `vault_params_hash`| SHA-256 of `.vault_params` for tamper detection              |
| `commit_nonce`     | 32-byte random nonce for HMAC commit tag                     |
| `commit_tag`       | `hex(HMAC-SHA256(masterKey, commit_nonce))` for wrong-password detection |

### HMAC Commit Tag (Wrong-Password Detection)

`.secret` stores an **HMAC commit tag** that enables explicit wrong-password detection before attempting to decrypt vault entries:

```
commit_nonce = random 32 bytes (generated once, stored in .secret)
commit_tag   = hex(HMAC-SHA256(master_key, commit_nonce))
```

**How it works:**

1. When `.secret` is created or rewritten, a fresh 32-byte random nonce is generated and `HMAC-SHA256(masterKey, nonce)` is computed. Both the nonce and the resulting tag are stored inside the encrypted `.secret` JSON.
2. On login, after decrypting `.secret`, the stored `commit_nonce` is used to recompute the HMAC with the candidate master key. If the recomputed tag does not match `commit_tag`, the password is definitively wrong and `ErrWrongPassword` is returned.
3. Old vaults without a commit tag (empty `commit_tag` or missing `commit_nonce`) are accepted for backward compatibility and automatically migrated on next login (see [Commit Tag Migration](#commit-tag-migration)).

**Why a random nonce instead of a fixed message?**

- Each vault gets a unique HMAC message, providing domain separation — even two vaults with the same master password produce different commit tags.
- The nonce is stored inside the encrypted `.secret`, so it is never exposed to an attacker who doesn't already have the master key.

### GCM AAD Binding

The `.secret` file is encrypted with the SHA-256 hash of `.vault_params` as **GCM Additional Authenticated Data (AAD)**. This cryptographically binds the two files together:

- If an attacker modifies `.vault_params` (e.g. to weaken Argon2id parameters), GCM authentication fails immediately — decryption is rejected before any plaintext is produced.
- The AAD is the hash of the raw bytes on disk, not a re-serialized value, eliminating serialization round-trip issues.

### Tamper Detection (Defense in Depth)

The `vault_params_hash` field inside the decrypted `.secret` JSON is compared against the SHA-256 of the raw `.vault_params` bytes on disk. This provides a secondary check for vaults migrated from older versions that were encrypted without AAD.

---

## The `.vault_params` File

Located at `<dataDir>/.vault_params`.

Versioned JSON document (currently `version: 1`) storing all public vault parameters:

| Field              | Description                                         |
| ------------------ | --------------------------------------------------- |
| `version`          | Schema version (`1`)                                |
| `salt`             | 32-byte random Argon2id salt                        |
| `time`             | Argon2id time/iterations                            |
| `memory_kb`        | Argon2id memory in KiB                              |
| `threads`          | Argon2id parallelism                                |
| `kdf`              | KDF identifier (e.g. `"argon2id"`)                  |
| `cipher`           | Cipher identifier (e.g. `"aes-256-gcm"`)            |
| `master_key_purpose` | HKDF info string for master key derivation        |
| `vault_key_purpose`  | HKDF info string for vault key derivation         |

**Properties:**

- These values are **not secret** — they are public parameters needed to re-derive keys from the password.
- **Deterministic serialization**: Written as compact JSON with keys in Go's `json.Marshal` order. The same `VaultParams` value always produces identical bytes.
- A SHA-256 hash of the **exact bytes on disk** is stored inside the encrypted `.secret` file for integrity verification.

---

## Entries and Attachments

All vault data is encrypted with the **vault key**:

- **Entry files** (`*.pb`): Protocol Buffer bytes encrypted with AES-256-GCM.
- **Attachment files** (`_attachments/`): Raw bytes encrypted with AES-256-GCM.
- **Format**: `[12-byte nonce][ciphertext + GCM tag]` — nonce is prepended.

---

## Key Zeroization

All derived keys are ephemeral and zeroed from memory as soon as they are no longer needed:

| Key                | Lifetime                                                                 |
| ------------------ | ------------------------------------------------------------------------ |
| **Root key**       | Wiped immediately after HKDF expansion (inside `DeriveKeys()`)          |
| **Master key**     | Wiped after `.secret` is verified. Never stored long-term.               |
| **Vault key**      | Held in memory while the app runs. Wiped on exit and on password change. |
| **Migration keys** | Old and new master/vault keys wiped via `defer` after re-encryption.     |
| **Decrypted plaintext** | Wiped immediately after re-encryption with new key.                 |

Zeroization uses `WipeBytes()`, which overwrites every byte with `0x00`. This is a best-effort defense — Go's garbage collector may have copied the backing array, but zeroing the authoritative slice limits the exposure window.

---

## Automatic Migrations

PassBook performs several transparent migrations on login to keep vaults up to date with the latest security improvements. All migrations are atomic: if any step fails, the vault continues working with its current configuration and retries on the next login.

### Legacy Scheme Migration

Older versions of PassBook used a fixed UUID string as the Argon2id salt for master key derivation, with a separate per-vault Argon2id pass for the vault key.

**Migration steps:**

1. The legacy master key is derived using the old fixed salt to verify the password.
2. A new random 32-byte salt is generated.
3. New master and vault keys are derived using the HKDF hierarchy.
4. `.secret` is re-encrypted with the new master key.
5. All entries and attachments are re-encrypted with the new vault key.
6. New vault parameters are saved to `.vault_params` and `is_migrated` is set in `config.json`.

Legacy support can be removed by setting `supportLegacy = false` in `internal/crypto/crypto.go`.

### Automatic Parameter Rehash

When the recommended Argon2id constants are increased in a future release:

1. On login, PassBook compares stored parameters against recommended values.
2. If any stored parameter is strictly weaker, a rehash is triggered.
3. Old keys are derived with the stored (weaker) parameters.
4. New keys are derived with the recommended (stronger) parameters, keeping the same salt.
5. `.secret`, all entries, and attachments are re-encrypted with the new keys.
6. Updated parameters are saved to `.vault_params`.

### HKDF Purpose String Migration

Vaults created before purpose strings were introduced have empty `master_key_purpose` and `vault_key_purpose` fields, causing HKDF to use the legacy fixed strings (`"master"` / `"vault"`).

**Migration steps:**

1. Old keys are derived with the legacy purpose strings.
2. New keys are derived with the versioned purpose strings (`"passbook:master:v1"` / `"passbook:vault:v1"`).
3. `.secret` is re-encrypted with the new master key.
4. All entries and attachments are re-encrypted with the new vault key.
5. Updated purpose strings are saved to `.vault_params`.

### Commit Tag Migration

Vaults created before the HMAC commit tag was introduced have no `commit_nonce` or `commit_tag` in `.secret`. On login:

1. `EnsureSecret` (or `EnsureKDFSecret` for legacy vaults) detects the missing commit tag.
2. `.secret` is rewritten with a fresh 32-byte random nonce and the corresponding `HMAC-SHA256(masterKey, nonce)` tag.
3. Subsequent logins benefit from explicit wrong-password detection.

---

## Design Rationale

| # | Principle | Explanation |
|---|-----------|-------------|
| 1 | **Random salt** | Each vault gets a unique random salt, preventing rainbow table attacks across vaults. |
| 2 | **Key separation** | HKDF produces independent master and vault keys from a single Argon2id pass — one slow KDF call instead of two. |
| 3 | **Defense in depth** | `.secret` is encrypted with a separate key from vault data; compromising one doesn't directly expose the other. |
| 4 | **Explicit wrong-password detection** | An HMAC commit tag (`HMAC-SHA256(masterKey, random_nonce)`) stored inside `.secret` enables definitive wrong-password detection without relying solely on GCM authentication failure semantics. |
| 5 | **Portability** | Everything needed to unlock (except the password) lives under `<dataDir>`. Copy the directory to a new machine and it just works. |
| 6 | **Auto-rehash** | Argon2id parameters are stored per-vault. When recommended parameters increase, PassBook automatically re-derives keys and re-encrypts the vault — no user intervention needed. |
| 7 | **Key zeroization** | All ephemeral key material is wiped from memory as soon as it is no longer needed. |
| 8 | **Tamper detection (three layers)** | **(a)** GCM AAD binds `.secret` to `.vault_params` at the cryptographic level. **(b)** A SHA-256 hash inside `.secret` provides a secondary JSON-level check. **(c)** The HMAC commit tag verifies the master key itself is correct. |
| 9 | **Nonce uniqueness** | Every AES-256-GCM encryption uses a fresh 12-byte nonce from `crypto/rand`, validated for full read and non-zero output. A new vault key is issued on every password change, resetting the per-key nonce counter well within the birthday bound. |
| 10 | **Versioned purpose strings** | HKDF info strings are stored in `.vault_params` with explicit version tags (`passbook:master:v1`), enabling safe protocol evolution without breaking existing vaults. |
| 11 | **Transparent migration** | All security upgrades (salt, rehash, purpose strings, commit tag) are applied automatically on login with atomic rollback on failure. |

---

## Config File

Located at `~/.passbook/config.json`.

| Field        | Description                                                      |
| ------------ | ---------------------------------------------------------------- |
| `data_dir`   | Path to the vault directory.                                     |
| `is_migrated`| Whether the vault has been migrated to the HKDF scheme (`true`/`false`). |

---

## Vault Layout (on disk)

```
<dataDir>/
├── .vault_params          # Public vault parameters (JSON)
├── .secret                # Encrypted vault secret (AES-256-GCM)
├── logins/                # Encrypted login entries (*.pb)
├── cards/                 # Encrypted card entries (*.pb)
├── notes/                 # Encrypted note entries (*.pb)
├── files/                 # Encrypted file entries (*.pb)
└── _attachments/          # Encrypted attachment blobs
```

**Important**: Don't delete `.secret` or `.vault_params`. Without them, existing encrypted data becomes undecryptable.

