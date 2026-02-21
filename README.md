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
- Attachments: `<dataDir>/_attachments`

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

### Two-stage key derivation

PassBook uses a two-stage key derivation process:

#### Stage 1: Master Key

- **Purpose**: Encrypt/decrypt the vault-local `.secret` file.
- **KDF**: Argon2id with fixed parameters (hard-coded).
  - Time: 6
  - Memory: 256 MB
  - Threads: 4
  - Salt: fixed UUID string 
- **Input**: your master password
- **Output**: 32-byte master key

#### Stage 2: Vault Encryption Key

- **Purpose**: Encrypt/decrypt all vault entry files (`*.pb`) and attachment blobs.
- **KDF**: Argon2id with vault-specific parameters loaded from `.secret`.
  - Defaults (used when `.secret` is first created, and enforced if fields are missing):
    - Time: 6
    - Memory: 256 MB
    - Threads: 4
  - Salt: 16-byte random salt (per vault)
- **Input**: your master password + vault salt
- **Output**: 32-byte vault encryption key

### The `.secret` file

Located at `<dataDir>/.secret`.

- The file contains a small JSON document (versioned schema) with:
  - `salt` (16 bytes)
  - Argon2id parameters (`time`, `memory_kb`, `threads`)
  - metadata like `kdf` ("argon2id") and `key_len` (32)
- The JSON is **encrypted at rest** using AES-256-GCM with the **Stage 1 master key**.

**Portability**: The vault is self-contained—moving `<dataDir>` also moves `.secret`, entries, and attachments.

**Important**: Don't delete `<dataDir>/.secret`. Without it, the vault salt/KDF params are lost and existing encrypted data becomes undecryptable.

### Entries and attachments

All vault data is encrypted with the Stage 2 vault encryption key:

- **Entry files** (`*.pb`): protobuf bytes encrypted with AES-256-GCM.
- **Attachment files** (`_attachments/`): raw bytes encrypted with AES-256-GCM.
- **Format**: `[nonce][ciphertext+tag]` where nonce is prepended.

### Why two stages?

1. **Vault-specific KDF**: You can store per-vault salt/parameters without baking them into code.
2. **Defense in depth**: `.secret` is encrypted too; stealing it doesn't expose vault params without the password.
3. **Portability**: Everything needed to unlock (except the password) lives under `<dataDir>`.

## ☁️ Cloud syncing (example: iCloud Drive on macOS)

PassBook stores the vault under `data_dir` from `~/.passbook/config.json`. To sync via iCloud Drive, set `data_dir` to something like:

`~/Library/Mobile Documents/com~apple~CloudDocs/PassBook`

Paths starting with `~/` are expanded.

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
