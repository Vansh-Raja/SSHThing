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

---

## Feature: Mount Remote Server in Finder (FUSE-T + SSHFS)

### Why This Approach
- macOS kernel-extension FUSE stacks (e.g. macFUSE) can require “Reduced Security” on Apple Silicon.
- **FUSE-T** is a kext-less FUSE implementation (NFSv4 userspace server under the hood) and is designed to mount/unmount using standard macOS tools.
- The FUSE-T project provides a companion SSHFS package (`fuse-t-sshfs`) installable via Homebrew.

### User Experience
- From the host list and spotlight, add a “Mount” action (beta):
  - `M` then `Enter`: mount/unmount the selected host (same chord pattern as SFTP).
    - If not mounted, this arms “Mount”; `Enter` performs the mount.
    - If already mounted, this arms “Unmount”; `Enter` unmounts.
    - `Esc` cancels the armed action.
  - Default remote path: user home directory.
  - Remote path should be configurable in the future (settings screen / per-host config); v1 uses the default only.
- On success, open the local mount folder in Finder (`open <localPath>`).
- Show mount status in the details panel (Mounted/Not Mounted + local path).
- Show a clear **beta** warning + install hints on first use if prerequisites are missing.
  - On quit, if mounts are active, show a confirmation modal: Unmount & Quit vs Leave Mounted & Quit.

### Prerequisites (What We Check)
- Verify `sshfs` is available in `$PATH`.
- Verify `umount` exists (macOS built-in).
- Verify `open` exists (macOS built-in).
- Provide a clear error if the user hasn’t installed FUSE-T + SSHFS:
  - `brew install --cask fuse-t`
  - `brew tap macos-fuse-t/homebrew-cask`
  - `brew install --cask fuse-t-sshfs`
- This feature is **macOS-only** for now (`runtime.GOOS == "darwin"`). On other OSes we show a friendly error.

### Mount Command (Shell-out Strategy)
The sshfs interface is:
`sshfs [user@]host:[dir] mountpoint [options]` and unmount uses `umount mountpoint` on macOS.

Proposed command:
- Create mountpoint: `~/.config/sshthing/mounts/<safe-name>` (or `os.UserConfigDir()/sshthing/...` if we switch to platform conventions).
- Run (examples):
  - `sshfs user@host:/remote/path /local/path -o reconnect,volname=HostLabel`
  - For the default “home directory” mount, use an empty remote path if supported by the installed sshfs build (commonly `user@host:`), otherwise fall back to a configurable path later.
  - Add port via ssh options: `-o port=<PORT>`
  - For key-based hosts, pass identity via ssh options: `-o IdentityFile=<tempKeyPath>`
  - Additional macOS-friendly options (optional): `-o defer_permissions` (lets Finder access files without strict UID mapping).

Mount success detection:
- `sshfs` may daemonize depending on build/options. After starting it, poll for a mounted filesystem at the mountpoint (e.g. run `mount` and look for the mountpoint) with a short timeout (1–2s). If not mounted, treat as failure and surface stderr.

### Security / Key Handling
- While mounted, keep key material available for reconnect:
  - Store a per-host mount key file at `~/.config/sshthing/mount-keys/host_<id>.key` (0600).
  - Delete (overwrite+remove) the file on unmount.
- Don’t write permanent keys to disk.

### Mount Lifecycle Manager (Implementation Plan)
Add `internal/mount/mount.go`:
- `type MountManager struct { active map[int]Mount }` keyed by Host ID
- `type Mount struct { HostID int; Hostname string; LocalPath string; RemotePath string; KeyPath string; PID int }`
- `CheckPrereqs() error` uses `exec.LookPath`.
- `MountHost(conn ssh.Connection, remotePath string, displayName string) (Mount, error)`
  - Create local mount dir.
  - If `remotePath` is empty, use the default (home directory).
  - Start `sshfs` as a background process (`cmd.Start()`), capture PID.
  - On success, run `open <LocalPath>`.
  - If mount fails, capture stderr and return it to the UI.
- `UnmountHost(hostID int) error`
  - Run `umount <LocalPath>` first.
  - Fallback: `diskutil unmount <LocalPath>` (or `diskutil unmount force <LocalPath>`) if `umount` fails.
  - Cleanup temp key file and remove from active map.
- `UnmountAll()` called during app exit to prevent orphan mounts.

### Beta / Safety Notes (UX Requirements)
- Show a one-time warning that this is **beta** and depends on external tooling (`fuse-t` + `fuse-t-sshfs`).
- Best-effort cleanup on normal exit (`q` / `ctrl+c`), but mounts may persist if the app crashes; provide an easy “Unmount” path (`M` then `Enter`) and clear error messages.
- For future settings (not in v1):
  - Default remote path (global)
  - Per-host remote path override
  - Mount options (reconnect, cache, volname, permissions behavior)
  - Custom mount root directory

### Where to Hook in the TUI
- `internal/app/app.go`:
  - Add `mountManager *mount.Manager` to `Model`.
  - Extend key handling in list + spotlight:
    - `M` arms mount/unmount; `Enter` executes (mirrors the SFTP chord).
  - On quit (`q` / `ctrl+c`): call `mountManager.UnmountAll()` before returning `tea.Quit`.
  - Add messages to report mount/unmount success/failure (so UI can show a footer notice).

### macOS Notes / Troubleshooting
- Because FUSE-T uses NFS under the hood, macOS privacy settings may require enabling “Network Volumes” access for the terminal app.

---

## Feature: Settings Screen (Configurable App Options)

### Goal
Add a dedicated **Settings** screen to configure global app behavior (SSH/SFTP/Mount defaults, UI options, safety/security toggles) without changing per-host data.

### UX / Keybindings
- From the main host list: press `,` (comma) to open Settings.
  - Rationale: common convention in apps, doesn’t conflict with existing bindings.
- In Settings:
  - `↑/↓` or `j/k`: navigate options
  - `Enter`: edit/select
  - `←/→` or `h/l`: change enum/number values (where applicable)
  - `Space`: toggle boolean options
  - `Esc`: back / close
  - `Shift+Enter`: save & close (matches modal save behavior)
- Note: we should not bind Settings globally inside spotlight search input because `,` is a normal character when typing.

### Storage & Format
- Store config at: `~/.config/sshthing/config.json` (via `os.UserConfigDir()`).
- Keep config **separate from the encrypted DB**:
  - DB = secrets + hosts (encrypted).
  - Config = behavior/preferences (non-secret, but still private).
- Support overrides (already exists / planned):
  - `SSHTHING_DB_PATH` / `SSHTHING_DATA_DIR` for DB location
  - `SSHTHING_SSH_TERM` for TERM override
  - Settings should display when an env var is overriding a value (“Managed by env var”).

### Proposed Settings (v1)

**General**
- Confirm on quit when mounts are active: `Always` / `Only if mounted` (default) / `Never`
- Default search behavior: `Spotlight opens empty` vs `keeps last query`
- Clipboard integration: `On` / `Off` (if used for copy actions)

**UI**
- Theme: `Default` / `High contrast`
- List sorting: `Label` / `Last connected` / `Created`
- Show icons in host list (e.g. ⚡): `On` / `Off`
- Vim navigation: `On` / `Off` (j/k/h/l)

**SSH / SFTP**
- Host key policy:
  - `accept-new` (current default; convenient)
  - `strict` (recommended; `StrictHostKeyChecking=yes`)
  - `off` (danger; `StrictHostKeyChecking=no`)
- Keepalive interval (seconds): default `60`
- TERM for launched sessions:
  - `auto` (current logic + Ghostty fix)
  - `xterm-256color`
  - custom string

**Finder Mounts (beta, macOS)**
- Enable mounts feature: `On` / `Off`
- Default remote path: `""` (home) by default, but configurable for future settings screen
- Default mount root: `~/.config/sshthing/mounts/` (display only unless we support changing)
- Quit behavior when mounts exist: `Prompt` (default) / `Always unmount` / `Leave mounted`

**Security / Safety**
- Hide private keys in edit view by default: `On` / `Off` (future UX, but should be a setting)
- Require confirmation before revealing/copying a key: `On` / `Off`
- Warn on first-connect host key trust (`accept-new`): `On` / `Off`

### Integration Points (Code-level Plan)
1. Add `internal/config/`:
   - `Config` struct with defaults
   - load/save JSON helpers
   - “effective config” overlay with env overrides
2. Add new view state: `ViewModeSettings`
3. Build Settings UI:
   - category list (left) + options pane (right), or single list with section headers
   - inline editing for booleans/enums, small input for strings/numbers
4. Wire config usage:
   - SSH/SFTP: host key policy, keepalive, TERM override
   - Mounts: remote path default, quit behavior
   - UI: list sorting, icon toggles, vim mode
5. Tests:
   - config load/save roundtrip
   - default config generation when file missing
   - env var override behavior (no secrets in config)

### Acceptance Criteria
- Settings screen opens via `,` from the main view.
- Changes persist across restarts (saved to config file).
- Existing behavior remains default-compatible when config file is absent.
- Clear indication for “beta” and “danger” settings (mounts, host key policy off).
