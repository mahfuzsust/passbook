# 🔐 PassBook (Terminal Password Manager)

![Downloads](https://img.shields.io/github/downloads/mahfuzsust/passbook/total)

PassBook is a terminal-based password manager built in Go. It stores your vault locally as encrypted files, provides a TUI for browsing/editing entries, and includes built-in TOTP generation with a live countdown.

## ✨ Features

- Local encryption: Entries and attachments are encrypted at rest using AES-256-GCM.
- Entry types: Logins, Cards, Notes, and Files.
- Built-in TOTP: Generates 6-digit codes for Login entries with a live progress bar.
- Smart clipboard handling:
  - Copying sensitive values clears the clipboard after 30 seconds if it still contains the copied value.
  - Copying non-sensitive values shows a quick status.
- Password history: Login entries keep prior passwords + timestamps when the password changes.
- Password generator: Generate a password and insert it into the editor.
- Change master password: Re-encrypts all entries and attachments with a new password and fresh salt.
- Import from Bitwarden: Import your vault from a Bitwarden JSON export via the CLI.
- Import from 1Password: Import your vault from a 1Password `.1pux` export via the CLI.
- Import from LastPass: Import your vault from a LastPass CSV export via the CLI.
- Cloud-sync friendly: Point the data directory at iCloud Drive / Dropbox / etc.
- Responsive layout: Left pane stays ~30% width and right pane ~70% width as the terminal resizes.

## 🚀 Installation

### Option A: Homebrew (macOS/Linux)

```bash
brew install mahfuzsust/tap/passbook
```


### Option B: Download from GitHub Releases (recommended)

1) Open the latest release page and copy the download URL for your OS/arch:

https://github.com/mahfuzsust/passbook/releases/latest

2) Download + install (macOS/Linux)

Update the version (`vX.Y.Z`) and OS/arch in the URL, then run:

```bash
curl -fL https://github.com/mahfuzsust/passbook/releases/download/vX.Y.Z/passbook_vX.Y.Z_darwin_arm64.tar.gz -o passbook.tar.gz

tar -xzf passbook.tar.gz

chmod +x passbook

sudo cp -f passbook /usr/local/bin/passbook
```

Verify:

```bash
passbook --help
passbook --version
```

Notes:
- For Windows, download the `.zip` asset and place `passbook.exe` somewhere on your `PATH`.
- For Linux assets, the archive name will include `linux_<arch>`.
- For Intel macOS, use `darwin_amd64`.

### Option C: Build from source

Prerequisites:
- Go (see `go.mod`)

Clone and build using the new `cmd/` structure:

```bash
git clone https://github.com/mahfuzsust/passbook.git
cd passbook

go build -o passbook ./cmd/passbook
./passbook
```

Or install into your Go bin:

```bash
go install ./cmd/passbook
passbook
```

## ▶️ Usage

Run:

```bash
passbook
```

On first run, PassBook creates:

- Config: `~/.passbook/config.json` (stores your `data_dir`)
- Default vault: `~/.passbook/data`
- Vault secret: `<dataDir>/.secret`
- Root salt: `<dataDir>/.root_salt`
- Attachments: `<dataDir>/_attachments`

## ☁️ iCloud sync

PassBook stores the vault under `data_dir` from `~/.passbook/config.json`. To sync via iCloud Drive, set `data_dir` to something like:

`~/Library/Mobile Documents/com~apple~CloudDocs/PassBook`

```bash
sed -i '' 's|"data_dir":[[:space:]]*"[^"]*"|"data_dir": "~/Library/Mobile Documents/com~apple~CloudDocs/PassBook"|' ~/.passbook/config.json
```

Paths starting with `~/` are expanded.

## 📥 Importing

PassBook can import entries from external password managers without launching the TUI. You will be prompted for your master password.

### Bitwarden (JSON)

```bash
passbook --import bitwarden /path/to/bitwarden_export.json
```

Export your Bitwarden vault as **unencrypted JSON** (`Settings → Export Vault → File format: .json`).

Item type mapping:
- Type 1 (Login) → Login
- Type 2 (Secure Note) → Note
- Type 3 (Card) → Card

Password history and custom fields are preserved.

### 1Password (.1pux)

```bash
passbook --import 1password /path/to/1password_export.1pux
```

Export your 1Password vault via `File → Export → 1PUX format`.

Category mapping:
- `001` (Login) → Login
- `002` (Credit Card) → Card
- `003` (Secure Note) → Note
- `006` (Document) → Note
- Other categories → Note (to avoid data loss)

TOTP secrets, extra section fields, and cardholder names are preserved.

### LastPass (CSV)

```bash
passbook --import lastpass /path/to/lastpass_export.csv
```

Export your LastPass vault via `Account Options → Advanced → Export`.

- Standard entries are imported as Login entries.
- Secure Notes (URL = `http://sn`) are imported as Note entries.
- TOTP secrets and extra/notes fields are preserved.

### Common behavior

- Duplicate titles are handled by appending a numeric suffix (e.g. `GitHub_1.pb`).
- Each entry is encrypted and written to the vault.
- **Delete the export file after importing.**

## 🗂️ Vault layout (on disk)

Inside `<dataDir>` you'll see:

- `logins/` — encrypted protobuf entries stored as `*.pb`
- `cards/` — encrypted protobuf entries stored as `*.pb`
- `notes/` — encrypted protobuf entries stored as `*.pb`
- `files/` — encrypted protobuf entries stored as `*.pb` (plus attachment metadata)
- `_attachments/` — encrypted attachment blobs keyed by attachment ID
- `.secret` — vault-local KDF configuration (salt + Argon2id parameters)
- `.root_salt` — 32-byte random salt for Argon2id root key derivation (plaintext)

Notes:

- The `*.pb` extension indicates Protocol Buffer binary format; the content is encrypted protobuf data.
- Entry filenames are based on the entry Title.
- If you create a new entry with a duplicate title, you'll be prompted to Replace or Add Suffix.

## 🔐 Security architecture

### Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode) providing authenticated encryption.
- **Nonce**: Random nonce generated per encryption operation (12 bytes with Go's `cipher.NewGCM`).
- **Format**: Nonce is prepended to ciphertext for storage.
- **Key size**: 256-bit (32-byte) keys.

### Key derivation (HKDF-based hierarchy)

PassBook derives all encryption keys from a single master password using a three-step hierarchy:

```
root_key   = Argon2id(password, random_salt)   ← stored in <dataDir>/.root_salt
master_key = HKDF-SHA256(root_key, "master")   ← encrypts .secret
vault_key  = HKDF-SHA256(root_key, "vault")    ← encrypts entries & attachments
```

#### Step 1: Root Key

- **KDF**: Argon2id
  - Time: 6
  - Memory: 256 MB
  - Threads: 4
  - Salt: cryptographically random 32-byte salt (unique per vault, stored in `<dataDir>/.root_salt`)
- **Input**: your master password + random salt
- **Output**: 32-byte root key (ephemeral — never stored)

#### Step 2: Master Key

- **Purpose**: Encrypt/decrypt the vault-local `.secret` file.
- **KDF**: HKDF-SHA256 with info string `"master"`.
- **Input**: root key
- **Output**: 32-byte master key (ephemeral — never stored)

#### Step 3: Vault Key

- **Purpose**: Encrypt/decrypt all vault entry files (`*.pb`) and attachment blobs.
- **KDF**: HKDF-SHA256 with info string `"vault"`.
- **Input**: root key
- **Output**: 32-byte vault key (ephemeral — never stored)

### The `.secret` file

Located at `<dataDir>/.secret`.

- The file contains a small JSON document (versioned schema) with:
  - `salt` (16 bytes)
  - Argon2id parameters (`time`, `memory_kb`, `threads`)
  - metadata like `kdf` ("argon2id") and `key_len` (32)
- The JSON is **encrypted at rest** using AES-256-GCM with the **master key**.
- Serves as a password-correctness check: if decryption of `.secret` succeeds, the password is correct.

**Portability**: The vault is fully self-contained—moving `<dataDir>` moves everything needed to unlock it (`.root_salt`, `.secret`, entries, and attachments). Only the master password is external.

**Important**: Don't delete `<dataDir>/.secret` or `<dataDir>/.root_salt`. Without them, existing encrypted data becomes undecryptable.

### Config file

Located at `~/.passbook/config.json`.

- `data_dir`: Path to the vault directory.
- `is_migrated`: Whether the vault has been migrated to the new HKDF scheme (`true`/`false`).

### Entries and attachments

All vault data is encrypted with the vault key:

- **Entry files** (`*.pb`): protobuf bytes encrypted with AES-256-GCM.
- **Attachment files** (`_attachments/`): raw bytes encrypted with AES-256-GCM.
- **Format**: `[nonce][ciphertext+tag]` where nonce is prepended.

### Automatic migration from legacy scheme

Older versions of PassBook used a fixed UUID string as the Argon2id salt for master key derivation, with a separate per-vault Argon2id pass for the vault key. The current version automatically migrates existing vaults on first login:

1. The legacy master key is derived using the old fixed salt to verify the password.
2. A new random 32-byte salt is generated.
3. New master and vault keys are derived using the HKDF hierarchy.
4. The `.secret` file is re-encrypted with the new master key.
5. All entries and attachments are re-encrypted with the new vault key.
6. The new salt is saved to `<dataDir>/.root_salt` and `is_migrated` is set in `config.json`.

If migration fails (e.g. disk error), the vault falls back to the legacy scheme for that session and retries on the next login. Changing the master password also always uses the new scheme.

Legacy support can be removed entirely by setting `supportLegacy = false` in `internal/crypto/crypto.go` and deleting all code blocks marked with `// --- BEGIN supportLegacy ---` / `// --- END supportLegacy ---`.

### Why this design?

1. **Random salt**: Each vault gets a unique random salt instead of a fixed one, preventing rainbow table attacks across vaults.
2. **Key separation**: HKDF produces independent master and vault keys from a single Argon2id pass — one slow KDF call instead of two.
3. **Defense in depth**: `.secret` is encrypted with a separate key from vault data; compromising one doesn't directly expose the other.
4. **Portability**: Everything needed to unlock (except the password) lives under `<dataDir>`. Copy the directory to a new machine and it just works.



## ⌨️ Keyboard shortcuts

### Main screen

| Shortcut | Action |
| --- | --- |
| `Ctrl+A` | Create a new entry |
| `Ctrl+E` | Edit selected entry |
| `Ctrl+D` | Delete selected entry |
| `Ctrl+F` | Focus search |
| `Ctrl+P` | Change master password |
| `Ctrl+Q` | Quit |
| `Esc` | Focus vault tree |

### Viewer actions

Buttons are compact ASCII labels:

- `cp` = copy
- `vw` = view/toggle visibility
- `his` = history
- `open` = open URL

Viewer behavior:

- Login:
  - Username shows `cp` only when a username exists.
  - Password row shows `vw`, `cp`, `his` only when a password exists.
  - Link row shows `open` + `cp` only when a link exists.
  - TOTP shows `cp` only when a TOTP secret exists.
- Card:
  - Number shows `vw` + `cp`.
- Notes:
  - Notes header shows `cp` only when notes exist.
- File:
  - Selecting an attachment downloads it to your Downloads folder.

### Modals / editor

| Context | Shortcut | Action |
| --- | --- | --- |
| Login screen | `Enter` | Login |
| Editor | `Esc` | Close editor |
| File browser | `Esc` | Cancel file picker |
| Password generator | `Esc` | Close generator |
| History | `Esc` | Close history |

## Protobuf code generation

This repo includes a simple command to regenerate `internal/pb/entry.pb.go` from `internal/pb/entry.proto`.

Prerequisites:

- `protoc` installed and on your `PATH`
- `protoc-gen-go` installed (example):

  ```sh
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  ```

Generate via Taskfile:

```sh
task gen:rpc
```

Or generate via Go:

```sh
go generate ./...
```

## 🧰 Built with

- tview: https://github.com/rivo/tview
- tcell: https://github.com/gdamore/tcell
- otp: https://github.com/pquerna/otp
- clipboard: https://github.com/atotto/clipboard
- argon2 (x/crypto): https://pkg.go.dev/golang.org/x/crypto/argon2
- hkdf (x/crypto): https://pkg.go.dev/golang.org/x/crypto/hkdf
