# 🔐 PassBook (Terminal Password Manager)

PassBook is a terminal-based password manager built in Go. It stores your vault **locally** as encrypted files, provides a TUI for browsing/editing entries, and includes built-in TOTP generation with a live countdown.

## ✨ Features

- **Local encryption**: Entries and attachments are encrypted at rest using **AES-256-GCM**.
- **Entry types**: **Logins**, **Cards**, **Notes**, and **Files**.
- **Built-in TOTP**: Generates 6‑digit codes for Login entries with a live progress bar.
- **Smart clipboard handling**:
  - “Copy password / card / TOTP” clears the clipboard after **30 seconds** if it still contains the copied value.
  - “Copy username” shows a quick “copied” status.
- **Password history**: Login entries keep prior passwords + timestamps when the password changes.
- **Password generator**: Generate a password and insert it into the editor.
- **Auto-lock**: Vault locks after **5 minutes** of keyboard/mouse inactivity.
- **Cloud-sync friendly**: Point the data directory at iCloud Drive / Dropbox / etc.
- **Responsive layout**: Left pane stays ~**30%** width and right pane ~**70%** width as the terminal resizes.

## 🚀 Installation

### Prerequisites

- Go (the project’s `go.mod` declares Go **1.25.7**)

### Build from source

```bash
git clone https://github.com/mahfuzsust/passbook.git
cd passbook

go build -o passbook .
./passbook
```

On first run, PassBook creates:

- Config: `~/.passbook/config.json`
- Default vault: `~/.passbook/data`
- Attachments: `<dataDir>/_attachments`

## 🗂️ Vault layout (on disk)

Inside `<dataDir>` you’ll see:

- `logins/` — encrypted JSON entries stored as `*.md`
- `cards/` — encrypted JSON entries stored as `*.md`
- `notes/` — encrypted JSON entries stored as `*.md`
- `files/` — encrypted JSON entries stored as `*.md` (plus attachment metadata)
- `_attachments/` — encrypted attachment blobs keyed by attachment ID

Notes:

- The `*.md` extension is just a filename convention; the content is **encrypted JSON**, not Markdown.
- Entry filenames are based on the entry **Title**.
- If you create a new entry with a duplicate title, you’ll be prompted to **Replace** or **Add Suffix**.

## 🔐 Security architecture (current)

### Encryption

- AES-GCM with a random nonce.
- Nonce is prepended to ciphertext.

### Key derivation

- The encryption key is `SHA-256(master password)`.
- This is simple and fast, but not a memory-hard KDF (Argon2id/scrypt). If you’re storing high-value secrets, consider hardening this (would require a migration strategy).

### Attachments

- Attachments are encrypted with the same AEAD construction.
- Downloading an attachment writes the **decrypted file** to your OS **Downloads** folder.

## ☁️ Cloud syncing (example: iCloud Drive on macOS)

Open Settings (`Ctrl+O`) and set **Data Directory** to something like:

`~/Library/Mobile Documents/com~apple~CloudDocs/PassBook`

PassBook expands paths that start with `~/`.

## ⌨️ Keyboard shortcuts

### Main screen

| Shortcut | Action |
| --- | --- |
| `Ctrl+A` | Create a new entry |
| `Ctrl+E` | Edit selected entry |
| `Ctrl+D` | Delete selected entry |
| `Ctrl+F` | Focus search |
| `Ctrl+O` | Open Settings (change vault path) |
| `Ctrl+Q` | Quit |
| `Esc` | Focus vault tree |

### Viewer actions

- **Login**: Copy username, toggle password “View”, copy password, open password history, copy TOTP.
- **Card**: Toggle “View” for full number, copy number.
- **File**: Select an attachment to download it to your Downloads folder.

### Modals / editor

| Context | Shortcut | Action |
| --- | --- | --- |
| Login screen | `Enter` | Login |
| Editor | `Esc` | Close editor |
| File browser | `Esc` | Cancel file picker |
| Password generator | `Esc` | Close generator |
| History | `Esc` | Close history |

## 🧰 Built with

- [tview](https://github.com/rivo/tview)
- [tcell](https://github.com/gdamore/tcell)
- [otp](https://github.com/pquerna/otp)
- [clipboard](https://github.com/atotto/clipboard)
