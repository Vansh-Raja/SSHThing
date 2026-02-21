# Changelog

This repo is developed on `main`. Versioned releases are published as tags (starting at `v0.1.0`).
Entries below are written as an “engineering history” of the major problems we hit and how we fixed them (useful for future write-ups/portfolio posts).

## Running Log

- id: 2026-02-21T16:05:00Z-9f2c
  time: 2026-02-21T16:05:00Z
  type: code-change
  summary: Harden the release workflow with Windows/macOS smoke tests, explicit macOS amd64/arm64 runners, SHA256SUMS generation, artifact attestations, and draft-first publishing via gh with idempotency checks.
  files: .github/workflows/
  commit: pending

- id: 2026-02-21T16:04:40Z-a1b7
  time: 2026-02-21T16:04:40Z
  type: code-change
  summary: Extend CI with binary smoke tests to catch broken builds early.
  files: .github/workflows/
  commit: pending

- id: 2026-02-21T16:04:10Z-3d0e
  time: 2026-02-21T16:04:10Z
  type: code-change
  summary: Document release download verification steps in the README.
  files: README.md
  commit: pending

- id: 2026-02-20T11:44:38Z-8192
  time: 2026-02-20T11:44:38Z
  type: code-change
  summary: Auto-clear status messages by type in the UI, with checkmark-prefixed successes clearing after 5s and warnings/errors after 10s while keeping the sequence-safe timer to avoid clearing newer messages.
  files: internal/app/, internal/ui/
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T13:08:17Z-c19e
  time: 2026-02-19T13:08:17Z
  type: code-change
  summary: Add a Windows build helper script and document CGO_CFLAGS to suppress a known sqlite warning during CGO builds.
  files: scripts/build-windows.ps1, BUILDING_WINDOWS.md
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T13:08:14Z-6a2f
  time: 2026-02-19T13:08:14Z
  type: code-change
  summary: Reduce modal jank by skipping list rebuild and render prep when the app is not in the list view.
  files: internal/app/app.go, internal/ui/main.go
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T12:51:02Z-1d4d
  time: 2026-02-19T12:51:02Z
  type: code-change
  summary: Clean up create/rename group modal focus + layout with explicit input/submit/cancel focus, correct button highlighting, fixed Tab/Shift+Tab navigation, and updated help text.
  files: internal/app/app.go, internal/ui/modals.go, internal/ui/main.go
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T10:37:53Z-db0c
  time: 2026-02-19T10:37:53Z
  type: code-change
  summary: Classify DB unlock errors so a locked database shows a clear "database is in use by another process" message instead of a generic invalid password.
  files: internal/db/, internal/ui/
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T10:37:53Z-ab81
  time: 2026-02-19T10:37:53Z
  type: code-change
  summary: Improve modal QoL with a group selector spinner + position cue and Enter/Ctrl+S save behavior in the add/edit modal.
  files: internal/app/, internal/ui/
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

- id: 2026-02-19T10:37:53Z-1afd
  time: 2026-02-19T10:37:53Z
  type: code-change
  summary: Implement Groups across DB/app/ui/sync with grouped list, spotlight fuzzy group matches, and tombstones with 90-day retention.
  files: internal/db/, internal/app/, internal/ui/, internal/sync/
  commit: 729b0b250d21cc87d0773051f4c21bf829f62faa @ 2026-02-20T12:18:30Z

## 2025-12-13 — Stabilization, Security, and Polish

### Authentication & Encrypted DB Unlocking

**Problem:** The login screen accepted *any* master password and “unlocked” into an empty database view.

**Root cause:** With SQLCipher, opening a DB with the wrong key can still succeed at the SQLite driver level. If we then run schema creation/migrations on that handle, we risk interacting with a brand-new encrypted database view instead of the user’s real one.

**Fix:**
- If the DB file exists, we first open it in **read-only** mode and verify it’s actually unlocked before doing anything else.
- Only after verification do we open read-write and run schema setup/migrations.

**Where:** `db/db.go` (`Init`, `verifyUnlocked`).

### Host Persistence & Schema Migration (Notes → Label)

**Problem:** Hosts weren’t saving reliably, and the UI still referenced a legacy “Notes” field even though we wanted a “Label” for friendly names.

**Fix:**
- Removed “Notes” from the UI and data model, replaced with optional **Label**.
- Added a migration that **drops** the legacy `notes` column by rebuilding the `hosts` table (SQLite can’t `DROP COLUMN`), while copying `notes → label` when `label` is empty.

**Where:** `db/db.go` (`migrateHostsDropNotes`), UI modal rendering (`ui/modals.go`), list/details rendering (`ui/main.go`), app wiring (`app.go`).

### Per-Key Encryption (Second Layer) & Key Handling

**Goal:** Even after the DB is unlocked, stored private keys should still be protected at rest.

**Implementation:**
- PBKDF2 (100k iterations, SHA-256) derives keys.
- Private key blobs are encrypted with **AES-GCM** and stored base64-encoded in the DB (`hosts.key_data`).

**Where:** `crypto/crypto.go`, `db/db.go`.

### SSH Connect Reliability (“invalid format” / Permission denied)

**Problem:** SSH failed with messages like:
- `Load key "...": invalid format`
- `Permission denied (publickey).`

**Root cause:** Private key material written to a temp file can become invalid if:
- line endings are inconsistent (`CRLF` vs `LF`), or
- the file is missing a trailing newline (some tools are picky), or
- permissions aren’t strict enough.

**Fix:**
- Normalize line endings and ensure a trailing newline before writing.
- Write the temp key file with `0600` permissions in a dedicated temp directory.
- Clean up the file after the SSH session ends (including best-effort overwrite).

**Where:** `ssh/connect.go` (`NewTempKeyFile`, cleanup).

### SFTP Sessions (Using Saved Credentials)

**Goal:** Launch an interactive `sftp` session using the same stored host + key setup as SSH.

**Implementation:**
- Added a second connection path that runs system `sftp` instead of `ssh`.
- Reuses the same temp identity file + cleanup logic for key-based auth.
- Keybinding uses a reliable chord: press `S` to arm SFTP, then `Enter` to connect.
- Works from both the main list and spotlight search.

**Where:** `internal/ssh/connect.go` (`ConnectSFTP`), `internal/app/app.go` (keybinding + routing), `internal/ui/modals.go` (spotlight footer hints), `internal/ui/main.go` (footer/help text).

### Finder Mounts (Beta, macOS)

**Goal:** Mount a remote host in macOS Finder (Explorer-like workflow) using SFTP/SSHFS.

**Implementation:**
- Added a mount manager that shells out to `sshfs` (FUSE-T) and opens the mountpoint in Finder.
- Keybinding uses a reliable chord: press `M` to arm mount/unmount, then `Enter` to execute.
- Added a quit confirmation modal when mounts are active (Unmount & Quit vs Leave Mounted & Quit).
- Persisted mount state in the DB and reconciles it on next login by checking the OS’s actual mounted filesystems.
- Improved “leave mounted” safety by writing a per-host mount key file under `~/.config/sshthing/mount-keys/` (0600) and deleting it on unmount.

**Where:** `internal/mount/mount.go`, `internal/db/db.go` (`mounts` table), `internal/app/app.go` (quit modal + restore), `internal/ui/modals.go` (quit modal UI).

### Ghostty Compatibility (Remote “unknown terminal type”)

**Problem:** When connecting from Ghostty, remote shells could error on `clear`:
- `'xterm-ghostty': unknown terminal type.`

**Fix:**
- For SSH sessions launched via SSHThing, if local `TERM` is `xterm-ghostty`, force `TERM=xterm-256color` in the SSH process environment.
- Support an override via `SSHTHING_SSH_TERM` for users who want to force a specific value.

**Where:** `ssh/connect.go` (`sshEnv`).

### Modal UX & Keybindings

**Problems addressed:**
- “Broken” looking input on the login screen (cursor artifacts).
- Modal save instructions not matching behavior.
- Editing a host with an existing key forced re-entry because the key wasn’t prefilled.

**Fixes:**
- Improved cursor handling in the login/setup views to reduce terminal cursor artifacts.
- Add/Edit modal now supports **Shift+Enter** to save and close from any field; Enter only submits when the Save/Add button is focused.
- Edit modal pre-populates the existing decrypted key (so you can keep it unchanged or copy/edit it).
- Required fields are marked with `*` for quick scanning.

**Where:** `app.go`, `ui/login.go`, `ui/modals.go`.

### Search & Information Density

**Problem:** Showing raw IPs everywhere makes it hard to distinguish hosts.

**Fix:**
- Main host list shows **Label** (or hostname fallback) only.
- Spotlight search shows label first, with host/ip still available for context.

**Where:** `ui/main.go`, spotlight rendering in `app.go`/`ui`.

### Developer Experience

**Improvements:**
- Standardized build output name to `sshthing` (consistent with docs).
- Updated docs to reflect the current encrypted architecture and keybindings.
- Added/updated tests around filtering and DB unlock behavior (and adjusted Go tooling to work with restricted cache paths by allowing `GOCACHE` override during tests).

### Git Sync Feature

**Goal:** Enable users to sync their encrypted SSH hosts across multiple devices using a private Git repository.

**Implementation:**
- Added a sync manager that orchestrates Git operations, export/import, and conflict resolution.
- Export: Hosts are serialized to JSON with keys remaining encrypted (AES-GCM), including the source database salt for re-encryption.
- Import: Remote hosts are merged with local data, respecting timestamps to handle conflicts.
- Git operations: Uses SSH key authentication, supports custom branches, and handles empty/first-time repositories.
- Security: Private keys never leave the database decrypted; the master password is required on all devices to decrypt synced keys.
- UI: Sync status shown in settings, with Shift+Y to trigger manual sync.

**Where:** `internal/sync/` (manager, git, export, import, data), `internal/app/app.go` (sync triggers), `internal/ui/settings.go` (sync config display).
