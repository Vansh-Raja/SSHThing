---
name: sshthing
description: >
  Run commands on remote servers via SSHThing token-authenticated SSH.
  Use when the user asks to execute commands on a remote server, deploy code,
  check server status, manage services, transfer files, or interact with any
  remote host managed by SSHThing.
  Also use when the user asks to create tokens, manage SSH hosts, or configure SSHThing.
  Do not use for local shell commands, Docker containers, or cloud provider CLIs (AWS/GCP/Azure).
disable-model-invocation: true
allowed-tools: Bash(sshthing *)
argument-hint: "[command-to-run-on-server]"
---

# SSHThing — Remote Server Execution

Execute commands on remote SSH servers using token-based authentication.

## Prerequisites

SSHThing must be installed and accessible as `sshthing` in PATH. Verify:

```bash
sshthing --version
```

If not installed, the user needs to download it from the GitHub releases page or build from source with `go build -o sshthing ./cmd/sshthing`.

## Quick Start

Run a command on a remote server:

```bash
sshthing exec -t "Server Label" --auth-file ~/.sshthing/token.txt "command here"
```

## Core Commands

### Execute a remote command

```bash
# Using a token file (recommended)
sshthing exec -t "<host-label>" --auth-file <token-file> "<command>"

# Using a token from stdin
echo "$TOKEN" | sshthing exec -t "<host-label>" --auth-stdin "<command>"

# Using a token directly (not recommended — visible in process args)
sshthing exec -t "<host-label>" --auth "<token-string>" "<command>"
```

- `-t` is the **host label** exactly as shown in the SSHThing TUI (e.g. "GPU Server", "Main Ubuntu Server")
- The command runs on the remote server via SSH and stdout/stderr are forwarded
- Exit code from the remote command is preserved

### Session management (for v2 tokens only)

```bash
# Unlock session — caches master password for a duration
printf 'MASTER_PASSWORD' | sshthing session unlock --password-stdin --ttl 15m

# Check session status
sshthing session status

# Lock session — clears cached password immediately
sshthing session lock
```

Session unlock is only needed if the token doesn't already carry the DB unlock secret (e.g. synced tokens from another device). Most tokens created locally work without session unlock.

### Open the TUI

```bash
sshthing
```

The TUI is used to add/edit hosts, create tokens, manage groups, and configure settings. It is interactive and should not be launched by the agent.

## Token System

Tokens authorize remote command execution. They are created in the SSHThing TUI.

**Token format:** `stk_<tokenID>_<secret>` (e.g. `stk_abc123_xYz789...`)

**Token types:**
- **v2 tokens** (current) — contain an encrypted DB unlock secret; the SSH key is fetched from the encrypted database at runtime
- **Legacy tokens** — contain encrypted SSH credentials directly; fully self-contained

Both types work identically with `sshthing exec`. The user doesn't need to know which type they have.

**Creating a token (user must do this in the TUI):**
1. Open `sshthing`
2. Press `Shift+Tab` to navigate to the Tokens page
3. Press `a` to create a new token
4. Enter a name, select target hosts, press Enter
5. Copy the displayed token string — it is shown only once

**Storing tokens safely:**
```bash
# Save to a file with restricted permissions
echo "stk_..." > ~/.sshthing/token.txt
chmod 600 ~/.sshthing/token.txt
```

## Workflow for Remote Execution

1. Confirm the user has a token and knows the target host label
2. If the user doesn't have a token, tell them to create one in the SSHThing TUI
3. Run the command:
   ```bash
   sshthing exec -t "Host Label" --auth-file ~/.sshthing/token.txt "the-command"
   ```
4. Read stdout/stderr from the result
5. Check the exit code — non-zero means the remote command failed

## Multi-Command Execution

For multiple commands on the same server, chain them in a single exec call:

```bash
sshthing exec -t "Host Label" --auth-file ~/.sshthing/token.txt "cd /app && git pull && systemctl restart myapp"
```

Or run separate exec calls for independent commands (allows per-command error handling).

## Common Patterns

### Check server status
```bash
sshthing exec -t "Web Server" --auth-file token.txt "uptime && df -h && free -m"
```

### Deploy code
```bash
sshthing exec -t "Prod Server" --auth-file token.txt "cd /app && git pull origin main && npm install && pm2 restart all"
```

### Read logs
```bash
sshthing exec -t "API Server" --auth-file token.txt "tail -100 /var/log/app/error.log"
```

### Service management
```bash
sshthing exec -t "DB Server" --auth-file token.txt "systemctl status postgresql"
```

## Error Handling

- **"token not found"** — the token string doesn't match any token in the vault
- **"target not allowed by token"** — the host label doesn't match any host in the token's scope; check exact label spelling
- **"token is not active on this device"** — v2 token needs session unlock or activation
- **"failed to unlock database"** — wrong master password in session; re-run session unlock
- **Exit code 255** — SSH connection failed (host unreachable, auth rejected, etc.)

## Important Notes

- Never expose token strings in code, commits, or logs
- Use `--auth-file` or `--auth-stdin` instead of `--auth` to avoid leaking tokens in shell history
- Host labels are case-sensitive and must match exactly
- The TUI is interactive — do not try to script or automate TUI interactions
- If you need the user to create a token or add a host, instruct them to use the TUI manually
