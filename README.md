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
- 🔄 **Git Sync**: sync hosts across devices via a private Git repository

### 📅 Planned
- SSH config integration
- Extra UX polish

## Requirements

- Go **1.25+** (matches `go.mod`)
- OpenSSH tools available: `ssh`, `ssh-keygen`, `sftp`
- A terminal with 256-color support
- SQLCipher build support (this project uses `github.com/mutecomm/go-sqlcipher/v4`, which typically requires CGO and SQLCipher on your system)
- Finder mounts (beta): macOS only + `sshfs` (FUSE-T)

## Install / Run

### Homebrew (macOS)

```bash
brew tap vansh-raja/tap
brew install sshthing
sshthing
```

### macOS (Download a Release)

1. Download the right ZIP from [Releases](https://github.com/Vansh-Raja/SSHThing/releases):
   - Apple Silicon (M1/M2/M3): `sshthing-macos-arm64.zip`
   - Intel: `sshthing-macos-amd64.zip`
2. Unzip it and run:

```bash
unzip sshthing-macos-*.zip
chmod +x sshthing
./sshthing
```

3. (Optional) Install it on your PATH:

```bash
sudo mv sshthing /usr/local/bin/sshthing
```

If macOS blocks the binary on first run:

```bash
xattr -dr com.apple.quarantine sshthing
```

### Windows

**Requirements:**
- Windows 10/11
- OpenSSH Client (Settings > Apps > Optional Features > OpenSSH Client)

**Install from Release:**
1. Download `sshthing-setup-windows-amd64.exe` from [Releases](https://github.com/Vansh-Raja/SSHThing/releases)
2. Run the installer
3. Leave the “Add SSHThing to PATH” option enabled (recommended)
4. Launch from the Start Menu, or run `sshthing` in a new terminal

**Alternative (portable zip):**
1. Download `sshthing-windows-amd64.zip` from [Releases](https://github.com/Vansh-Raja/SSHThing/releases)
2. Extract to a folder (e.g., `C:\Tools\sshthing`)
3. Run `sshthing.exe`
4. (Optional) Add that folder to PATH manually

**Note:** The Mount feature is not available on Windows.

### Verify Downloads (Recommended)

Each release includes a `SHA256SUMS` file.

1. Download your asset and `SHA256SUMS` from [Releases](https://github.com/Vansh-Raja/SSHThing/releases)
2. Verify checksum:

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

On PowerShell (Windows):

```powershell
Get-FileHash .\sshthing-setup-windows-amd64.exe -Algorithm SHA256
```

Optional provenance verification (requires `gh`):

```bash
gh attestation verify sshthing-windows-amd64.zip --repo Vansh-Raja/SSHThing
```

### From source

```bash
git clone https://github.com/Vansh-Raja/SSHThing.git
cd SSHThing
go build -o sshthing ./cmd/sshthing
./sshthing
```

**Windows from source:** See [BUILDING_WINDOWS.md](BUILDING_WINDOWS.md) for CGO/SQLCipher setup.

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
- `Shift+Y`: sync hosts with Git repository
- `a`: add host
- `e`: edit host
- `d`: delete host
- `/`: spotlight search
- `,`: settings
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

## Git Sync

Sync your hosts across multiple devices using a private Git repository.

### Setup

1. Create a **private** Git repository (e.g., on GitHub). It can be empty.
2. Ensure your SSH key has **read/write access** to the repo (e.g., add it to GitHub as a Deploy Key or to your account).
3. Press `,` to open Settings.
4. Enable **Sync: Enabled**.
5. Set **Sync: Repository URL** (e.g., `git@github.com:username/sshthing-sync.git`).
6. Set **Sync: SSH Key Path** (defaults to `~/.ssh/id_ed25519` if left empty).
7. Press `Esc` to save settings.
8. Press `Shift+Y` to sync.

### How It Works

- Hosts are exported to a JSON file in a local Git repository
- Private keys remain **encrypted** (AES-GCM) in the sync file
- The encryption salt is included, allowing keys to be re-encrypted when imported to a different database
- Uses SSH key authentication for Git operations
- **Important**: Use the **same master password** on all devices to decrypt synced keys

### Multi-Device Usage

1. Set up sync on your primary device and push
2. On a new device, install SSHThing and create a database with the **same master password**
3. Configure the same sync repository URL
4. Press `Shift+Y` to pull hosts from the remote

The sync status is displayed in the footer (e.g., "Sync: 2m ago", "Syncing...", or "Error: ...").

## Data & Safety Notes

- Database location:
  - macOS/Linux: `~/.ssh-manager/hosts.db`
  - Windows: `%APPDATA%\sshthing\hosts.db`
- Config location:
  - macOS: `~/Library/Application Support/sshthing/config.json`
  - Linux: `~/.config/sshthing/config.json`
  - Windows: `%APPDATA%\sshthing\config.json`
- Sync repository:
  - macOS: `~/Library/Application Support/sshthing/sync/`
  - Linux: `~/.config/sshthing/sync/`
  - Windows: `%APPDATA%\sshthing\sync\`
- If you forget the master password, the encrypted DB cannot be recovered.
- Mount points (macOS only): `~/Library/Application Support/sshthing/mounts/`
- If you choose "Leave Mounted & Quit", a mount key file may remain at the mount-keys directory until you unmount.

### Environment Variables

- `SSHTHING_DATA_DIR`: Override the data directory (useful for testing or multiple instances)
- `SSHTHING_SSH_TERM`: Override the TERM value for SSH sessions

## Ghostty TERM Note

If you use Ghostty, some servers may not have `xterm-ghostty` terminfo installed. When your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions to avoid errors like “unknown terminal type”. You can also override the value by setting `SSHTHING_SSH_TERM`.
