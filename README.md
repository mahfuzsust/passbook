# 🔐 PassBook (Terminal Password Manager)

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
- Cloud-sync friendly: Point the data directory at iCloud Drive / Dropbox / etc.
- Responsive layout: Left pane stays ~30% width and right pane ~70% width as the terminal resizes.

## 🚀 Installation

### Prerequisites

- Go (see `go.mod`)

### Build from source

```bash
git clone https://github.com/mahfuzsust/passbook.git
cd passbook

go build -o passbook .
./passbook
```

On first run, PassBook creates:

- Config: `~/.passbook/config.json` (stores your `data_dir`)
- Default vault: `~/.passbook/data`
- Vault secret: `<dataDir>/.secret`
- Attachments: `<dataDir>/_attachments`

## 🗂️ Vault layout (on disk)

Inside `<dataDir>` you’ll see:

- `logins/` — encrypted JSON entries stored as `*.md`
- `cards/` — encrypted JSON entries stored as `*.md`
- `notes/` — encrypted JSON entries stored as `*.md`
- `files/` — encrypted JSON entries stored as `*.md` (plus attachment metadata)
- `_attachments/` — encrypted attachment blobs keyed by attachment ID
- `.secret` — vault-local KDF configuration (salt + Argon2id parameters)

Notes:

- The `*.md` extension is just a filename convention; the content is encrypted JSON, not Markdown.
- Entry filenames are based on the entry Title.
- If you create a new entry with a duplicate title, you’ll be prompted to Replace or Add Suffix.

## 🔐 Security architecture (current)

### Encryption

- AES-GCM with a random nonce.
- Nonce is prepended to ciphertext.

### Key derivation (production)

- The encryption key is derived from the master password using Argon2id (password-hardening KDF).
- A per-vault random salt and the Argon2id parameters are stored in `<dataDir>/.secret`.

Why `.secret` lives in the vault:

- It keeps the vault portable (move `<dataDir>` anywhere and it still unlocks).
- Losing `~/.passbook/config.json` won’t permanently lock you out as long as you still have the vault folder.

Important:

- Don’t delete `<dataDir>/.secret`. Without it, previously-encrypted data can’t be decrypted.

### Attachments

- Attachments are encrypted with the same AEAD construction.
- Downloading an attachment writes the decrypted file to your OS Downloads folder.

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
- otp: https://github.com/pquerna/otp
- clipboard: https://github.com/atotto/clipboard
- argon2 (x/crypto): https://pkg.go.dev/golang.org/x/crypto/argon2
