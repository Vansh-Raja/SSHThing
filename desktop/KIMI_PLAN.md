# Kimi Desktop Polish Plan

> Based on explore-agent audits of the actual code. Claude's IMPLEMENTATION_TODO.md is ~60% out of date — many "missing" items are already shipped. This plan covers only the *real* gaps.

## Verified Baseline (already done — do NOT rebuild)
- All 75 daemon RPCs, Electron IPC bridge, preload API
- Renderer pages: Unlock, Hosts, Teams, SignIn, Account, Keys, Settings
- Multi-tab xterm.js, command palette, help overlay, drag-drop key import, tags chips, group rename/delete, confirm dialogs, empty states, skeletons, theme picker, audit filters, exec history, batch exec, transfer cancel, mount start/stop

## Phase 1 — Fix-It Friday (parallel swarm, ~1 day)
High-bug-surface, low-effort fixes that unblock everything else.

### 1A. Daemon + IPC holes
- `vault.vacuum` — RPC handler exists in `methods_vault.go`? **No** — add it.
- `hosts.updateWithKey` — daemon has it, but **not wired** in `main/index.ts` or `daemon.ts`.
- `mount.checkPrereqs` — daemon has it, **not wired** in IPC.
- `keyring.healthCheck` — daemon has it, **not wired** in IPC.
- `sync.events` / `sync.devices` / `sync.forgetDevice` / `sync.testGit` — daemon has them, **not wired** in IPC.
- `vault.locked` notification — **never emitted** by daemon. Emit it when `vault.lock` is called.

### 1B. Type-sync + renderer micro-bugs
- `sshthing.d.ts` missing: `chooseDirectory`, `transferCancel`
- `sshthing.d.ts` drift: remove `gpuPresent`/`gpuName` from `HealthResult` (daemon never populates)
- `useDaemonHealth.ts` broken — references `onDaemonExit` (doesn't exist); should use `onDaemonExited`
- InvitesBadge — `useIncomingInvites.ts` hook exists but badge is **not mounted** in `Topbar.tsx`

## Phase 2 — v1 Polish (parallel swarm, ~2–3 days)
Features that make the app feel finished.

### 2A. Window state + app identity
- `electron-window-state` — persist window size/position
- App icon set for macOS (.icns) + placeholder paths for Win/Linux
- Custom About dialog (version, daemon version, license) replacing native `{ role: 'about' }`
- First-run welcome screen (one-time overlay after vault creation)

### 2B. Mount lifecycle
- `MountDrawer.tsx` — prereq check UI using wired `mount.checkPrereqs` RPC
- Restore mounts on startup (call `mountList` on app launch, re-mount active ones)
- Mount-aware quit dialog in `before-quit` (respects `Mount.QuitBehavior` config)
- "Show in Finder" action on mounted hosts via `system:openPath`

### 2C. Auto-update surface
- Add `electron-updater` dependency
- Settings section: update channel (stable/beta), auto-apply toggle
- Topbar banner when update available
- Apply-update + restart flow

### 2D. Teams polish
- Reorder teams UI (drag or move-up/down buttons)
- Decline incoming invite action
- Transfer ownership flow (replace "Coming soon" stub)
- Token execution audit view per token

## Phase 3 — Sync Depth (parallel swarm, ~3–4 days)
- Conflict resolution UI after `sync.now` detects server-side changes
- Devices list + sync events timeline in Settings
- Scope toggles UI (hosts/credentials/token defs/health/mount-state)
- Git sync setup wizard (URL, branch, SSH key picker, test-connection)

## Phase 4 — Test + Packaging (parallel swarm, ~2–3 days)
- Vitest scaffolding + one smoke test for `Unlock.tsx`
- Playwright e2e: unlock → connect → edit host
- `electron-builder.yml` with per-OS targets
- CI matrix stub (GitHub Actions workflow)

## Execution Strategy
Use **swarm mode**: each phase launches N agents in parallel, each owning one (A–D) workstream. Agents only read/write their assigned files. I (Kimi) act as coordinator: review diffs, resolve conflicts, run builds/tests.
