# Security Review (Developer Notes)

This document is a best‑effort review of SSHThing’s security posture based on the current codebase. It is **not** a professional audit. Treat it as a checklist of risks and hardening ideas.

## What SSHThing Protects

- **At rest:** host metadata + private keys stored in an **encrypted SQLCipher** DB (`~/.ssh-manager/hosts.db`), plus a second layer of **AES‑GCM** encryption for each key blob.
- **In use:** decrypted keys are materialized to disk for `ssh`/`sftp`/`sshfs` (required by those tools), then cleaned up.
- **In sync:** private keys remain AES-GCM encrypted in the sync file; only the encryption salt is shared to allow re-encryption on import.

## Trust Boundaries

- The app shells out to system tools: `ssh`, `sftp`, `ssh-keygen`, `sshfs` (FUSE‑T), `umount`, `mount`, `open`.
- Hostnames/ports/usernames come from user input and are passed as **exec arguments** (no shell interpolation), reducing command‑injection risk.

## Findings & Risks

### High

1) **First‑connect MITM risk (`StrictHostKeyChecking=accept-new`)**
   - SSHThing uses `StrictHostKeyChecking=accept-new` for SSH/SFTP and passes similar options to SSHFS.
   - This avoids interactive prompts but **trusts the first key you see**. If a user is attacked on the first connection, a hostile key can be persisted in `known_hosts`.
   - Mitigation: allow users to choose strict checking (`yes`) and/or pin expected host keys per host.

2) **Mount “Leave Mounted” stores a decrypted private key on disk**
   - To allow Finder mounts to persist after quitting, SSHThing writes a per-host key file under `~/.config/sshthing/mount-keys/` (0600) until unmount.
   - If the machine/user account is compromised while mounted, that key file can be used to authenticate.
   - Mitigation: prefer SSH agent integration; prompt users clearly; auto-expire or require a passphrase; store the key in Keychain where feasible.

### Medium

3) **SQLCipher key handling via DSN parameter**
   - The SQLCipher key is embedded into the DSN (`_pragma_key=x'...'`). While this is not printed by default, it can leak via logging, crash dumps, or debugging tooling.
   - Mitigation: avoid logging DSNs; consider passing a passphrase directly to SQLCipher PRAGMA after opening; consider per‑DB random salt/KDF settings.

4) **Fixed salt for SQLCipher pre-derivation**
   - The app derives a hex key from the master password using PBKDF2 with a **fixed** salt before passing it to SQLCipher.
   - Fixed salt means identical passwords yield identical derived keys across users/installs, which is sub‑optimal for offline attack resistance.
   - Mitigation: use a per‑DB random salt (stored in the DB header/config) or let SQLCipher manage the passphrase/KDF directly.

5) **Secrets remain in process memory**
   - Passwords/derived keys/decrypted private keys are held in Go memory as `string`/`[]byte` and are not reliably zeroized (Go GC copies/moves data).
   - Mitigation: minimize secret lifetime; avoid duplicating strings; consider best‑effort zeroing of byte slices; document limitations.

6) **Temp-file “secure delete” is best-effort**
   - Key files are overwritten with zeros before removal, but on modern SSD/APFS this is **not guaranteed** to prevent recovery (copy‑on‑write, journaling, snapshots).
   - Mitigation: prefer agent/keychain; use memory-backed fs where available; accept as a documented limitation.

### Low

7) **Generated keys are unencrypted (no passphrase)**
   - `ssh-keygen -N ""` creates a passphrase-less private key. The app encrypts it at rest, but when exported/moved outside the DB it’s unprotected.
   - Mitigation: offer an optional passphrase (stored nowhere, user enters on connect via agent) or encourage agent usage.

8) **UI can expose private keys**
   - Edit host currently pre-populates the private key text area for copy/edit, which can leak into screen recordings, terminal scrollback, or shoulder-surfing.
   - Mitigation: hide by default (“Reveal key”); warn the user; avoid rendering full key unless requested.

9) **Mount/DB directories rely on local account security**
   - Directories are created with 0700 and key files with 0600, which is good, but threat model still assumes local user account is trusted.
   - Mitigation: document the assumption; add warnings in the mount “beta” screen.

## Suggested Next Hardening Steps

- Add a per-host “Host key policy” setting (accept-new vs strict vs pinned fingerprint).
- Integrate with `ssh-agent` / macOS Keychain to avoid writing decrypted keys to disk where possible.
- Add a “Reveal private key” toggle and avoid rendering keys by default in edit views.
- Consider per-DB random salt / KDF parameters for SQLCipher instead of fixed-salt pre-derivation.

