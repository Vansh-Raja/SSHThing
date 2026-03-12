---
name: sshthing
description: >
  Run commands on remote servers via SSHThing token-authenticated SSH.
  Use when the user asks to execute commands on a remote server, deploy code,
  check server status, manage services, transfer files, or interact with any
  remote host managed by SSHThing.
  Also use when the user asks to create tokens, manage SSH hosts, or configure SSHThing.
  Do not use for local shell commands, Docker containers, or cloud provider CLIs (AWS/GCP/Azure).
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

**Flags:**
- `-t` / `--target` — Host label exactly as shown in the SSHThing TUI (case-sensitive)
- `--auth` — Token string directly (leaks in process list; avoid in production)
- `--auth-file` — Path to file containing the token string
- `--auth-stdin` — Read token from stdin

**Behavior:**
- stdout/stderr from the remote command are forwarded to the caller
- Exit code from the remote command is preserved
- Non-zero exit means the remote command failed

### Session management

```bash
# Unlock session — caches master password for a duration
printf 'MASTER_PASSWORD' | sshthing session unlock --password-stdin --ttl 15m

# Check session status
sshthing session status

# Lock session — clears cached password immediately
sshthing session lock
```

Session unlock is only needed if the token doesn't carry the DB unlock secret (e.g. synced tokens from another device). Most tokens created locally work without session unlock.

### Other commands

```bash
sshthing                # Open the interactive TUI (do NOT launch from agent)
sshthing --version      # Print version
sshthing --help         # Show help
```

## Token System

Tokens authorize remote command execution. They are created in the SSHThing TUI by the user.

**Token format:** `stk_<tokenID>_<secret>`

**Creating a token (user must do this manually in the TUI):**
1. Run `sshthing` to open the TUI
2. Press `Shift+Tab` to navigate to the Tokens page
3. Press `a` to create a new token
4. Enter a name, select target hosts, press Enter
5. Copy the displayed token string — it is shown only once

**Storing tokens safely:**
```bash
echo "stk_..." > ~/.sshthing/token.txt
chmod 600 ~/.sshthing/token.txt
```

## Workflow

1. Confirm the user has a token and knows the target host label
2. If the user doesn't have a token, instruct them to create one in the SSHThing TUI
3. Execute the command:
   ```bash
   sshthing exec -t "Host Label" --auth-file ~/.sshthing/token.txt "the-command"
   ```
4. Parse stdout/stderr from the result
5. Check exit code — non-zero means the remote command failed

## Multi-Command Execution

For multiple commands on the same server, chain them:

```bash
sshthing exec -t "Host Label" --auth-file ~/.sshthing/token.txt "cd /app && git pull && systemctl restart myapp"
```

Or run separate exec calls for per-command error handling.

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

## Error Reference

| Error | Cause | Fix |
|---|---|---|
| "token not found" | Token string doesn't match any vault entry | Check token string, ensure vault exists |
| "target not allowed by token" | Host label not in token's scope | Check exact label spelling (case-sensitive) |
| "token is not active on this device" | v2 token needs activation or session unlock | Run `sshthing session unlock` or activate in TUI |
| "failed to unlock database" | Wrong master password | Re-run session unlock with correct password |
| Exit code 255 | SSH connection failed | Host unreachable, auth rejected, or network issue |

## Important Notes

- Never expose token strings in code, commits, or logs
- Use `--auth-file` or `--auth-stdin` to avoid leaking tokens in shell history
- Host labels are case-sensitive and must match exactly
- The TUI is interactive — do not launch it from the agent
- If the user needs to create a token or add a host, instruct them to use the TUI manually
