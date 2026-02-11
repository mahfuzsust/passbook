# 🔐 PassBook (Terminal Password Manager)

A robust, terminal-based password manager built entirely in Go. Designed for developers and power users, it features AES-GCM encryption, built-in real-time TOTP generation, smart clipboard handling, and a highly reactive TUI (Terminal User Interface).

## ✨ Features

* **Master Password Encryption:** Everything is encrypted locally using AES-256-GCM.
* **Built-in Authenticator (TOTP):** Generates 6-digit 2FA codes in real-time with a live progress bar. No need to reach for your phone.
* **Smart Clipboard Security:** Copying a password or TOTP automatically clears your clipboard 30 seconds later (only if the clipboard contents haven't been overwritten).
* **Password Generator:** Generate strong, customizable passwords with a live preview and instant injection into your edit forms.
* **Password History:** Automatically tracks and timestamps previous passwords if you change them, ensuring you never lose access to an old account.
* **Auto-Lock:** Secures your vault and returns to the login screen after 5 minutes of keyboard/mouse inactivity.
* **Cloud Sync Ready:** Set your vault directory to any local path (including iCloud Drive, Dropbox, or Google Drive) to sync your encrypted entries seamlessly across devices.

## 🚀 Installation

### Prerequisites
* [Go](https://go.dev/doc/install) (1.18 or higher recommended)

### Build from Source
1. Clone the repository or navigate to the source directory:
   ```bash
   git clone [https://github.com/mahfuzsust/passbook.git](https://github.com/mahfuzsust/passbook.git)
   cd passbook
   ```
2. Build the application:
   ```bash
    go build -o passbook .
    ```
3. Run the application:
4. ```bash
    ./passbook
    ```
### ☁️ Cloud Syncing (iCloud Setup)
To sync your vault via iCloud, go to Settings (Ctrl+O) and set your Data Directory to: `~/Library/Mobile Documents/com~apple~CloudDocs/PassBook`

> Note: PassBook handles tilde (~) expansion automatically.

### 🔒 Security Architecture

**Storage**: Each entry is stored as an individual .md file. The JSON payload is encrypted with AES-256-GCM.

**Key Derivation**: Master keys are derived via SHA-256.

**Zero-Knowledge**: No data ever leaves your machine unless you choose to store it in a cloud-synced folder.

### Built With
- [tview](https://github.com/rivo/tview)
- [tcell](https://github.com/gdamore/tcell)
- [otp](https://github.com/pquerna/otp)
- [clipboard](https://github.com/atotto/clipboard)

### Keyboard Shortcuts

| Shortcut | Action |
| -------- | ------- |
| Ctrl + A, | Create a new entry |
| Ctrl + E, | Edit selected entry |
| Ctrl + D, | Delete selected entry |
| Ctrl + F, | Focus Search bar |
| Ctrl + O, | Open Settings (Change Vault Path) |
| Ctrl + Q, | Quit application |
| Esc, | Focus File List |
| Tab, | Next input field |
| Shift+Tab, | Previous input field |