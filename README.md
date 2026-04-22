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
  - Password auth (optional encrypted storage + auto-login)
- 🔌 **SSH connect**: connects using system `ssh`
- 🔎 **Spotlight search**: `/` to search and connect quickly
- 📁 **SSHFS Mounts (beta)**: mounts remote filesystems via SSHFS (macOS Finder / Linux file manager)
- 🔄 **Git Sync**: sync hosts across devices via a private Git repository

### 📅 Planned
- SSH config integration
- Extra UX polish

## Requirements

- Go **1.25+** (matches `go.mod`)
- OpenSSH tools available: `ssh`, `ssh-keygen`, `sftp`
- Optional for best password auto-login on Linux/macOS: `sshpass`
- A terminal with 256-color support
- SQLCipher build support (this project uses `github.com/mutecomm/go-sqlcipher/v4`, which typically requires CGO and SQLCipher on your system)
- SSHFS mounts (beta): macOS requires FUSE-T + `sshfs`; Linux requires `sshfs` + FUSE

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

### Linux

**Install from Release (.deb — Debian/Ubuntu):**

```bash
curl -LO https://github.com/Vansh-Raja/SSHThing/releases/latest/download/sshthing-linux-amd64.deb
sudo dpkg -i sshthing-linux-amd64.deb
sshthing
```

**Install from Release (.rpm — Fedora/RHEL):**

```bash
curl -LO https://github.com/Vansh-Raja/SSHThing/releases/latest/download/sshthing-linux-amd64.rpm
sudo rpm -i sshthing-linux-amd64.rpm
sshthing
```

**Install from Release (tarball):**

1. Download the right tarball from [Releases](https://github.com/Vansh-Raja/SSHThing/releases):
   - x86_64: `sshthing-linux-amd64.tar.gz`
   - ARM64: `sshthing-linux-arm64.tar.gz`
2. Extract and run:

```bash
tar xzf sshthing-linux-*.tar.gz
chmod +x sshthing
./sshthing
```

3. (Optional) Install it on your PATH:

```bash
sudo mv sshthing /usr/local/bin/sshthing
```

**Note:** SSHFS mounts require `sshfs` and `fusermount` — install via your package manager (e.g., `apt install sshfs`).

### Windows

**Requirements:**
- Windows 10/11
- OpenSSH Client (Settings > Apps > Optional Features > OpenSSH Client)

**Install from Release:**
1. Download `sshthing-setup-windows-amd64.exe` from [Releases](https://github.com/Vansh-Raja/SSHThing/releases)
2. Run the installer
3. Leave the “Add SSHThing to PATH” option enabled (recommended)
4. Launch from the Start Menu, or run `sshthing` in a new terminal
5. If an in-app update finishes but the relaunch step fails, open a new terminal before retrying `sshthing`

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

### Beta Releases

SSHThing now supports an opt-in `beta` release feed alongside the default stable feed.

- Stable releases continue to use normal GitHub releases from tags like `v2.0.3`
- Beta releases use GitHub prereleases from tags like `v2.1.0-beta.1`
- Stable remains the default for all users

To opt in from the TUI:

1. Open `Settings`
2. Go to `updates`
3. Turn `beta releases` on
4. Optionally turn `auto apply updates` on
5. Run `check now`

First-cut beta auto-apply support is intentionally limited to standalone installs:

- macOS zip installs
- Linux tarball installs
- Windows standalone installer / portable installs backed by GitHub release assets

Package-manager installs stay stable/manual in the first beta implementation:

- Homebrew installs do not auto-apply beta builds
- Winget installs do not auto-apply beta builds
- Chocolatey installs do not auto-apply beta builds

If you enable `beta releases` on a package-manager install, SSHThing will still show beta availability when appropriate, but it will guide you to download the prerelease asset manually.

To install a beta build manually, use the prerelease tag URL instead of the stable `latest` release endpoint:

- Release page: `https://github.com/Vansh-Raja/SSHThing/releases/tag/v2.1.0-beta.1`
- Windows installer: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-setup-windows-amd64.exe`
- Windows portable zip: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-windows-amd64.zip`
- macOS Apple Silicon zip: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-macos-arm64.zip`
- macOS Intel zip: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-macos-amd64.zip`
- Linux amd64 tarball: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-linux-amd64.tar.gz`
- Linux arm64 tarball: `https://github.com/Vansh-Raja/SSHThing/releases/download/v2.1.0-beta.1/sshthing-linux-arm64.tar.gz`

Recommended beta install path for coworkers:

1. Download the standalone asset for their platform from the beta prerelease page.
2. Install or extract it using the same standalone steps documented above for macOS, Linux tarballs, or the Windows installer.
3. Launch SSHThing.
4. Open `Settings` -> `updates`.
5. Turn `beta releases` on.
6. Optionally turn `auto apply updates` on if they are on a standalone install.
7. Run `check now` to stay on the beta feed.

To publish a beta build from GitHub Actions, push an explicit beta tag:

```bash
git tag v2.1.0-beta.1
git push origin v2.1.0-beta.1
```

That tag runs `.github/workflows/release-beta.yml` and publishes a GitHub prerelease with the same platform asset names the updater already expects.

### From source

```bash
git clone https://github.com/Vansh-Raja/SSHThing.git
cd SSHThing
go build -o sshthing ./cmd/sshthing
./sshthing
```

**Windows from source:** See [BUILDING_WINDOWS.md](BUILDING_WINDOWS.md) for CGO/SQLCipher setup.

## SSHFS Mounts (Beta)

### macOS

```bash
brew install --cask fuse-t
brew tap macos-fuse-t/homebrew-cask
brew install --cask fuse-t-sshfs
```

### Linux

```bash
# Debian/Ubuntu
sudo apt install sshfs

# Fedora/RHEL
sudo dnf install fuse-sshfs

# Arch
sudo pacman -S sshfs
```

## Keybindings

### Main View
- `↑/↓` or `j/k`: navigate
- `Enter`: connect to selected host (SSH)
- `S` then `Enter`: connect to selected host (SFTP)
- `M` then `Enter`: mount/unmount selected host (beta, macOS/Linux)
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
- `M` then `Enter`: mount/unmount (beta, macOS/Linux)

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
- Sensitive host data in the sync file is encrypted with your master password before commit/push
- Private key/password secrets remain encrypted and are re-encrypted as needed during import
- Uses SSH key authentication for Git operations
- **Important**: Use the **same master password** on all devices to decrypt synced keys

### Password Auto-Login

- Default is **Off** (enable in Settings: `SSH: Password auto-login`).
- Windows uses OpenSSH askpass mode by default.
- Linux/macOS uses `sshpass` first, then askpass fallback.
- Tip: install `sshpass` on Linux/macOS for the most reliable password auto-login flow.

### Multi-Device Usage

1. Set up sync on your primary device and push
2. On a new device, install SSHThing and create a database with the **same master password**
3. Configure the same sync repository URL
4. Press `Shift+Y` to pull hosts from the remote

The sync status is displayed in the footer (e.g., "Sync: 2m ago", "Syncing...", or "Error: ...").
When you press `Shift+Y`, SSHThing now runs sync asynchronously and shows a live syncing indicator + loading bar in the footer while work is in progress.

## SSHThing Teams Browser Setup

The repo also includes a Next.js + Convex browser surface under `web/` for
Teams sign-in and CLI auth handoff.

1. From the repo root, start Convex once to create a local deployment and generate `convex/_generated`:

```bash
./node_modules/.bin/convex dev --once
```

2. Create a Clerk app, enable the Convex integration in Clerk, and collect:
   - `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
   - `CLERK_SECRET_KEY`
   - Clerk Frontend API / issuer domain (`CLERK_FRONTEND_API_URL` or `CLERK_JWT_ISSUER_DOMAIN`)

3. Put those values in the repo-root `.env.local`.
   `convex dev` already writes `CONVEX_URL` and `CONVEX_DEPLOYMENT` there.
   The Next app automatically maps `CONVEX_URL` to `NEXT_PUBLIC_CONVEX_URL`, so you do not need to duplicate it in `web/.env.local`.

4. Run the two dev processes:

```bash
./node_modules/.bin/convex dev
cd web && ./node_modules/.bin/next dev
```

5. Open `http://localhost:3000`, sign in with Clerk, switch or create an organization, and then return to the TUI device-flow page.

## Automation Tokens + `sshthing exec`

Use automation tokens when you want `sshpass`-style command execution for agents/scripts without exposing VPS passwords in plaintext files.

### What this gives you

- Tokens are created only inside logged-in SSHThing
- One token can allow access to multiple hosts
- Token access is bound internally to host IDs (label renames keep working)
- Tokens are immutable scope (to change scope, create a new token)
- Revoked tokens stop working immediately
- New tokens use DB-backed execution (host credentials are fetched from your encrypted DB at exec time)

### Create a token (inside app)

1. Open settings with `,`
2. Open `Automation: Manage tokens`
3. Press `N` to create
4. Enter a token name and press `Enter`
5. Select hosts with `Space`
6. Press `Enter` to create
7. In the one-time popup:
   - press `C` to copy token
   - press `Esc` to close

### Token manager keybindings

- `N`: create new token
- `A`: activate selected inactive token on this device
- `R`: revoke selected token
- `D`: delete selected token (revoked only)
- `Esc`: back

### CLI exec usage

```bash
sshthing exec -t "Production App Server" --auth "stk_xxx_yyy" "hostname"
```

Also supported:

```bash
sshthing exec -t "Production App Server" --auth-file /path/to/token.txt "hostname"
printf 'stk_xxx_yyy' | sshthing exec -t "Production App Server" --auth-stdin "hostname"
```

Session cache commands (optional):

```bash
printf 'MASTER_PASSWORD' | sshthing session unlock --password-stdin --ttl 15m
sshthing session status
sshthing session lock
```

### Multi-host token example

If one token has both `Production App Server` and `Background Worker` in scope:

```bash
sshthing exec -t "Production App Server" --auth "stk_xxx_yyy" "systemctl status my-app --no-pager"
sshthing exec -t "Background Worker" --auth "stk_xxx_yyy" "systemctl status my-worker --no-pager"
```

### Test flow (recommended)

1. Create token with two hosts in scope
2. Run one command against each host with same token
3. Revoke token in app (`R`)
4. Re-run command and confirm it fails
5. Delete revoked token (`D`)

### Important behavior

- `-t` must match a host label in that token's scope exactly (same text/case)
- use the value shown in the host `Label` field inside SSHThing
- if the label contains spaces, wrap it in quotes: `-t "Deployment Server"`
- If label not in scope, command is denied
- If token is revoked/deleted, command is denied
- Commands return remote exit code (good for CI/agent workflows)
- Synced token definitions may appear as inactive on a new device until activated (`A`) locally

### Security notes

- Prefer `--auth-file` or `--auth-stdin` over `--auth` for less shell-history exposure
- Token is shown once at creation: save it safely
- Revoked tokens should be deleted when no longer needed
- By default, usable token secrets are local to the current SSHThing data directory
- Optional setting `Automation: Sync token definitions` syncs only names/scope/revocations (no usable token secret material)

## AI Agent Skills

SSHThing ships with agent skills that let AI coding assistants (Claude Code, OpenCode, Codex) run commands on your remote servers using automation tokens.

**Quick setup** — paste this into any AI assistant:

```
Set up SSHThing agent skills for my AI coding assistant by following the instructions at: https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/SETUP_PROMPT.md
```

Or see [`skills/README.md`](skills/README.md) for manual installation.

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
- Mount points:
  - macOS: `~/Library/Application Support/sshthing/mounts/`
  - Linux: `~/.config/sshthing/mounts/`
- If you choose "Leave Mounted & Quit", a mount key file may remain at the mount-keys directory until you unmount.

### Environment Variables

- `SSHTHING_DATA_DIR`: Override the data directory (useful for testing or multiple instances)
- `SSHTHING_SSH_TERM`: Override the TERM value for SSH sessions

## Ghostty TERM Note

If you use Ghostty, some servers may not have `xterm-ghostty` terminfo installed. When your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions to avoid errors like “unknown terminal type”. You can also override the value by setting `SSHTHING_SSH_TERM`.
