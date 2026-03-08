# 🔐 Security Architecture

This document describes PassBook's cryptographic design, authentication flow, database encryption, and security properties.

---

## Table of Contents

- [Database Encryption (SQLCipher)](#database-encryption-sqlcipher)
- [Master Password](#master-password)
- [Password Change](#password-change)
- [Password Strength Requirements](#password-strength-requirements)
- [Two-Factor Authentication (2FA)](#two-factor-authentication-2fa)
- [Clipboard Security](#clipboard-security)
- [File Permissions](#file-permissions)
- [Key Zeroization](#key-zeroization)
- [Authentication Flow](#authentication-flow)
- [Database Schema Security](#database-schema-security)
- [Design Rationale](#design-rationale)

---

## Database Encryption (SQLCipher)

PassBook stores all vault data in a single SQLite database encrypted by [SQLCipher](https://www.zetetic.net/sqlcipher/).

- **Library**: `go-sqlcipher` v4 (`github.com/mutecomm/go-sqlcipher/v4`).
- **Cipher**: AES-256-CBC with HMAC-SHA512 page-level authentication (SQLCipher defaults).
- **Key derivation**: SQLCipher internally uses PBKDF2-HMAC-SHA512 to derive the encryption key from the master password.
- **Page-level MAC**: Every database page is independently authenticated. Tampered pages are detected on read, preventing silent data corruption.
- **Transparent encryption**: All reads and writes go through the SQLCipher layer — data is never stored in plaintext on disk.
- **WAL mode**: The database operates in Write-Ahead Logging mode (`PRAGMA journal_mode = WAL`) for performance. WAL and shared-memory files (`passbook.db-wal`, `passbook.db-shm`) are also encrypted by SQLCipher.
- **Foreign keys**: Enforced via `PRAGMA foreign_keys = ON` for referential integrity.

### Database location

```
<dataDir>/passbook.db
```

By default, `<dataDir>` is `~/.passbook/data`. The path is configurable via `~/.passbook/config.json`.

---

## Master Password

The master password is the sole cryptographic key protecting the vault:

- Passed to SQLCipher via `PRAGMA key` (URL-encoded in the connection DSN).
- On login, PassBook queries `sqlite_master` to verify the key is correct. A wrong password produces a "wrong database key or corrupt database" error.
- The password is never stored on disk.

---

## Password Change

Password change uses SQLCipher's `PRAGMA rekey`:

1. The current password is verified by opening a separate connection and querying `sqlite_master`.
2. The new password's strength is checked — must be at least "Good".
3. `PRAGMA rekey` re-encrypts the entire database in place with the new password.
4. Existing 2FA configuration (PIN key, TOTP secret) is unaffected because it lives inside the encrypted database — re-encryption is transparent.

---

## Password Strength Requirements

PassBook enforces password quality on vault creation and password change. Passwords rated below "Good" are rejected.

The scoring system is aligned with NIST SP 800-63B guidelines:

| Factor | Points | Weight |
| --- | --- | --- |
| Length | 0–45 | Primary factor |
| Character-class variety | 0–30 | Secondary factor |
| Uniqueness ratio | 0–15 | Bonus for non-repetitive characters |
| Common/breached password penalty | -25 | Checked against a blocklist |
| Sequential character penalty | -8 to -15 | Detects runs like `abc`, `123` |
| Repeated character penalty | -5 to -12 | Detects runs like `aaaa` |

**Hard rule**: Passwords of 8 characters or fewer are always rated "Weak" regardless of other factors.

| Level | Score range | Vault creation |
| --- | --- | --- |
| Strong | 70–100 | Allowed |
| Good | 45–69 | Allowed (minimum required) |
| Fair | 25–44 | Rejected |
| Weak | 0–24 | Rejected |

---

## Two-Factor Authentication (2FA)

After master password verification, PassBook requires a second factor before granting access. On first login, the user chooses one of two modes:

### 6-Digit PIN

- A 32-byte random `pin_key` is generated via Go's `crypto/rand`.
- The PIN is verified using `HMAC-SHA256("passbook:pin:" + PIN, pin_key)`. The domain-prefixed message prevents cross-protocol confusion.
- Both `pin_key` and the resulting `pin_verify_tag` (hex-encoded) are stored in the `pin_config` table.
- Verification uses `crypto/hmac.Equal` for constant-time comparison, preventing timing side-channel attacks.

### TOTP (Authenticator App)

- A random TOTP secret is generated using `github.com/pquerna/otp/totp`.
- A QR code is rendered in the terminal for scanning with authenticator apps (Google Authenticator, Authy, etc.), along with the text secret.
- TOTP codes are validated with `totp.ValidateCustom` using SHA1, 6 digits, 30-second period, and skew of 2 (allowing ±60 seconds for clock drift).
- The TOTP shared secret is stored in the `pin_config` table.

### Security Properties

- **Application-layer gate**: The PIN/TOTP is an authentication factor, not a cryptographic key derivation input. A 6-digit PIN has only ~20 bits of entropy and cannot provide meaningful protection against offline attacks. The master password (via SQLCipher's PBKDF2) remains the sole cryptographic security boundary.
- **Encrypted storage**: All 2FA configuration (`pin_key`, `pin_verify_tag`, `totp_secret`) is stored inside the SQLCipher-encrypted database. An attacker without the master password cannot access or tamper with the 2FA settings.
- **Preserved across password changes**: Since 2FA data lives inside the encrypted database, `PRAGMA rekey` transparently re-encrypts it along with everything else.

---

## Clipboard Security

When sensitive values are copied to the clipboard (passwords, card numbers, TOTP codes):

1. The value is written to the system clipboard.
2. A background goroutine waits 30 seconds.
3. After 30 seconds, the clipboard is read — if it still contains the copied value, it is cleared.
4. If the user has copied something else in the meantime, the clipboard is left untouched.

This limits the exposure window for sensitive data on the system clipboard.

---

## File Permissions

PassBook applies restrictive file permissions:

| Path | Permission | Notes |
| --- | --- | --- |
| `~/.passbook/` | `0700` | Config directory |
| `~/.passbook/config.json` | `0600` | Config file |
| `<dataDir>/` | `0700` | Database directory |
| `<dataDir>/passbook.db` | `0600` | Best-effort `chmod` after open |

---

## Key Zeroization

- **PIN key**: The 32-byte `pin_key` used for HMAC computation is wiped via `WipeBytes()` (overwriting every byte with `0x00`) when no longer needed.
- **Limitation**: Go's garbage collector may copy byte slices during memory management. Zeroing the authoritative slice is a best-effort defense that limits the exposure window but cannot guarantee complete erasure from process memory.

---

## Authentication Flow

```
User launches PassBook
        │
        ▼
  Enter master password
        │
        ▼
  Open SQLCipher database
  (PRAGMA key = password)
        │
        ├── Fails → "Wrong password."
        │
        ▼
  Check if vault is new
  (no entries + no pin_config)
        │
        ├── New vault ──────────────┐
        │   Check password strength │
        │   (must be ≥ Good)        │
        │         │                 │
        │         ├── Too weak → reject, close DB
        │         │
        │         ▼
        │   PIN/TOTP setup
        │         │
        ├── Existing vault ─────────┐
        │   Read pin_config         │
        │         │                 │
        │         ├── Has config → verify PIN or TOTP
        │         │
        │         ├── No config → PIN/TOTP setup
        │         │
        ▼         ▼
     Enter main TUI
```

---

## Database Schema Security

The database schema enforces data integrity constraints:

- **Unique folder names**: `folders.name` has a `UNIQUE` constraint.
- **Unique entries per folder**: A unique index on `(folder_id, title)` prevents duplicate entry titles within a folder.
- **Cascading deletes**: `password_history` and `attachments` use `ON DELETE CASCADE` foreign keys — deleting an entry automatically removes its history and attachments.
- **Single-row PIN config**: `pin_config.id` has a `CHECK (id = 1)` constraint ensuring only one 2FA configuration row exists.
- **Single connection**: `db.SetMaxOpenConns(1)` prevents concurrent access issues with SQLite.

---

## Design Rationale

| # | Principle | Explanation |
|---|-----------|-------------|
| 1 | **Single encrypted file** | A single SQLCipher database simplifies backup, sync, and portability. Copy `passbook.db` to a new machine and it just works. |
| 2 | **SQLCipher defaults** | SQLCipher's vetted defaults (AES-256-CBC, HMAC-SHA512, PBKDF2) provide strong encryption without requiring custom cryptographic code. |
| 3 | **No custom crypto** | By delegating encryption entirely to SQLCipher, PassBook avoids the risks of implementing cryptographic primitives in application code. |
| 4 | **Password strength enforcement** | Weak passwords are rejected at vault creation and password change, ensuring the SQLCipher key has sufficient entropy. |
| 5 | **Two-factor authentication** | A second factor (PIN or TOTP) provides defense-in-depth at the application layer, protecting against shoulder surfing and casual access. |
| 6 | **Constant-time PIN verification** | HMAC comparison uses `crypto/hmac.Equal` to prevent timing side-channel attacks. |
| 7 | **Domain-prefixed HMAC message** | The PIN HMAC input is prefixed with `"passbook:pin:"` to prevent cross-protocol confusion if the key material were reused. |
| 8 | **Clipboard hygiene** | Automatic 30-second clipboard clearing limits the exposure window for sensitive values. |
| 9 | **Restrictive file permissions** | Config and database files are readable only by the owner. |
| 10 | **Portability** | Everything needed to unlock (except the password) lives in a single file under `<dataDir>`. |

---

## Config File

Located at `~/.passbook/config.json`.

| Field | Description |
| --- | --- |
| `data_dir` | Path to the vault directory (default: `~/.passbook/data`). Supports `~/` expansion. |
