# 🔐 PassBook (Terminal Password Manager)

![Downloads](https://img.shields.io/github/downloads/mahfuzsust/passbook/total)

PassBook is a terminal-based password manager built in Go. It stores your vault locally in a single encrypted SQLite database (via SQLCipher), provides a TUI for browsing/editing entries, and includes built-in TOTP generation with a live countdown.

## ✨ Features

- Local encryption: The entire vault is stored in a single SQLCipher-encrypted database file.
- Two-factor authentication: After login, an additional 6-digit PIN or TOTP authenticator app verification is required. Configurable on first use with QR code setup for authenticator apps.
- Entry types: Logins, Cards, Notes, and Files.
- Built-in TOTP: Generates 6-digit codes for Login entries with a live progress bar.
- Smart clipboard handling:
  - Copying sensitive values clears the clipboard after 30 seconds if it still contains the copied value.
  - Copying non-sensitive values shows a quick status.
- Password history: Login entries keep prior passwords + timestamps when the password changes.
- Password generator: Generate a password and insert it into the editor.
- Change master password: Re-encrypts the database with a new key via SQLCipher's `PRAGMA rekey`.
- Import from Bitwarden: Import your vault from a Bitwarden JSON export via the CLI.
- Import from 1Password: Import your vault from a 1Password `.1pux` export via the CLI.
- Import from LastPass: Import your vault from a LastPass CSV export via the CLI.
- Folders: Organize entries into named folders.
- Attachments: Store binary files alongside entries, encrypted within the database.
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
- C compiler (required by go-sqlcipher/CGO)

Clone and build:

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
- Default vault directory: `~/.passbook/data/`
- Database: `~/.passbook/data/passbook.db` (SQLCipher-encrypted)

## ☁️ iCloud sync

PassBook stores the vault under `data_dir` from `~/.passbook/config.json`. To sync via iCloud Drive (macOS only), run:

```bash
passbook --icloud
```

This moves your existing database to `~/Library/Mobile Documents/com~apple~CloudDocs/PassBook` and updates the config. Run the same command on another Mac to point both machines at the same vault.

To use a different cloud provider (Dropbox, Google Drive, etc.), edit `~/.passbook/config.json` and set `data_dir` to any synced folder path. Paths starting with `~/` are expanded.

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

- Duplicate titles within a folder are prevented by a unique index.
- Each entry is written directly to the encrypted database.
- **Delete the export file after importing.**

## 🗂️ Vault layout (on disk)

Inside `<dataDir>` you'll see:

- `passbook.db` — a single SQLCipher-encrypted SQLite database containing all entries, folders, attachments, password history, and 2FA configuration.
- `passbook.db-wal` — SQLite Write-Ahead Log (created automatically when the database is open).
- `passbook.db-shm` — SQLite shared-memory file (created automatically when the database is open).

The database schema includes:

| Table | Purpose |
| --- | --- |
| `folders` | Named folders for organizing entries |
| `entries` | All entry data (logins, cards, notes, files) |
| `password_history` | Historical passwords with timestamps |
| `attachments` | Binary file attachments stored as BLOBs |
| `pin_config` | 2FA configuration (PIN or TOTP) |

## 🔐 Security architecture

For the full security architecture — encryption details, authentication flow, 2FA design, and password strength requirements — see **[SECURITY.md](SECURITY.md)**.

**Summary:**

- **Encryption**: SQLCipher (AES-256-CBC with HMAC-SHA512 page-level authentication). The entire database is transparently encrypted.
- **Key**: The master password is used directly as the SQLCipher encryption key.
- **Password change**: `PRAGMA rekey` re-encrypts the entire database with the new key.
- **Two-factor authentication**: 6-digit numeric PIN (verified via HMAC-SHA256 with a random 32-byte key) or TOTP authenticator app. Configuration is stored in the encrypted database.
- **Password strength**: Enforced on vault creation and password change — weak passwords are rejected. Scoring is aligned with NIST SP 800-63B guidelines.
- **Clipboard clearing**: Sensitive values are automatically cleared from the clipboard after 30 seconds.
- **File permissions**: Database directory is `0700`, database file is `0600`, config file is `0600`.

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

## 🧰 Built with

- tview: https://github.com/rivo/tview
- tcell: https://github.com/gdamore/tcell
- go-sqlcipher: https://github.com/mutecomm/go-sqlcipher
- otp: https://github.com/pquerna/otp
- go-qrcode: https://github.com/skip2/go-qrcode
- clipboard: https://github.com/atotto/clipboard
