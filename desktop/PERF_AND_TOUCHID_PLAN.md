# Snappy-feel + Touch ID + TUI Bundling — Implementation Plan

## Decisions locked in

- **Touch ID expiry:** fixed 7 days from the first password unlock. After that, password required again.
- **Touch ID UX:** auto-prompt the macOS biometric dialog the moment the app opens (when set up + not expired). User can cancel → falls back to password screen.
- **Touch ID implementation:** bundled Swift helper binary `sshthing-biometric` (~100 KB) that the daemon shells out to. Cleanest separation, no cgo.
- **Platform scope:** macOS only for v1. Windows Hello + Linux PAM/fprint deferred.
- **TUI:** bundled inside the .app and exposed via a `/usr/local/bin/sshthing` symlink the user can install with one click.

The **single biggest perceived win** is Touch ID + warm-unlock — it removes the 200–500 ms PBKDF2 cost on every app open. Stale-while-revalidate is the second biggest. Together these should make cold open ~80 ms to first paint, hot open ~40 ms.

---

## Phase A — TUI bundling (small, isolated)

Goal: ship `sshthing` (the TUI binary) inside the .app and let users invoke it from a terminal as `sshthing`.

### Steps

1. **Build the TUI alongside the daemon.** Update `desktop/package.json` `daemon:build` script to also produce `bin/sshthing`:
   ```json
   "daemon:build": "cd .. && \
     go build -ldflags='-X github.com/Vansh-Raja/SSHThing/internal/cloud.DefaultBaseURL=https://testsshthing.vanshraja.me' \
       -o desktop/bin/sshthing-daemon ./cmd/sshthing-daemon && \
     go build -ldflags='-X github.com/Vansh-Raja/SSHThing/internal/app.defaultCloudBaseURL=https://testsshthing.vanshraja.me' \
       -o desktop/bin/sshthing ./cmd/sshthing"
   ```

2. **Add the binary to the bundle.** Extend `desktop/electron-builder.yml`:
   ```yaml
   extraResources:
     - from: bin/sshthing-daemon
       to:   bin/sshthing-daemon
     - from: bin/sshthing
       to:   bin/sshthing
   ```

3. **Surface the binary inside the GUI.** Settings → "Command-line tools" section:
   - Button: **"Install `sshthing` command"** → calls a new IPC handler `system:install-cli-symlink` that runs `osascript … with administrator privileges` to `ln -sf` the bundled binary to `/usr/local/bin/sshthing`. Idempotent.
   - Button: **"Open in Terminal"** — opens Terminal.app at the bundled binary path, runs the TUI in a new window.

4. **macOS app menu integration.** Add an "Open TUI in Terminal" item under the SSHThing menu so users discover it.

5. **README docs.** Tell users that installing the GUI also installs the TUI.

**Acceptance**: after running `pnpm dist:mac` and installing the new app, both the GUI and `sshthing` work. Settings → Install CLI works idempotently.

---

## Phase B — Touch ID auth

### B1. Swift helper binary

Create a new Swift source tree at `mac-helpers/sshthing-biometric/`:

- **Project layout** — single Swift source file + a `Package.swift` for `swift build`.
- **Subcommands**:
  | Command | What it does |
  |---|---|
  | `available` | exit 0 if `LAContext.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics)` succeeds, else exit 1 |
  | `store --service S --account A` | reads stdin, stores it in macOS Keychain at `(S, A)` with `SecAccessControl(.biometryCurrentSet)` so reading it requires Touch ID |
  | `fetch --service S --account A --reason "Unlock SSHThing"` | reads the keychain item — this triggers the Touch ID prompt automatically. Prints the secret to stdout on success |
  | `forget --service S --account A` | deletes the item |

- **Build** with `swift build -c release --arch arm64 --arch x86_64` to produce a universal binary.

- **Bundle**: `daemon:build` script also runs `swift build` and copies the resulting binary to `desktop/bin/sshthing-biometric`. `electron-builder.yml` adds a third `extraResources` entry.

- **Code signing**: ad-hoc signed in dev (the same `codesign --force --deep --sign -` pass we ran on the .app picks it up since it's in `Contents/Resources/bin/`). Touch ID won't work on properly-notarized builds without entitlements; v1 ships unsigned-locally so this is fine. Note for v1.1: production needs `keychain-access-groups` entitlement.

### B2. Daemon integration

New Go file `internal/securestore/biometric.go`:

- `BiometricAvailable() bool` — runs `sshthing-biometric available` (binary path resolved via env var `SSHTHING_BIOMETRIC_BIN` or sibling-of-daemon lookup).
- `BiometricStore(password string) error` — pipes password to `sshthing-biometric store`.
- `BiometricFetch() (string, error)` — runs `sshthing-biometric fetch`. Blocks until user touches sensor or cancels. Returns the password.
- `BiometricForget() error` — runs `sshthing-biometric forget`.

### B3. Config additions

`internal/config/config.go` — extend the `Vault` section (already exists) or `Auth` section:
```go
type AuthSection struct {
    BiometricEnabled  bool  `json:"biometric_enabled"`
    BiometricExpiry   int64 `json:"biometric_expiry"` // unix seconds
}
```

Migration: existing configs default to `false` / `0`. No version bump needed (additive only).

### B4. New RPCs

| Method | Behaviour |
|---|---|
| `vault.biometricStatus` | `{available, enabled, expiry}` — non-prompting, used by the renderer to decide whether to auto-trigger the prompt |
| `vault.enableBiometric({password})` | Verifies password by trying to unlock the vault. On success: stores the password via `BiometricStore`, sets `BiometricEnabled=true`, `BiometricExpiry=now+7d`. Returns `{ok: true, expiresAt}` |
| `vault.disableBiometric` | Calls `BiometricForget`, clears flags |
| `vault.unlockWithBiometric` | Checks `BiometricEnabled && !expired`. If not, returns error. Calls `BiometricFetch` (this triggers Touch ID prompt). On success, unlocks the vault using the fetched password. Returns the same shape as `vault.unlock` |

The `vault.unlockWithBiometric` RPC is the only one that prompts. It's called explicitly by the renderer; never automatically by another RPC.

### B5. Renderer flow

Modify `Unlock.tsx` and `App.tsx`:

1. On mount, renderer calls `vault.status()` (cheap, no prompt) AND `vault.biometricStatus()` in parallel.
2. If already unlocked → go to /hosts.
3. Else if `biometricStatus.enabled && !expired`:
   - Show a placeholder unlock screen with "Touch ID to unlock" + a small fingerprint icon
   - Auto-fire `vault.unlockWithBiometric()` (which causes the macOS Touch ID prompt to appear)
   - On success → /hosts
   - On user-cancel or hardware fail → swap the placeholder for the password input
4. Else → show password screen as today.

After successful password unlock, if `biometricStatus.available && !enabled`, show a one-time toast: **"Enable Touch ID? You won't need to type your password for the next 7 days."** with Enable / Not now buttons. Enable calls `vault.enableBiometric({password})`.

### B6. Settings UI

Settings → Vault gains a new row:
- "Use Touch ID to unlock" toggle
- Below: "Active until <date>" when enabled
- Disable clears the keychain item via `vault.disableBiometric`
- Re-enable prompts for password again

### Acceptance — Phase B

- First app launch after install: password screen, no Touch ID option.
- After first password unlock + accepting the toast: Touch ID enrolled.
- Quit app, relaunch: Touch ID dialog pops up automatically, fingerprint unlocks straight to /hosts in <1 s.
- 7 days later: dialog stops appearing; password screen returns.
- Cancelling Touch ID dialog falls back to password screen without errors.

---

## Phase C — Renderer perf wins (snappy UI)

### C1. Stale-while-revalidate cache for hosts

Largest perceived improvement. New `desktop/src/renderer/hooks/useHostsCache.ts`:

- `useHostsCache()` returns `{hosts, loading, refresh}`.
- On mount: synchronously read from `localStorage["sshthing.hosts.v1"]` (keyed by vault salt, see C5) → render those immediately.
- In parallel: fire `window.sshthing.listHosts()`. When it resolves, diff vs cache, update state, write fresh JSON back to localStorage.
- Subsequent host CRUD operations call `refresh()` which re-fetches.

Replace **both** existing `listHosts` callers (`App.tsx` palette load + `Hosts.tsx` `loadHosts`) with this hook. Removes the duplicate fetch.

### C2. Parallel + deferred bootstrap

In `Hosts.tsx` today, the bootstrap fans out across multiple `useEffect`s that all fire on mount but each kicks off an independent fetch. Replace with one effect that:

```ts
useEffect(() => {
  // critical path
  void hosts.refresh();
  void groups.refresh();

  // deferred — these are nice-to-have, not blocking first paint
  const idle = window.requestIdleCallback ?? ((cb: IdleRequestCallback) => setTimeout(() => cb({} as IdleDeadline), 200));
  idle(() => {
    void health.loadAll();
    void mounts.loadMounts();
    void teams.reload();
    void invites.reload();
    void sync.refresh();
  });
}, []);
```

Net effect: hosts paint first; team / sync / invite calls happen after the layout is on screen.

### C3. Route-level code splitting

Wrap heavy routes with `React.lazy`:

```tsx
const Settings = lazy(() => import('./pages/Settings'));
const Teams    = lazy(() => import('./pages/Teams'));
const Account  = lazy(() => import('./pages/Account'));
const Keys     = lazy(() => import('./pages/Keys'));
```

Wrap `<Outlet />` in `<Suspense fallback={null}>`. Hosts page stays eager.

Vite will produce per-route chunks. The initial bundle drops from 730 KB to ~400 KB. JS parse time improves proportionally.

### C4. Settings RPC: avoid the round-trip on first paint

`useSyncStatus` and `useTeams` poll `getSettings`/`teamsList` on mount. Move these behind the deferred-bootstrap idle callback (C2) — they're for the topbar's sync indicator and team switcher, neither of which is critical for first paint.

### C5. Cache invalidation

Cache key is `sshthing.hosts.v1.<vaultSalt>`. The vault salt is already returned from `vault.unlock` and `vault.status` — store it in localStorage too, scope all caches by it. On vault-locked notification → `localStorage.removeItem(...)` for all keyed entries. Prevents one user's hosts leaking into another vault.

### C6. Daemon-side memoization (optional)

`internal/daemon/service/hosts.go` `ListSummary` already wraps `store.GetHosts()`. Add a `sync.RWMutex`-protected memoized slice that's invalidated on Create/Update/Delete. SQLite is already plenty fast (<5 ms for typical row counts) so this is small (~1–3 ms saved per call) — leave for last unless profiling shows otherwise.

### Acceptance — Phase C

Use Chrome DevTools' Performance tab via Electron's DevTools (`Cmd+Opt+I`):
- Cold open (first launch after build): hosts list visible within 250 ms of typing the password.
- Warm open with Touch ID: hosts list visible within 80 ms of fingerprint.
- Re-opening after quit but within 7 days: same as warm open.

---

## Phase D — Daemon pre-warm (optional, only if Phase A–C don't get us there)

Two options, in order of preference:

1. **Eager daemon spawn in Electron main**: today the daemon starts in `app.whenReady().then(...)`. Move the `startDaemon()` call to fire as early as possible — **before** `whenReady` if Electron allows. This shaves ~100 ms by overlapping daemon spawn with Chromium startup.

2. **macOS LaunchAgent that pre-spawns the daemon at user login**: ship a `~/Library/LaunchAgents/com.sshthing.daemon.plist`. Trade-off: the daemon is always running in the background even when the app isn't open. Memory cost ~30–50 MB. Probably overkill; skip unless 1+B+C aren't fast enough.

---

## Sequencing & order I'd actually do this in

```
Day 1  ─ Phase A (TUI bundling)              ~half day
         Phase B1 + B2 (Swift helper + Go bindings)   ~half day

Day 2  ─ Phase B3, B4, B5, B6 (Touch ID end-to-end)   ~one full day

Day 3  ─ Phase C1 + C2 (cache + parallel bootstrap)   ~half day
         Phase C3 (lazy routes)               ~quarter day
         Phase C4 + C5 (defer + invalidate)   ~quarter day

Day 4  ─ Profile, tighten, ship 1.0 of the snappy build
         Phase D only if measurements still show > 200 ms cold open
```

## What I'd verify before declaring done

1. **Cold cold open** (after a reboot, fresh machine): time from app icon click → host list pixels. Target <500 ms.
2. **Warm open with Touch ID**: same metric. Target <120 ms.
3. **Settings persistence still works** (no regression from the merge fix earlier).
4. **Mount + connect** still work with the new bootstrap order.
5. **Touch ID 7-day expiry** is honoured: rewind the system clock past the expiry, confirm password is required.
6. **Cancelling Touch ID** falls back to password without weird state.
7. **TUI** launches both via the `/usr/local/bin/sshthing` symlink and the GUI's "Open in Terminal" action.

## Risks & callouts

- **Touch ID requires the binary to be runnable** — when the .app is moved between Macs without proper signing, the helper may silently fail. Document the limitation; the v1 plan is for the user's own builds.
- **macOS may store the Touch ID consent per-binary path** — bundling `sshthing-biometric` inside the .app is critical because the consent is keyed by the calling binary. Don't move the helper around at runtime.
- **Renderer cache and a stale daemon DB** — if a user manually edits the SQLCipher file (very rare), the cache will show stale data until a refresh. Accept; document.
- **Lazy routes change error UX** — code-split chunks fail to load (rare, but possible offline). Add an error boundary that retries.

## Out of scope (do not do here)

- Windows Hello / Linux fingerprint.
- Switching SQLCipher KDF iterations to make unlock faster (security tradeoff).
- Replacing the existing JSON-RPC socket with anything else.
- TUI ↔ daemon refactor (still a good idea long-term; not part of this pass).
