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

1. Press `:` and run `:add`
2. Fill required fields: `Host*`, `User*`, `Port*`
3. Optional: set a `Label` (recommended)
4. Choose auth:
   - **Paste key**: paste your private key (multi-line)
   - **Generate**: create a new key (Ed25519/RSA/ECDSA)
   - **Password**: `ssh` will prompt on connect (password is not stored)
5. Press `Shift+Enter` to save

## Connect

- Select a host and press `Enter` (SSH)
- For SFTP: run `:sftp`
- For SSHFS mount (beta, macOS/Linux): run `:mount`
- Or press `/` to open spotlight search and:
  - `Enter` for SSH
  - `S`, then `Enter` for SFTP
  - `M`, then `Enter` for SSHFS mount (beta)
- Press `:` to see all available commands with autocomplete, including
  `:add`, `:edit`, `:delete`, `:health`, `:sync`, `:settings`, and `:help`.

## SSHFS Mounts (Beta, macOS/Linux)

**macOS dependencies:**

```bash
brew install --cask fuse-t
brew tap macos-fuse-t/homebrew-cask
brew install --cask fuse-t-sshfs
```

**Linux dependencies:**

```bash
sudo apt install sshfs    # Debian/Ubuntu
sudo dnf install fuse-sshfs  # Fedora/RHEL
sudo pacman -S sshfs      # Arch
```

Usage:
- `:mount` mounts/unmounts the selected host and opens it in your file manager.
- On quit, if mounts are active, SSHThing asks whether to unmount or leave them mounted (and restores state on next login).

## Reset DB (Destructive)

- If you forget the master password, the encrypted DB cannot be recovered. To start fresh, delete `~/.ssh-manager/hosts.db` and rerun SSHThing.

## Sync Hosts Across Devices

SSHThing can sync your personal library with either a private Git repository or SSHThing Cloud.

### SSHThing Cloud Sync

1. Run `:profile` and sign in.
2. Run `:settings`.
3. Set **Sync: Provider** → `sshthing cloud`.
4. Keep the default portable scope unless you want to reduce what syncs.
5. Press `Esc` to save.
6. Run `:sync`.

Cloud personal sync is end-to-end encrypted. The TUI uses your local master password as the first-cut sync password. The browser personal library at `/personal` asks for that sync password and decrypts locally with WebCrypto; the server never receives it.

### Git Initial Setup (Primary Device)

1. Create a **private** Git repo (e.g., `github.com/you/sshthing-sync`). It can be empty.
2. Ensure your SSH key has **read/write access** to the repo.
3. Run `:settings` to open Settings.
4. Set:
   - **Sync: Provider** → `github`
   - **Sync: Repository URL** → `git@github.com:you/sshthing-sync.git`
   - **Sync: SSH Key Path** → `~/.ssh/id_ed25519` (or leave empty for default)
5. Press `Esc` to save
6. Run `:sync` to sync

### Setup on New Device

1. Install SSHThing and create a database with the **same master password**
2. Configure sync settings (same repo URL)
3. Run `:sync` to pull hosts

Your private keys stay encrypted - only the same master password can decrypt them.

Sync status appears in the footer (e.g., "Sync: 5m ago" or "Syncing...").

## Settings

Run `:settings` to access settings:
- **UI**: Vim mode, icons
- **SSH**: Host key policy, keepalive, terminal mode
- **Mount**: Enable/disable, default path, quit behavior
- **Sync**: Enable, repository URL, SSH key, branch

## Agent / CLI commands

SSHThing also has a non-interactive CLI surface for AI coding assistants and
scripts. After creating an automation token in the TUI, you can:

```bash
# Run a remote command (token-authenticated, no master-password prompt)
sshthing exec -t "Server" --auth-file ~/.sshthing/token.txt "uptime"

# Pipe a local file as the remote command's stdin
sshthing exec --in ./schema.sql -t "DB" --auth-file token.txt "psql -f -"

# File transfer (cp = scp-style, put/get = streaming)
sshthing cp  -t "Server" --auth-file token.txt ./build.tar :/srv/releases/
echo hi | sshthing put -t "Server" --auth-file token.txt /tmp/hi.txt
sshthing get  -t "Server" --auth-file token.txt /var/log/app.log > ./app.log
```

In Teams mode, create an `stt_...` team automation token from `:tokens`.
Team token executions are resolved through the Teams backend and every command
is logged for audit:

```bash
sshthing exec --team-id "<team_id>" --target-id "<host_id>" --auth-file team-agent.token "nvidia-smi"
sshthing exec -t "GPU Server" --auth-file team-agent.token "uptime"
```

See the [README](./README.md#automation-tokens-sshthing-exec--cp--put--get)
for full token / transfer docs, or install the agent skills at
[`skills/`](./skills/) so Claude Code / OpenCode / Codex can drive these
commands for you.

## Troubleshooting

- Ghostty + remote `clear` errors: if your local `TERM` is `xterm-ghostty`, SSHThing forces `TERM=xterm-256color` for SSH sessions. If you still see issues, set `TERM=xterm-256color` on the remote shell profile.
- Finder mounts (beta): if you see permission errors in Finder/Terminal, check macOS privacy settings for your terminal app (Network Volumes / Files and Folders).
- Sync "reference not found": the remote repo may be empty or using a different branch. SSHThing handles this automatically on first sync.
- Sync "failed to decrypt key": ensure you're using the **same master password** on all devices.
