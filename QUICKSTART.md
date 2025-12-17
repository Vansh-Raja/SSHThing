# Quick Start

## Build & Run

```bash
go build -o sshthing ./cmd/sshthing
./sshthing
```

## First Run (Setup)

- On first launch (no DB yet), you’ll be prompted to create a **master password**.
- This password encrypts/unlocks your database. If you lose it, the database can’t be recovered.

## Add Your First Host

1. Press `a`
2. Fill required fields: `Host*`, `User*`, `Port*`
3. Optional: set a `Label` (recommended)
4. Choose auth:
   - **Paste key**: paste your private key (multi-line)
   - **Generate**: create a new key (Ed25519/RSA/ECDSA)
   - **Password**: `ssh` will prompt on connect (password is not stored)
5. Press `Shift+Enter` to save

## Connect

- Select a host and press `Enter` (SSH)
- For SFTP: press `S`, then `Enter`
- For Finder mount (beta, macOS): press `M`, then `Enter`
- Or press `/` to open spotlight search and:
  - `Enter` for SSH
  - `S`, then `Enter` for SFTP
  - `M`, then `Enter` for Finder mount (beta)

## Finder Mounts (Beta, macOS)

Install dependencies:

```bash
brew install --cask fuse-t
brew tap macos-fuse-t/homebrew-cask
brew install --cask fuse-t-sshfs
```

Usage:
- `M`, then `Enter` mounts/unmounts the selected host and opens it in Finder.
- On quit, if mounts are active, SSHThing asks whether to unmount or leave them mounted (and restores state on next login).

## Reset DB (Destructive)

- If you forget the master password, the encrypted DB cannot be recovered. To start fresh, delete `~/.ssh-manager/hosts.db` and rerun SSHThing.

## Sync Hosts Across Devices

SSHThing can sync your hosts to a private Git repository for multi-device access.

### Initial Setup (Primary Device)

1. Create a **private** Git repo (e.g., `github.com/you/sshthing-sync`). It can be empty.
2. Ensure your SSH key has **read/write access** to the repo.
3. Press `,` to open Settings.
4. Set:
   - **Sync: Enabled** → On
   - **Sync: Repository URL** → `git@github.com:you/sshthing-sync.git`
   - **Sync: SSH Key Path** → `~/.ssh/id_ed25519` (or leave empty for default)
5. Press `Esc` to save
6. Press `Shift+Y` to sync

### Setup on New Device

1. Install SSHThing and create a database with the **same master password**
2. Configure sync settings (same repo URL)
3. Press `Shift+Y` to pull hosts

Your private keys stay encrypted - only the same master password can decrypt them.

Sync status appears in the footer (e.g., "Sync: 5m ago" or "Syncing...").

## Settings

Press `,` to access settings:
- **UI**: Vim mode, icons
- **SSH**: Host key policy, keepalive, terminal mode
- **Mount**: Enable/disable, default path, quit behavior
- **Sync**: Enable, repository URL, SSH key, branch

## Troubleshooting

- Ghostty + remote `clear` errors: if your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions. If you still see issues, set `TERM=xterm-256color` on the remote shell profile.
- Finder mounts (beta): if you see permission errors in Finder/Terminal, check macOS privacy settings for your terminal app (Network Volumes / Files and Folders).
- Sync "reference not found": the remote repo may be empty or using a different branch. SSHThing handles this automatically on first sync.
- Sync "failed to decrypt key": ensure you're using the **same master password** on all devices.
