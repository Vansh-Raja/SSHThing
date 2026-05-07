# SSHThing Desktop

Electron + React + Vite + Go daemon. Phase 0 + Phase 3 complete.

## Architecture

```
desktop/
  src/main/       Node process: spawns daemon, registers IPC handlers
  src/preload/    contextBridge → window.sshthing (typed API surface)
  src/renderer/   React + Vite app: pages, components, UI primitives
  dist/           Compiled output (git-ignored)
  bin/            Daemon binary for dev workflow (git-ignored)
```

Wire protocol: newline-delimited JSON-RPC 2.0. Every request carries an `auth`
field with the 32-byte hex token written to `daemon.token` at startup.

## Prerequisites

- Go 1.25+ with CGO enabled (for go-sqlcipher)
- Node.js 20+
- pnpm (`npm install -g pnpm`)
- Existing SSHThing vault — run the TUI once to create it if absent.

## Build

```sh
# 1. Install JS deps (desktop is standalone, not in the workspace)
cd desktop
pnpm install --ignore-workspace

# 2. Build Go daemon
pnpm run daemon:build

# 3. Build everything (main + preload + renderer via Vite)
pnpm run build

# 4. Typecheck (renderer — no emit)
pnpm run typecheck

# 5. Dev HMR for renderer (not wired to Electron; useful for CSS/component iteration)
pnpm run dev:renderer
```

## Launch

```sh
# Build first, then:
pnpm run dev
# or: electron .
```

Electron window flow:
1. Spawns `desktop/bin/sshthing-daemon`
2. Reads token from `~/Library/Application Support/SSHThing/daemon.token`
3. Connects to daemon socket
4. Loads `dist/renderer/index.html` via `file://` (Vite's `base: './'` makes paths relative)
5. Renderer checks vault status → routes to `/unlock` or `/hosts`

## Component tree

```
App (HashRouter)
  ├── /unlock → Unlock
  │     └── PasswordField, Button
  ├── /hosts → Hosts (Phase 3A/3B/3C)
  │     ├── Sidebar
  │     │     ├── HostList
  │     │     │     ├── Tag, DropdownMenu
  │     │     │     └── Dialog (delete confirm)
  │     │     └── RevealCredentialModal (Modal + Spinner)
  │     ├── Tabs (Radix Tabs — tab strip)
  │     │     └── TerminalTab × N (xterm.js + FitAddon + ResizeObserver)
  │     ├── HostDrawer (Drawer)
  │     │     ├── TextField, PasswordField, Select, Tag, Button
  │     │     └── key gen (generateKey RPC) + drag-drop .pem import
  │     └── CommandPalette (Fuse.js fuzzy, portal overlay)
  ├── /settings → Settings (Phase 3D)
  │     ├── Vault: change password, lock, vacuum
  │     ├── Appearance: theme (light/dark/system), font size
  │     ├── SSH Defaults: term type, keep-alive, host-key policy
  │     └── Sync: stub UI (off / git / cloud)
  ├── /sign-in → placeholder (Phase 4)
  └── /teams → placeholder (Phase 5)
Toaster (portal, imperative toast.success/error/info)
```

## IPC channels (renderer → daemon RPC)

| IPC channel | Daemon RPC | Status |
|---|---|---|
| `vault:unlock` | `vault.unlock` | Implemented |
| `vault:status` | `vault.status` | Implemented |
| `vault:create` | `vault.create` | Phase 1 |
| `vault:changePassword` | `vault.changePassword` | Phase 1 |
| `vault:lock` | `vault.lock` | Phase 1 |
| `vault:vacuum` | `vault.vacuum` | Phase 1 |
| `hosts:list` | `hosts.list` | Implemented |
| `hosts:get` | `hosts.get` | Implemented |
| `hosts:create` | `hosts.create` | Phase 1 |
| `hosts:update` | `hosts.update` | Phase 1 |
| `hosts:delete` | `hosts.delete` | Phase 1 |
| `hosts:revealCredential` | `hosts.revealCredential` | Phase 1 |
| `hosts:generateKey` | `hosts.generateKey` | Phase 1 |
| `hosts:importKey` | `hosts.import` | Phase 1 |
| `groups:list` | `groups.list` | Phase 1 |
| `groups:create` | `groups.create` | Phase 1 |
| `groups:rename` | `groups.rename` | Phase 1 |
| `groups:delete` | `groups.delete` | Phase 1 |
| `session:open` | `session.open` | Implemented |
| `session:write` | `session.write` | Implemented |
| `session:resize` | `session.resize` | Implemented |
| `session:close` | `session.close` | Implemented |
| `session:list` | `session.list` | Phase 1 |
| `settings:get` | `settings.get` | Phase 1 |
| `settings:set` | `settings.set` | Phase 1 |

Daemon push notifications consumed by renderer:
- `session.data` — terminal output bytes (base64)
- `session.exit` — session exited with code
- `session.titleChanged` — OSC title update (Phase 1 daemon side)
- `vault.locked` — TTL expired → redirect to /unlock

Phase 1 RPCs return `-32601 method not found`; the renderer catches these and
shows a friendly toast rather than crashing.

## Memory leak discipline (multi-tab terminal)

Each `TerminalTab` component owns one Terminal, one FitAddon, one ResizeObserver,
and one notification subscription — all stored in refs, not state.

Tab close sequence:
1. `session.close(sessionId)` called first
2. Tab removed from state array → `TerminalTab` unmounts
3. Cleanup effect runs: `unsub()` + `ro.disconnect()` + `term.dispose()`

**50-tab smoke test (requires display):**
1. Open 50 tabs via Cmd+T → select host → repeat
2. Close all via Cmd+W × 50
3. Open Activity Monitor, observe renderer process RAM
4. RAM should return within 20 MB of baseline after GC

## TODOs for follow-up

- All Phase 1 daemon RPCs (tracked in IMPLEMENTATION_TODO.md §Phase 1)
- Replace deprecated `xterm` package with `@xterm/xterm` + add `@xterm/addon-webgl`
- Code-split xterm.js into a separate chunk (currently 531 KB bundle)
- First-run vault creation UI (currently requires TUI to create vault first)
- Phase 4: sign-in, personal cloud sync
- Phase 5: teams UI
- Phase 6: health dashboard, SSHFS mount, file transfer
- Phase 7: cross-OS packaging, code signing
