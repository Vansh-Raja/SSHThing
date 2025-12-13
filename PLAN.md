# Plan

## Feature: Initiate SFTP Sessions (macOS Terminal / OpenSSH)

### Goal
Support launching an interactive **SFTP** session using the same saved credentials as SSH (username/host/port + optional stored private key) from both the host list and spotlight.

### UX / Keybindings
- `Enter`: connect via `ssh` (current behavior).
- `Shift+S`, then `Enter`: connect via `sftp` to the selected host.
  - In terminals, `Shift+S` is the same as the uppercase key `S`, so we’ll treat this as a **two-step chord**: press `S` to “arm” SFTP, then `Enter` to launch it.
  - Optional polish: show a short hint in the footer/status line while SFTP is armed (and allow `Esc` to cancel the armed state).

### How SFTP Works (system tooling)
We’ll use the system OpenSSH `sftp` client (available on macOS):
- Key auth: `sftp -i <temp_key_path> -P <port> -o StrictHostKeyChecking=accept-new user@host`
- Password auth: `sftp -P <port> -o StrictHostKeyChecking=accept-new user@host` (prompts; not stored)
Notes:
- `-P` (uppercase) is the port flag for `sftp`.
- `-i` points at an identity file (same temp key strategy as SSH).
- `-o` passes options through to the underlying SSH transport.

### Implementation Steps
1. **Command builder**
   - Add `ConnectSFTP(conn ssh.Connection) (*exec.Cmd, *TempKeyFile, error)` in `internal/ssh/`.
   - Reuse `TempKeyFile` and key normalization logic so we keep strict permissions (`0600`) and cleanup behavior.
   - Keep consistent options with SSH (`StrictHostKeyChecking=accept-new`, `ServerAliveInterval=60` via `-o`).
2. **App wiring**
   - Add a small “armed action” state to `internal/app/` (e.g. `armedConnectMode: ssh|sftp`).
   - In list view and spotlight:
     - On `S`: set mode to `sftp` (armed).
     - On `Enter`: choose SSH vs SFTP based on armed mode; reset mode after launching.
3. **UI updates**
   - Update footer help text to mention `Enter` (SSH) and `S` + `Enter` (SFTP).
   - If armed, show a subtle status message (e.g. “SFTP armed: press Enter”).
4. **Persistence / metadata**
   - Treat SFTP like a “connection”: update `last_connected` on launch (same as SSH).
5. **Tests**
   - Add a unit test for keybinding state transitions in list view (arming + launching resets).
   - Add tests for SFTP command args (port flag `-P`, identity `-i` when key present).

### Acceptance Criteria
- `Enter` still opens SSH exactly as before.
- `S` then `Enter` opens an interactive SFTP prompt for the selected host.
- Key-based hosts use a temp identity file; password-auth hosts prompt normally.
- Temp key file is cleaned up after SFTP exits.
