# SSHThing

A secure, modern SSH host manager TUI built with Go and Bubble Tea.

## What It Does

- Stores SSH hosts in an **encrypted SQLCipher database**
- Protects each saved private key with **per-key AES-GCM encryption**
- Unlocks everything with a **master password** (first-run setup + login)
- Connects via `ssh` by writing decrypted keys to a **secure temp file** and cleaning it up after exit

## Features

### ✅ Implemented
- 🔐 **Encrypted storage**: SQLCipher DB + AES-GCM per-key encryption
- 🔑 **Master password**: setup + login to unlock the DB
- 🏷️ **Labels**: optional friendly names for hosts (recommended)
- 🏠 **Host CRUD**: add/edit/delete
- 🔑 **Auth options**:
  - Paste existing private key (multi-line)
  - Generate new key (Ed25519/RSA/ECDSA)
  - Password auth (never stored; `ssh` prompts on connect)
- 🔌 **SSH connect**: connects using system `ssh`
- 🔎 **Spotlight search**: `/` to search and connect quickly
- 📁 **Mount in Finder (beta, macOS)**: mounts via FUSE-T + SSHFS and opens in Finder

### 📅 Planned
- Vim-mode toggle and more keybindings
- Import/export hosts
- SSH config integration
- Config file support and extra UX polish
- Mount settings (default remote path, options)

## Requirements

- Go **1.25+** (matches `go.mod`)
- OpenSSH tools available: `ssh`, `ssh-keygen`
- A terminal with 256-color support
- SQLCipher build support (this project uses `github.com/mutecomm/go-sqlcipher/v4`, which typically requires CGO and SQLCipher on your system)
- Finder mounts (beta): macOS + `sshfs` (FUSE-T)

## Install / Run

```bash
git clone https://github.com/Vansh-Raja/SSHThing.git
cd SSHThing
go build -o sshthing ./cmd/sshthing
./sshthing
```

## Homebrew (Tap)

Once a tap is published (see `HOMEBREW.md`):

```bash
brew tap Vansh-Raja/tap
brew install sshthing
```

## Finder Mounts (Beta, macOS)

Install dependencies:

```bash
brew install --cask fuse-t
brew tap macos-fuse-t/homebrew-cask
brew install --cask fuse-t-sshfs
```

## Keybindings

### Main View
- `↑/↓` or `j/k`: navigate
- `Enter`: connect to selected host (SSH)
- `S` then `Enter`: connect to selected host (SFTP)
- `M` then `Enter`: mount/unmount selected host in Finder (beta, macOS)
- `a`: add host
- `e`: edit host
- `d`: delete host
- `/`: spotlight search
- `?`: help
- `q`: quit

### Add/Edit Modal
- `Tab` / `Shift+Tab` or `↑/↓`: move between fields
- `←/→` (or `h/l`) on Auth selector: change auth mode
- `Space` on Key Type: cycle key type
- `Shift+Enter`: save and close
- `Esc`: cancel

### Spotlight
- `Enter`: connect (SSH)
- `S` then `Enter`: connect (SFTP)
- `M` then `Enter`: mount/unmount (beta, macOS)

## Data & Safety Notes

- Database location: `~/.ssh-manager/hosts.db`
- If you forget the master password, the encrypted DB cannot be recovered.
- Login screen: `Ctrl+R` deletes the DB (destructive) so you can start fresh.
- Mount points: `~/.config/sshthing/mounts/`
- If you choose “Leave Mounted & Quit”, a mount key file may remain at `~/.config/sshthing/mount-keys/` until you unmount.

## Ghostty TERM Note

If you use Ghostty, some servers may not have `xterm-ghostty` terminfo installed. When your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions to avoid errors like “unknown terminal type”. You can also override the value by setting `SSHTHING_SSH_TERM`.
