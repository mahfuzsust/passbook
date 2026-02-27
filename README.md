# 🔐 PassBook (Terminal Password Manager)

![Downloads](https://img.shields.io/github/downloads/mahfuzsust/passbook/total)

PassBook is a terminal-based password manager built in Go. It stores your vault locally as encrypted files, provides a TUI for browsing/editing entries, and includes built-in TOTP generation with a live countdown.

## ✨ Features

- Local encryption: Entries and attachments are encrypted at rest using AES-256-GCM.
- Two-factor authentication: After login, an additional 6-digit PIN or TOTP authenticator app verification is required. Configurable on first use with QR code setup for authenticator apps.
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
- Vault params: `<dataDir>/.vault_params`
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
- `.secret` — vault-local KDF configuration and 2FA settings (encrypted protobuf)
- `.vault_params` — vault parameters: salt, Argon2id cost, KDF/cipher identifiers (protobuf)

Notes:

- The `*.pb` extension indicates Protocol Buffer binary format; the content is encrypted protobuf data.
- Entry filenames are based on the entry Title.
- If you create a new entry with a duplicate title, you'll be prompted to Replace or Add Suffix.

## 🔐 Security architecture

For the full security architecture — encryption details, key derivation hierarchy, HMAC commit tags, tamper detection, automatic migrations, and design rationale — see **[SECURITY.md](SECURITY.md)**.

**Summary:**

- **Encryption**: AES-256-GCM with 12-byte random nonces.
- **Key derivation**: Argon2id → HKDF-SHA256 hierarchy producing separate master and vault keys from a single password.
- **Two-factor authentication**: 6-digit numeric PIN (verified via HMAC-SHA256) or TOTP authenticator app. Configuration is stored encrypted inside `.secret` and preserved across password changes.
- **Wrong-password detection**: An HMAC commit tag (`HMAC-SHA256(masterKey, random_nonce)`) stored inside `.secret` provides explicit detection.
- **Tamper detection**: Three layers — GCM AAD binding, SHA-256 hash verification, and HMAC commit tag.
- **Auto-migration**: Legacy vaults, weaker Argon2id parameters, old HKDF purpose strings, and missing commit tags are all upgraded transparently on login.
- **Key zeroization**: All ephemeral keys are wiped from memory as soon as they are no longer needed.
- **Serialization**: Vault metadata (`.secret`, `.vault_params`) uses Protocol Buffers for compact, deterministic serialization.


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

This repo includes a simple command to regenerate Go code from the `.proto` files in `internal/pb/`.

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
