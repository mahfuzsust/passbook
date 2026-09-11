# 🔐 PassBook (Terminal Password Manager)

![Downloads](https://img.shields.io/github/downloads/mahfuzsust/passbook/total)

PassBook is a terminal-based password manager built in Go. It stores your vault locally in a single encrypted SQLite database (via SQLCipher), provides a [Bubble Tea](https://github.com/charmbracelet/bubbletea)-powered TUI for browsing/editing entries, and includes built-in TOTP generation with a live countdown.

## ✨ Features

- Local encryption: The entire vault is stored in a single SQLCipher-encrypted database file.
- Two-factor authentication: After login, an additional 6-digit PIN or TOTP authenticator app verification is required. Configurable on first use with QR code setup for authenticator apps.
- Entry types: Logins, Cards, Notes, and Files.
- Built-in TOTP: Generates 6-digit codes for Login entries with a smooth, color-coded live progress bar.
- Smart clipboard handling:
  - Copying sensitive values clears the clipboard after 30 seconds if it still contains the copied value.
  - Copying non-sensitive values shows a quick status.
- Password history: Login entries keep prior passwords + timestamps when the password changes.
- Password generator: Configure length and character classes, preview live, and insert into the editor.
- Folder picker: Assigning an entry to a folder opens a searchable list overlay instead of free-text entry.
- Change master password: Re-encrypts the database with a new key via SQLCipher's `PRAGMA rekey`.
- Import from Bitwarden: Import your vault from a Bitwarden JSON export via the CLI.
- Import from 1Password: Import your vault from a 1Password `.1pux` export via the CLI.
- Import from LastPass: Import your vault from a LastPass CSV export via the CLI.
- Folders: Organize entries into named folders.
- Attachments: Store binary files alongside entries, encrypted within the database.
- Cloud-sync friendly: Point the data directory at iCloud Drive / Dropbox / etc.
- Responsive layout: Left pane stays ~30% width and right pane ~70% width as the terminal resizes.

## 🚀 Installation

### Option A: Homebrew (macOS)

PassBook is distributed as a [Homebrew cask](https://docs.brew.sh/Cask-Cookbook) (pre-built binary). Linux users should use [Option B](#option-b-download-from-github-releases-recommended) or [build from source](#option-c-build-from-source).

```bash
brew install --cask mahfuzsust/tap/passbook
```

If you previously installed the old formula (`brew install mahfuzsust/tap/passbook`), uninstall it first:

```bash
brew uninstall passbook
brew install --cask mahfuzsust/tap/passbook
```

Upgrade:

```bash
brew upgrade --cask passbook
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

## 📦 Releases

Every push to `main` runs CI, then automatically:

1. Bumps the patch version from the latest [GitHub release](https://github.com/mahfuzsust/passbook/releases) (e.g. `v7.0.1` → `v7.0.2`)
2. Creates and pushes that semver tag
3. Builds and publishes release assets for Linux, macOS, and Windows via [GoReleaser](https://goreleaser.com/)
4. Updates the Homebrew cask in [mahfuzsust/homebrew-tap](https://github.com/mahfuzsust/homebrew-tap)

Binaries are built with **CGO enabled** (required for SQLCipher). Do not distribute builds compiled with `CGO_ENABLED=0`.


Manual release for an existing tag:

```bash
gh workflow run Release --ref vX.Y.Z
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
| `Ctrl+A` | Create a new entry / folder |
| `Ctrl+E` | Edit selected entry, or rename selected folder |
| `Ctrl+D` | Delete selected entry, or delete selected folder |
| `Ctrl+N` | Create a new folder |
| `Ctrl+F` | Focus search |
| `Ctrl+Y` | Quick copy (jump straight to a copy action for the selected entry) |
| `Ctrl+P` | Change master password |
| `Ctrl+Q` | Quit |
| `↑`/`↓` or `j`/`k` | Move selection in the vault tree |
| `Enter` | Open entry / expand-collapse folder |
| `Esc` | Focus vault tree |

### Viewer actions (entry detail pane)

| Key | Action |
| --- | --- |
| `u` | Copy username (Login) |
| `c` | Copy password (Login) or card number (Card) |
| `l` | Copy link (Login) |
| `t` | Copy current TOTP code (Login) |
| `v` | Reveal/mask sensitive value |
| `o` | Open link in browser (Login) |
| `h` | View password history (Login) |
| `1`-`9` | Download the corresponding attachment to `~/Downloads` |

Each action key only appears/works when the relevant field exists on the entry (e.g. `l` does nothing without a Link).

### Editor

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | Move between fields |
| `Enter` | Insert a newline in Notes; otherwise advance to the next field/button |
| `Ctrl+G` | Open the password generator (Login entries) |
| `Ctrl+B` | Browse the filesystem to attach a file (File entries) |
| `Esc` | Close the editor |

Tabbing onto the **Folder** field automatically opens a folder-picker overlay (`↑`/`↓` to choose, `Enter` to confirm, `Esc` to cancel) — there's no free-text folder input.

### Password generator

| Key | Action |
| --- | --- |
| `Tab` / `↑`/`↓` | Move between length, character-class checkboxes, Refresh, and Use |
| `Space` | Toggle a character class |
| `r` | Regenerate the preview |
| `Enter` | Regenerate (on Refresh) or apply the password (on Use) |
| `Esc` | Close without applying |

### Modals

| Context | Shortcut | Action |
| --- | --- | --- |
| Login / setup screen | `Enter` | Login or create vault |
| Login / setup screen | `Esc` | Quit |
| File browser | `Enter` | Open folder / attach selected file |
| File browser | `Esc` | Cancel file picker |
| Quick copy | letter shown next to each item | Run that copy action |
| History | `Esc` | Close history |

## 🧰 Built with

- bubbletea: https://github.com/charmbracelet/bubbletea
- bubbles: https://github.com/charmbracelet/bubbles
- lipgloss: https://github.com/charmbracelet/lipgloss
- go-sqlcipher: https://github.com/mutecomm/go-sqlcipher
- otp: https://github.com/pquerna/otp
- go-qrcode: https://github.com/skip2/go-qrcode
- clipboard: https://github.com/atotto/clipboard
