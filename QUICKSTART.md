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
- Or press `/` to open spotlight search and:
  - `Enter` for SSH
  - `S`, then `Enter` for SFTP

## Reset DB (Destructive)

- From the login screen, press `Ctrl+R` to delete `~/.ssh-manager/hosts.db` and re-run setup.

## Troubleshooting

- Ghostty + remote `clear` errors: if your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions. If you still see issues, set `TERM=xterm-256color` on the remote shell profile.
