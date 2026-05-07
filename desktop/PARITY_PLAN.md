# SSHThing Desktop — TUI Parity Plan

## Design language (locked-in, 2026-05-07)

Linear-inspired, dark-first, three-region layout:

```
┌──────┬─ Topbar: brand · search ⌘K · synced · avatar ──┐
│ icon │                                                │
│ rail ├─ pane-shell ───────────────────────────────────┤
│      │ ┌──────────────┬─────────────────────────────┐ │
│      │ │ Sidebar:     │ Detail or terminal tabs     │ │
│      │ │ groups +     │                             │ │
│      │ │ host rows    │                             │ │
│      │ │ + Add group  │                             │ │
│      │ └──────────────┴─────────────────────────────┘ │
│      ├─ Bottom command bar: >_ :connect nas …  ⌘K  ↑─┤
└──────┴────────────────────────────────────────────────┘
```

- **Palette**: dark navy `--paper #0e1117`, sidebar `--paper-2 #13171f`, surfaces `--paper-3/4`. Accent blue `--accent #3b82f6`. Status dots in green/yellow/red. Subtle `rgba(255,255,255,0.06)` borders, no heavy lines.
- **Typography**: system sans (`-apple-system, Inter, …`) for UI, JetBrains Mono only inside terminals and for code/addresses. 13 px base. Generous line-height. **No** uppercase or letterspacing tricks.
- **Surfaces**: rounded 4–10 px. Subtle `box-shadow` only on floating elements (drawers, modals, palette). No "paper / notebook" shadows.
- **Components shared with `web/`**: Drawer, Modal, Tabs, DropdownMenu primitives; the dashboard and desktop should feel like one product.
- **Mock**: `desktop/PARITY_PLAN.md` references the user-supplied mockup of `nas` host detail (group=Homelab, password auth, mounted at `/Volumes/nas`). Match that visual fidelity.

## Context

The spike at `desktop/` proved the architecture: Electron + Go sidecar daemon, JSON-RPC over Unix socket / named pipe, xterm.js + PTY, vault unlock + host list + working SSH session. End-to-end works on macOS today.

This plan brings the desktop app to **full TUI parity plus reasonable Termius-class polish** for v1, on macOS first, with Windows + Linux validation done as a single late-stage pass before release.

Locked-in decisions (do not re-litigate):
- **TUI becomes a daemon client too.** Both clients (TUI and GUI) talk to the same daemon. Single source of truth for crypto, DB, sync, audit. The TUI's existing inline-in-Bubble-Tea logic moves into daemon services.
- **React + Vite** for the desktop UI. Replaces the spike's vanilla TS. Reuse design tokens from `web/`.
- **Full parity scope** — vault/hosts/groups/keys, Clerk sign-in + Convex sync, teams + audit + automation tokens, health/mount/file-transfer/exec.
- **Mac-first dev**, Windows + Linux validated and packaged at the end.

Estimated calendar time, solo: ~14-16 weeks. Team of 2: ~9-11 weeks. This is a long plan — phases are designed to ship incrementally so each one is a useful checkpoint, not a halfway state.

## Target architecture (end-state)

```
                 ┌─────────────────────────────────────────────┐
                 │                  sshthing-daemon            │
                 │  - vault, hosts, groups, keygen, search     │
                 │  - sync (git + Convex), audit, teams cache  │
                 │  - sessions (PTY pool), exec, file xfer     │
                 │  - health probe scheduler, sshfs mount      │
                 │  - browser-handoff sign-in, token rotation  │
                 │  Unix socket / named pipe + JSON-RPC 2.0    │
                 └────────────┬────────────────┬───────────────┘
                              │                │
                  ┌───────────┘                └───────────┐
                  │                                        │
        ┌─────────▼────────┐                    ┌──────────▼─────────┐
        │  TUI client      │                    │  Electron client   │
        │  cmd/sshthing    │                    │  desktop/          │
        │  Bubble Tea +    │                    │  React + Vite +    │
        │  Go RPC client   │                    │  TS RPC client +   │
        │                  │                    │  xterm.js          │
        └──────────────────┘                    └────────────────────┘
                                                          │
                                                  ┌───────▼─────────┐
                                                  │   Convex        │
                                                  │   (renderer →   │
                                                  │   direct, for   │
                                                  │   real-time     │
                                                  │   data)         │
                                                  └─────────────────┘
```

Notes:
- **Convex direct from renderer.** The Electron renderer holds the Clerk JWT and uses `convex/browser` directly for real-time data. The daemon owns local state (SQLCipher vault, decrypted secrets, PTYs) and gates anything secret. This matches what the existing `web/` does.
- **Daemon is single-instance.** Both TUI and GUI launch / connect to the same daemon. First client to start wins; later starts attach. Lifecycle handled by a small lockfile + auto-spawn helper.
- **Audit logs are written by the daemon** anytime a secret is revealed (`hosts.revealCredential`, key reveals in teams flows, etc.) — single audit policy regardless of which client triggered the action.

## Phasing

Each phase ends with a working, releasable state. No phase is allowed to leave the app broken; if the phase blows past its envelope, cut scope, don't push half-done code.

---

### Phase 0 — UI foundation (week 1)

Goal: replace the vanilla-TS spike with a real React + Vite skeleton that the rest of the work plugs into. No new features.

**Deliverables**
- `desktop/` reorganized: `src/main/` (Electron main, unchanged), `src/preload/` (unchanged), `src/renderer/` rewritten as a Vite + React + TS app.
- Vite config that bundles renderer + watches in dev. Replace the current esbuild step.
- Design tokens copied from `web/` (CSS vars, typography). Use the same JetBrains Mono / `--paper` / `--ink` palette as the dashboard so the products look unified.
- Component primitives library at `src/renderer/ui/`: `Drawer`, `Modal`, `Dialog`, `DropdownMenu`, `Toast`, `Tabs`, `Sidebar`, `IconButton`, `TextField`, `PasswordField`. Tiny, hand-rolled to match `web/components/ui/`. Use `@radix-ui/react-*` for the hard ones (Dialog, DropdownMenu, Tooltip).
- Electron main spawns daemon (already done). Renderer connects to daemon via existing IPC bridge.
- App routes: vault unlock → host list (existing) — but rendered in React, with empty placeholders for the screens to come.

**Acceptance**
- `pnpm run dev` opens the window, password unlock works, host list renders, and clicking a host opens a single terminal — same as the spike, but in React.
- DevTools shows zero React warnings. Vite HMR works for both renderer code and CSS.

---

### Phase 1 — Daemon RPC expansion (weeks 1-3, mostly parallel with Phase 0)

Goal: implement every RPC in the existing spike plan's "stub catalog" (section A3 of `/Users/vanshraja/.claude/plans/yes-go-ahead-and-quizzical-firefly.md`) so both clients have the surface they need. Target file: `internal/daemon/rpc/*.go` and `internal/daemon/service/*.go`.

**Vault & keyring**
- `vault.create`, `vault.changePassword`, `vault.lock`, `keyring.healthCheck`. Wraps `db.Init`, future `db.ChangePassword` (add if absent), `unlock.Clear`, `securestore.kGet/kSet`.

**Hosts**
- `hosts.create`, `hosts.update`, `hosts.updateWithKey`, `hosts.delete`, `hosts.revealCredential` (audit-logged), `hosts.import`, `hosts.generateKey`. Wraps `store.CreateHost/UpdateHost/UpdateHostWithKey/DeleteHost/GetHostSecret` and `ssh.GenerateKey/ValidatePrivateKey`.
- `hosts.list` extended to support sort + filter parameters used by the GUI.

**Groups**
- `groups.list`, `groups.create`, `groups.rename`, `groups.delete`. Wraps `store.GetGroups/UpsertGroup/RenameGroup/DeleteGroup`.

**Sessions** (extends spike)
- `session.exec` for non-interactive runs — wraps `ssh.ConnectExecCaptured`.
- `session.list` — list active sessions in daemon memory (used by tab tray).
- New notifications: `session.titleChanged` (from OSC 1/2 sequences forwarded by xterm.js → write back to daemon for tab labels).

**File transfer**
- `transfer.upload` and `transfer.download` — wraps `internal/ssh/transfer.go`. Streamed progress via `transfer.progress` notifications.

**Health**
- `health.probe` (one-shot), `health.list`. Wraps `health.Probe` and `store.GetHostHealth`. A periodic scheduler in the daemon kicks probes for hosts marked "watch".

**Mount (sshfs)**
- `mount.start`, `mount.stop`, `mount.list`. Wraps `internal/mount`.

**Sync**
- `sync.status`, `sync.now`, `sync.pull`, `sync.push`, `sync.configure`. Wraps `sync.Manager` and `config.Save` for the `Sync` section.

**Auth (Convex device-code)**
- `auth.startSignIn` — wraps `teamsclient.StartCLIAuth`. Returns `{ url, sessionId, deviceCode, expiresAt }`.
- `auth.openBrowser({url})` — daemon shells out to `open` / `xdg-open` / `start`.
- `auth.pollSignIn({sessionId, pollSecret})` — wraps `teamsclient.PollCLIAuth`. Daemon stores returned access + refresh tokens via `teamssession.Save`.
- `auth.signOut`, `auth.session`, `auth.refresh`. Wraps `teamssession` + `rotateAccessToken` in Convex.

**Teams**
- `teams.list`, `teams.hosts.list/create/update/delete`, `teams.members.list/invite/updateRole/remove`, `teams.audit.list`, `teams.invites.list/accept/revoke`. Wraps `teamsclient.*`.

**Automation tokens**
- `tokens.list`, `tokens.create`, `tokens.revoke`. Wraps `internal/authtoken`.

**Settings**
- `settings.get`, `settings.set` (patch). Wraps `config.Load`/`config.Save`.

**Daemon meta**
- `daemon.shutdown`, `daemon.health` (uptime, version, vault locked status, active sessions count).

**Acceptance**
- Each RPC has a Go test exercising the happy path against a temp vault.
- `nc -U` smoke tests in `desktop/scripts/smoke.sh` cover: vault unlock → create host → list → reveal cred → audit shows entry → delete.
- `go vet ./...` clean. `golangci-lint run` clean (add config if absent).

---

### Phase 2 — TUI as a daemon client (weeks 3-5)

Goal: refactor the existing TUI to call the daemon over RPC instead of `internal/db`, `internal/ssh`, etc. directly. The TUI keeps its Bubble Tea UX; only the data plane changes.

**Sub-steps**
1. **Extract a shared Go RPC client** at `internal/daemon/client/`:
   - `client.Connect(sockPath, token) (*Client, error)`
   - Typed methods mirroring every RPC.
   - Notification subscription via channels.
   - This package becomes the single API the TUI uses to reach the daemon.
2. **Daemon auto-spawn helper** at `internal/daemon/launcher/`:
   - If the daemon socket isn't reachable, spawn the daemon binary and wait for it.
   - Lockfile-based single-instance guarantee.
   - Used identically by Electron main and TUI.
3. **TUI refactor**: starting with the highest-traffic call sites (`internal/app/backend.go connectToHost`, `internal/app/handlers/*`), replace direct `store.*` and `ssh.*` calls with daemon-client calls. Move the Bubble Tea state mutations to consume daemon responses. Keep the rendering code untouched.
4. **Decommission unused TUI plumbing**: remove the now-unused `*db.Store`, sync manager, etc. from the TUI's `Model` struct; they live solely in the daemon now.
5. **Backwards compat**: TUI auto-launches the daemon if not running, so no user-facing change. Existing keybindings, layouts, search behavior are identical.

**Acceptance**
- The TUI runs against the daemon for *every* operation (vault unlock, host CRUD, sync, connect, audit).
- All existing TUI tests pass without modification.
- Adding `--no-daemon` flag is **out of scope** — daemon is now mandatory.
- Manual smoke: do every TUI flow end-to-end (login → list → connect → exec → exit) and confirm parity.

---

### Phase 3 — Core GUI parity (weeks 5-8)

Goal: bring the GUI to the same feature footprint as today's TUI's "core" — everything you can do without sign-in or teams.

**Screens & flows**
- **Vault unlock** — first-run create, change password, lock-from-menu (already partly working).
- **Host list** — virtualized list, group folders, tags chips, last-connected timestamp, status pill (online/offline/unknown). Sort by recent / alphabetical / group. Inline `/` quick search.
- **Host drawer** — slide-out for create / edit. Form fields match `web/components/teams/HostDrawer.tsx` patterns: hostname, username, port, group dropdown, tags input, auth method tabs (key paste / generate / password). On save, `hosts.create` or `hosts.update`.
- **Key generation flow** — choose ed25519 / rsa / ecdsa, show public key with copy button, save private key encrypted to vault. Optional "save public key to ~/.ssh/" toggle for convenience.
- **Drag-drop key import** — drop a `.pem` / OpenSSH file onto the host drawer; `hosts.import` validates and stores.
- **Multi-tab terminals** — top-level tab strip. Each tab owns one xterm.js Terminal + one daemon `sessionId`. Closing a tab calls `session.close`. New-tab button presents the host picker. Tab labels track OSC 1/2 title sequences via the new daemon notification.
- **Tab tray + recent connections** — system-menu shortcut to reopen recent host with a hotkey.
- **Settings: vault** — change master password, lock, vacuum.
- **Settings: appearance** — light / dark, font family, font size, ligatures toggle for xterm.

**Out of scope for this phase** — sync, sign-in, teams, audit log UI, mount, file transfer, health dashboard.

**Acceptance**
- Everything a single user can do with a local-only TUI today is doable in the GUI.
- xterm.js is configured with WebGL renderer + Unicode 11 + image protocol stub. Resize, copy/paste, ctrl-c/ctrl-z all work.
- No memory leaks: open + close 50 tabs in a row, RAM stays flat (track via Activity Monitor).

---

### Phase 4 — Cloud sign-in + sync (weeks 8-10)

Goal: wire up the Convex device-code sign-in and the personal cloud vault.

**Sign-in**
- "Sign in" button on host list header. Click → call `auth.startSignIn` → daemon opens the browser to `/cli-auth/complete?session=…&code=…` (re-uses the existing web flow at `web/app/cli-auth/complete/page.tsx`).
- Renderer polls daemon's `auth.pollSignIn`. Once the daemon receives access + refresh tokens via `teamsclient.PollCLIAuth`, they're stored in `teamssession`.
- Renderer instantiates a Convex client with the access token and switches to "signed in" mode.
- Token rotation handled in the daemon; renderer is notified of token-expiry and re-uses the same connection.

**Personal cloud vault**
- New "Personal Cloud" tab under settings. Toggle to enable. UI shows: vault id, last sync, devices, recent sync events.
- Renderer subscribes to Convex queries (`personalVault:listItems`, `personalVault:listDevices`, `personalVault:listSyncEvents`) for real-time updates.
- Daemon owns the encryption: `sync.now` triggers a pull/push cycle that re-encrypts items with the user's vault key and writes to `personalVaultItems` via the existing `internal/sync` Convex provider. Renderer never sees plaintext.

**Sync provider settings**
- Provider chooser (off / git / cloud). Form for git: repo URL + branch + local path. Form for cloud: just a toggle since identity is implicit.
- "Sync now" button. Status badge with last result. Conflict resolution surface (existing logic in `internal/sync` already handles it; just expose results in the UI).

**Sign-out**
- Sign-out button → `auth.signOut` (revokes server-side session, clears `teamssession` locally). Renderer drops back to local-only mode.

**Acceptance**
- New device → sign in → personal vault syncs on first open.
- Add a host on device A, verify it appears on device B within a few seconds (real-time Convex sub).
- Sign out → device list on Convex shows the device removed (or marked inactive).
- Daemon log redacts tokens, vault key, and item plaintext.

---

### Phase 5 — Teams (weeks 10-12)

Goal: feature-match the existing `web/` Teams Dashboard inside the desktop app.

**Screens**
- **Team switcher** — dropdown in the top bar. Lists teams via `teams.list` (Convex sub for real-time). Switching context affects all "team-scoped" pages.
- **Members tab** — list of members + roles, invite form (`teams.invites.create`), accept incoming invites, role updates with role-gating UI (member can't change owner).
- **Hosts tab (team-shared)** — same UX as the personal host list, but team-scoped. Host drawer adds: credential mode (`shared` vs `per_member`), shared/per-member credential UI. "Reveal credential" button calls `hosts.revealCredential` (audit-logged).
- **Audit log tab** — virtualized timeline of `teamAuditEvents`. Filter by member, event type, date range. Click a row to see full event metadata.
- **Tokens tab** — list automation tokens, create new (with host scope), revoke, view recent execution audit. Maps to `tokens.list/create/revoke` and the existing automation token execution audit table.
- **Settings tab** — rename team, transfer ownership, leave team, danger zone.

**Role gating**
- `useTeamRole(teamId)` hook gates buttons / pages based on owner / admin / member.
- Permission-denied responses from Convex surface as toast errors with a retry hint.

**Acceptance**
- Every action available on `web/components/teams/TeamsDashboard.tsx` is available in the desktop app, with the same role gating.
- Audit log shows desktop-triggered actions identically to web-triggered ones (because both go through the same Convex mutations).

---

### Phase 6 — Power features (weeks 12-14)

**Health dashboard**
- New "Health" tab on each host detail. Shows latest probe (status, latency, uptime, CPU, RAM, disk). Manual "Probe now" button. Background scheduler pings hosts marked "watch" every N minutes (configurable in settings).
- Status pill on the host list reflects latest health check.

**SSHFS mount**
- "Mount" button on host detail. Drawer to choose remote path + local mount point. `mount.start` shells out to `sshfs` (or `sshfs-fuse-t` on Mac). "Show in Finder" / "Open in terminal" actions on a mounted host.
- `mount.list` shows currently-mounted hosts in a system-menu submenu.
- Pre-flight checks via `mount.checkPrereqs` — surface missing binaries with a docs link.

**File transfer**
- Drag-drop a local file onto a host row → `transfer.upload` (with destination prompt). Drag a remote file from a session → `transfer.download`.
- A small progress UI fed by `transfer.progress` notifications. Cancel button.
- Use this for `internal/ssh/transfer.go`'s SCP-style upload/download. SFTP browser is **out of scope** for v1 (it's a real product on its own).

**One-shot exec**
- Cmd-K palette (later) and a "Run command" action on hosts that invokes `session.exec`. Output rendered as a read-only terminal-ish view. Useful for "run `df -h` on all my prod boxes" patterns.

**Acceptance**
- Each power feature is reachable from the host list with at most two clicks.
- Background health scheduler doesn't peg the daemon's CPU; uses a goroutine-pool with conservative concurrency.
- Cleanup is enforced: every transfer / mount / exec has a deterministic cleanup path including the temp-key-file invariant from `ssh.NewTempKeyFile.Cleanup()`.

---

### Phase 7 — Cross-OS validation + packaging (weeks 14-16)

Goal: ship a signed, distributable build for macOS + Windows + Linux. **Do not start this phase until Phases 0-6 are merged and the macOS app has been used internally for at least a week.**

**Windows**
- Verify named-pipe daemon path (`\\.\pipe\sshthing-${user}`) under `Microsoft/go-winio`.
- Verify ConPTY support via `creack/pty` on Windows. If broken, swap to a Windows-specific PTY (Microsoft's Win32 ConPTY directly via `go-winio`).
- Code signing: EV certificate, smartscreen handling. Start the cert process now (3-week lead time); don't block on it.
- Auto-update via `electron-updater`.

**Linux**
- Verify Unix socket paths under `XDG_RUNTIME_DIR`.
- Test `.deb`, `.rpm`, `.AppImage` builds. Skip Snap/Flatpak for v1.
- Wayland + X11 smoke test.

**macOS**
- Apple Developer ID code signing + notarization (~$99/yr, 1-day setup).
- Universal binary (arm64 + amd64) for the daemon and the Electron app.
- DMG installer with the right rsync exclusion list.

**electron-builder config**
- `desktop/electron-builder.yml` with per-OS targets, embedded daemon binary, code-signing config, auto-update channel pointing at GitHub Releases.
- CI matrix on GitHub Actions: build mac (universal) + win + linux on each tag.

**Public release v1.0**
- Public landing-page section linking to the desktop app downloads.
- Release notes covering everything since the spike.

**Acceptance**
- Fresh download → install → first-run vault create → connect to a host → all three OSes.
- Auto-update can pull v1.0 → v1.0.1 (test with a no-op release).

---

## Cross-cutting workstreams (run alongside all phases)

### Telemetry + crash reporting
- Sentry or self-hosted equivalent. Renderer + main + daemon.
- Strict redaction: never log passwords, vault key, decrypted secrets, ssh stdout/stderr, audit-event PII.
- Opt-in only on first run.

### Internal dogfood
- After Phase 3 lands, the team (and you) use the GUI as the primary client. Bug reports go into a dogfood channel.
- After Phase 5, hand it to 5-10 friendly external testers.

### Documentation
- Treat `desktop/README.md` as living. Each phase updates it.
- New file `docs/architecture/desktop.md` after Phase 2 that documents the daemon RPC surface (auto-generated from Go source if possible).

### Tests
- Go: unit tests around every service. Integration tests using a temp vault for happy paths.
- TS: Vitest for renderer logic. Playwright (Electron mode) for one end-to-end smoke test per phase.
- No coverage gate, but no untested service either.

---

## Risks I'm watching

1. **TUI refactor lands invisible regressions.** Mitigation: keep a written list of every TUI flow before refactor, test each manually before declaring Phase 2 done.
2. **xterm.js memory leaks across tabs.** Mitigation: explicit `term.dispose()` on tab close + the open-50-tabs smoke test in Phase 3.
3. **Daemon-spawn lifecycle bugs** (e.g., orphaned daemon, stale lockfile, `ELECTRON_RUN_AS_NODE` recurrence). Mitigation: a single `internal/daemon/launcher` package with comprehensive tests for cold-start / warm-start / crash-restart.
4. **Convex token hand-off in Electron.** The renderer needs the access token to talk to Convex; the daemon owns it. Plan: daemon exposes `auth.tokenForRenderer()` that returns the *access* token (short-lived, low blast radius). Refresh token never leaves the daemon.
5. **Windows ConPTY.** The known wildcard. Mitigation: cut a Windows VM and run the spike on it before Phase 7 — early signal at end of Phase 3.
6. **EV cert lead time.** Don't let it become the bottleneck — buy it the week we start Phase 0.
7. **Scope creep on power features.** SFTP browser, command palette, plugin system, AI command bar are tempting but explicitly out of scope. Defer to v1.1.

## Out of scope for v1 (explicit)

- Mobile apps (iOS, Android).
- Plugin / extension system.
- AI / agentic features (the current TUI's `agent-host-map.md` is per-user, not productized).
- SFTP file browser — real product, real scope, separate plan.
- Self-hosted Convex / on-prem mode.
- Snap / Flatpak distribution.
- Browser-based desktop variant (use `web/` for that).

## Acceptance bar for v1.0

Every item below must be true on macOS + Windows + Linux before tagging:
1. Fresh install → first-run → connect to a host without a manual workaround.
2. Sign in → personal cloud sync survives a daemon restart, an Electron restart, and a network blip.
3. Switch teams → host list updates → audit log timeline shows past actions.
4. Open 10 tabs to different hosts simultaneously, run `htop`, no jank.
5. Drag-drop import a key, verify it works for SSH.
6. Probe a host's health, verify the metrics match `htop` / `df -h` on the host.
7. Mount a remote dir via sshfs, list its contents, unmount cleanly.
8. Auto-update from previous build to current build.
9. Daemon log redacts every secret class.
10. Code-signing valid on all three OSes.

If we hit all 10, we ship. If we miss one, we cut its phase scope and re-bake.
