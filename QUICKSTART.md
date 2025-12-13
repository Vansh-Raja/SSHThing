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

- From the login screen, press `Ctrl+R` to delete `~/.ssh-manager/hosts.db` and re-run setup.

## Troubleshooting

- Ghostty + remote `clear` errors: if your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions. If you still see issues, set `TERM=xterm-256color` on the remote shell profile.
- Finder mounts (beta): if you see permission errors in Finder/Terminal, check macOS privacy settings for your terminal app (Network Volumes / Files and Folders).
