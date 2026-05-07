# SSHThing Desktop — Full TUI Parity Plan & Checklist

> Source-of-truth checklist derived from three TUI audits (Personal hosts /
> Teams + tokens / peripherals + advanced) on 2026-05-07. Use this to track
> remaining work toward GUI feature parity with the TUI.

Legend: `[x]` shipped · `[~]` partial / needs polish · `[ ]` not started · `[!]` deferred / out of v1 scope

## Snapshot — where we are today

The Electron + Go-sidecar architecture works end-to-end. The shell, design
system, host list, host detail, session terminal, multi-tab UX, key gen,
credentials, health probe, mount, transfers, exec, settings, teams,
sign-in, account, and keys pages are all implemented at MVP fidelity.
The remaining work is breadth (TUI features we never wired up), polish
(visual consistency, empty states, keyboard parity), and cross-OS.

```
[x] Architecture: Electron + Go daemon + JSON-RPC over Unix-socket
[x] App shell: icon rail · topbar (search, sync, avatar) · routed pages
[x] Linear-style dark design tokens, primitives, layout
[x] Vault unlock, change password, lock, vacuum
[x] Host list (grouped + status dots), create / edit / delete / reveal / search
[x] Multi-tab terminal with auto-close on clean exit, Cmd+T/W/1..9
[x] Health probe + rich stats panel (CPU / RAM / GPU / disk / uptime)
[x] Mount, transfer (upload/download), exec
[x] Settings (vault, appearance, SSH defaults, sync, health scheduler)
[x] Teams: switcher, hosts, members, invites, tokens, audit, settings
[x] Sign-in flow + Account page + Keys page
[x] Cloud URL resolution mirrors TUI (env + ldflags + localhost fallback)
[x] macOS hiddenInset title bar; rail draggable
[~] :command bar removed pending VS-Code-style palette redesign
```

---

## Phase A — Core UX gaps (priority 1, "must-feel-finished")

### Vault & first-run
- [ ] **First-run vault create** flow (when no DB exists). Today's `Unlock` page only handles unlock. Show password + confirm fields, min-length 8.
- [ ] **Vault lock TTL warning** banner when expiry < 60 s (mirror TUI).
- [ ] **Reveal-credential** show-on-hold-only mode (avoid leaving secret on screen).

### Hosts & groups
- [ ] **Group rename / delete** in the sidebar (right-click + dropdown). Today only "Add group" works.
- [ ] **Tags as chip input** in HostDrawer (currently a comma-string).
- [ ] **Drag-drop key file onto HostDrawer** to populate paste-key field.
- [ ] **Paste-key full-screen editor** (`v` in TUI) — modal textarea for huge keys.
- [ ] **Reveal-on-hold** toggle for password / key fields in HostDrawer.
- [ ] **Last-connected** timestamp shown on host rows (currently only in detail).
- [ ] **Last-connected sort** option in sidebar header.
- [ ] **Confirm dialog** before destructive ops (delete host, delete group, delete team).

### Search / palette
- [ ] **Cmd+K palette** redesign (VS Code style): rich result rows, sections (Hosts / Groups / Commands / Settings), keyboard-only, fuzzy-rank.
- [ ] **Top-bar search** wires to Cmd+K state — typing in topbar = palette open.
- [ ] **Slash commands** to be re-introduced inside the palette (e.g. `:connect`, `:sync`, `:lock`) — replaces the old bottom command bar.

### Help / docs surface
- [ ] **Help overlay** (`?` keyboard shortcut) showing keyboard cheatsheet.
- [ ] **macOS app menu** (File / Edit / View / Window / Help) with Lock vault, Sign in/out, Quit.
- [ ] **System tray** with Quick Connect submenu (deferred to v1.1 polish if time).

### Visual polish
- [ ] Settings page: typography + spacing pass to match host detail rhythm.
- [ ] Teams page: same visual pass, plus a sticky tab bar.
- [ ] Empty states for every list (hosts, members, audit, tokens, transfers).
- [ ] Loading skeletons (host list, audit timeline) instead of "Loading…".
- [ ] Toaster: distinct success / error / info / warn colour rails.

---

## Phase B — Teams completeness (priority 2)

### Sign-in / session
- [x] Browser hand-off + poll
- [x] Token refresh on access-token expiry
- [x] Sign-out (revokes server-side)
- [ ] **Cancel sign-in** button on the polling screen (mirror TUI's `c`).
- [ ] **Re-open browser** button (TUI's `o`/`O`) when polling.

### Team management
- [x] List teams + switcher dropdown
- [ ] **Cmd+1…9** quick-switch shortcuts
- [ ] **Reorder teams** (move earlier / later) — wraps `teams.reorder` RPC
- [ ] **Create team** modal (currently TBD)
- [ ] **Rename team** in Team Settings
- [ ] **Delete team** with confirm (owner-only gating)
- [ ] **Leave team** action (non-owner)
- [ ] **Transfer ownership** flow

### Team hosts
- [x] List + create + update + delete
- [ ] **Import personal host → team** with 3-way conflict modal (Update / Duplicate / Cancel) mirroring `importPersonalHostToCurrentTeam`.
- [x] Reveal shared credential (audit-logged)
- [ ] **Reveal per-member credential** (admin) + roster view of who has set their credential.
- [ ] **Delete member credential** as admin (audit-logged).
- [ ] **Set per-member credential** UI (TUI doesn't have this; GUI lead).
- [ ] **Connect-config error surfaces** ("personal credential not configured", "shared credential not configured") — friendly wording matching TUI.

### Members & invites
- [x] List members
- [x] Update role / remove
- [x] Invite by email + role
- [ ] **Accept incoming invite** card on hosts page header when invites exist.
- [ ] **Decline invite** action.
- [ ] **Cancel sent invite** action with confirm.
- [ ] **Role-gating** in UI — hide owner-only and admin-only actions properly.

### Audit log
- [x] List
- [ ] **Filters**: by member, event type, date range.
- [ ] **Virtualised list** (react-window) — server can return 1000+ events.
- [ ] **Event detail modal** on row-click showing full metadata.

### Tokens
- [x] List, create, revoke, delete-revoked, copy-once on creation
- [ ] **Token execution audit** view per token (recent runs, exit codes) — wraps team-token execution audit.
- [ ] **Personal token sync** indicator + "Sync now" action when `Automation.SyncTokenDefinitions` is on.

---

## Phase C — Power features (priority 3)

### Mount / SSHFS
- [x] Start, stop, list (basic)
- [ ] **Prereq check UI** with platform-specific install instructions
  (sshfs / fuse-t / fusermount versions).
- [ ] **Restore mounts on startup** (TUI does this via `restoreMountsFromDB`).
- [ ] **Mount-aware quit dialog** ("unmount & quit" / "leave mounted" / "cancel") matching `Mount.QuitBehavior` config.
- [ ] **Show in Finder / Open in Terminal** actions for mounted hosts (reveal-in-OS).

### File transfer
- [x] Upload (drag-drop) + transfer tray
- [ ] **Download** flow surfaced from a future SFTP browser (deferred).
- [ ] **Cancel in-flight transfer** — daemon needs a `transfer.cancel` RPC.
- [ ] **Recursive** + **preserve** flags exposed in upload modal.

### Exec
- [x] One-shot exec modal
- [ ] **Result history** — store recent exec results so the user can re-open.
- [ ] **Run on multiple hosts** mode (the TUI's `:exec` workflow, batch).

### Auto-update
- [ ] **Update banner** in topbar when new release is detected.
- [ ] **Update channel toggle** in Settings (stable / beta) — wired to TUI's `Updates.ReleaseChannel`.
- [ ] **Auto-apply updates** toggle (writes to `Updates.AutoApplyUpdates`).
- [ ] **Apply update** button + restart flow via electron-updater.
- [ ] **PATH health** banner on Linux/Windows (mirror TUI's PATH check).

### Sync (cloud + git)
- [x] Sync now, status, configure
- [ ] **Auto-sync after CRUD** (config toggle: sync on host-add / edit / delete).
- [ ] **Conflict resolution UI** when push detects server-side change.
- [ ] **Devices list** (personal cloud) — wire `personalVault.listDevices`.
- [ ] **Sync events timeline** (personal cloud) — `personalVault.listSyncEvents`.
- [ ] **Scope toggles** UI for: hosts, credentials, token defs, health, mount-state.
- [ ] **Git sync setup wizard** (URL, branch, ssh-key path picker, "test connection").

### Diagnostics / extras
- [ ] **Help → System info** modal (versions, paths, keyring availability, etc.).
- [ ] **Daemon log viewer** modal (tail `daemon.log`).
- [ ] **Keyring health indicator** in Settings (matches TUI's `keyring.healthCheck`).

---

## Phase D — Polish & UX consistency (priority 4)

### Keyboard parity
- [ ] **Vim-mode opt-in** in Settings (j/k navigate sidebar, etc.).
- [ ] **Cmd+, → Settings** (mac convention).
- [ ] **Cmd+L → lock vault** + Cmd+Shift+T → switch team.
- [ ] **Esc closes any open overlay**, never quits the app.
- [ ] **All shortcuts surfaced in help overlay**, dynamically (don't hardcode).

### Theming
- [ ] **Theme picker** (System / Dark / Light) wired through config + the tokens.css `.light` overrides.
- [ ] **Reduced-motion** respect (prefers-reduced-motion).
- [ ] **Per-tab terminal theme** (sync with global theme by default).

### Errors / state
- [ ] **Route-level ErrorBoundary** with Slack-style "Something broke" panel.
- [ ] **Daemon-disconnected** banner with retry (when socket dies mid-session).
- [ ] **Vault-locked banner** instead of redirect, when locked while a tab is open.
- [ ] **Network offline** detection — Convex subscriptions clearly marked as stale.

### Identity / branding
- [ ] **App icon** for macOS / Windows / Linux (currently default Electron).
- [ ] **About dialog** with version + license.
- [ ] **First-run welcome screen** (one-time, with link to docs).

### Window
- [ ] **Window state persistence** (size + position) via electron-window-state.
- [ ] **Multi-window** support (per-team or per-host) — defer to v1.1 if scope creep.

---

## Phase E — Cross-OS + packaging (priority 5)

### Windows
- [ ] Smoke test on Windows VM: named-pipe socket, ConPTY, key-bindings.
- [ ] askpass over named pipe — verify `internal/ssh/askpass_windows.go` works through the daemon spawn path.
- [ ] EV cert procurement (lead time matters — start now).
- [ ] WiX / NSIS installer choice via electron-builder.

### Linux
- [ ] Smoke test on Wayland and X11.
- [ ] `.deb` + `.rpm` + `.AppImage` builds.
- [ ] sshfs prereq detection on common distros (Debian/Ubuntu, Fedora, Arch).

### macOS
- [x] Native title bar
- [ ] Apple Developer ID code signing + notarization.
- [ ] Universal binary (x86_64 + arm64) for the daemon (today builds host arch).
- [ ] DMG installer + auto-update channel.

### CI / release
- [ ] GitHub Actions matrix for tagged releases.
- [ ] electron-updater + GitHub Releases as update channel.
- [ ] Version bump + changelog automation.
- [ ] Beta channel published from `desktop-spike` until merged.

---

## Cross-cutting / debt

- [ ] **Refactor**: TUI should also become a daemon client (long-deferred). Adds 1-2 weeks; pays back forever. Don't tackle until Phase A-C are landed.
- [ ] **Tests**: at least one Playwright e2e per major flow (unlock, connect, edit, sign-in mock).
- [ ] **Telemetry / crash reporting** opt-in (Sentry or self-hosted), with strict redaction list.
- [ ] **Daemon `transfer.cancel`, `transfer.progress` polish** to match progress notifications already wired in the renderer.
- [ ] **Daemon `vault.lock` notification** → renderer redirects to /unlock (already partly wired; verify).

---

## Sequencing (one-pass plan)

```
Phase A  ─ 5-7 days  : "feels finished" core gaps + visual polish
Phase B  ─ 5-7 days  : Teams parity + member mgmt + audit polish
Phase C  ─ 7-10 days : Mount lifecycle + auto-update + sync depth
Phase D  ─ 4-6 days  : Keyboard / theming / errors / window state
Phase E  ─ 7-10 days : Cross-OS + packaging + signing + CI
                       ──────
                       ≈ 4-6 weeks calendar (solo)
```

Land each phase in main-line tagged commits so we can ship incrementally.
Phases A and B can run in parallel since they touch different surfaces.

## What's intentionally out of v1 scope

- [!] SFTP file browser (a real product on its own — separate plan).
- [!] AI / agent features (TUI's `agent-host-map.md` stays per-user).
- [!] Plugin system / theme marketplace.
- [!] Mobile apps (iOS / Android) — separate codebase.
- [!] Self-hosted Convex deployment.
- [!] Snap / Flatpak distribution (defer past v1).

---

## Detailed checklist (from TUI audits, deduped)

The following list is the union of the three audit reports, normalised
against what's already shipped. Use this when deciding what to pick up
mid-phase.

### Vault & auth
- [x] Vault unlock
- [ ] Vault first-run create
- [x] Change master password
- [x] Lock vault
- [x] Vacuum vault
- [x] TTL-cached unlock secret
- [ ] TTL expiry warning banner
- [x] Cloud sign-in (browser handoff)
- [x] Cloud sign-out
- [ ] Sign-in: re-open URL
- [ ] Sign-in: cancel poll
- [x] Auto access-token refresh

### Hosts CRUD
- [x] List + groups + status dots
- [x] Create / edit / delete
- [x] Generate key (ed25519 / rsa / ecdsa)
- [x] Import key (paste, file)
- [x] Reveal credential (audit-logged)
- [ ] Drag-drop key file
- [ ] Paste-key fullscreen editor
- [ ] Reveal-on-hold toggle in form
- [ ] Tags chip input
- [ ] Last-connected sort

### Groups
- [x] List
- [x] Create
- [ ] Rename
- [ ] Delete (with host-migration target)
- [x] Sidebar collapse/expand state (implicit via navigation)

### Search / commands
- [x] Sidebar text filter
- [x] Cmd+K palette (basic)
- [ ] Cmd+K palette (rich, sectioned, slash commands)
- [ ] Slash commands inside palette
- [ ] Help overlay (`?`)

### Sessions
- [x] Open / write / resize / close
- [x] Multi-tab + Cmd+T / W / 1-9
- [x] Auto-close on clean exit
- [x] Reconnect when tab is left open after non-zero exit
- [ ] Tab title from OSC sequences (already plumbed — verify)

### SFTP
- [ ] Open SFTP terminal (mirrors TUI's `S` mode)
- [!] SFTP file browser (out of v1)

### Health
- [x] Probe (with all fields)
- [x] List + scheduler + per-host badge
- [x] Rich stats panel (CPU / RAM / GPU / disk / uptime / latency)
- [ ] Health display modes (minimal / values / graph)
- [ ] Health graph history (sparkline)

### Mount
- [x] Start / stop / list
- [ ] Prereq check + install hints
- [ ] Restore on startup
- [ ] Mount-aware quit dialog
- [ ] Show in Finder / Open in Terminal

### Transfer
- [x] Upload (drag-drop)
- [x] Transfer tray + progress
- [ ] Cancel transfer
- [ ] Recursive / preserve flags
- [ ] Download surface (post-SFTP)

### Exec
- [x] One-shot exec modal
- [ ] Result history
- [ ] Multi-host batch exec

### Sync
- [x] Status, now, configure (basic)
- [ ] Auto-sync after CRUD toggle
- [ ] Conflict resolution UI
- [ ] Devices list
- [ ] Sync events timeline
- [ ] Scope toggles UI
- [ ] Git provider setup wizard

### Personal tokens
- [x] List, create, revoke, delete-revoked, copy-once
- [ ] Sync indicator + sync-now action
- [ ] Execute via CLI (already works — surface in UI?)

### Teams: hosts
- [x] List, create, edit, delete
- [x] Reveal shared credential
- [ ] Reveal per-member credential + roster
- [ ] Delete member credential (admin)
- [ ] Set per-member credential (GUI-original)
- [ ] Import personal → team with conflict resolution

### Teams: members + invites
- [x] List, role change, remove, invite
- [ ] Accept / decline incoming
- [ ] Cancel sent
- [ ] Role-gated UI

### Teams: audit
- [x] List
- [ ] Filters
- [ ] Event detail modal
- [ ] Virtualised list

### Teams: tokens
- [x] List, create, revoke, delete-revoked
- [ ] Execution audit per token

### Teams: management
- [ ] Create team
- [ ] Rename
- [ ] Delete
- [ ] Leave
- [ ] Transfer ownership
- [ ] Reorder (cmd+drag or move-earlier/later)
- [ ] Quick-switch Cmd+1..9

### Settings
- [x] All categories present (vault, appearance, SSH, sync, health)
- [ ] Settings filter (`/`)
- [ ] Visual polish pass
- [ ] Vim-mode toggle
- [ ] Theme picker (light / dark / system)
- [ ] Update channel toggle (stable / beta)
- [ ] Auto-apply updates toggle
- [ ] PATH health + fix
- [ ] Mount-quit-behavior radio

### Updates
- [ ] Banner when update available
- [ ] Channel switch
- [ ] Auto-apply
- [ ] Manual apply + restart

### Cross-OS
- [ ] Windows smoke + named pipe
- [ ] Linux smoke + sshfs prereqs
- [x] macOS title bar integration
- [ ] All three packaged + signed
- [ ] CI matrix on tags

### Engineering / debt
- [ ] TUI as daemon client refactor
- [ ] Playwright e2e
- [ ] Crash reporting opt-in
- [ ] Window state persistence
- [ ] App icon set
- [ ] About dialog
- [ ] First-run welcome
