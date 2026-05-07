#!/usr/bin/env bash
# smoke.sh — Smoke test for the sshthing-daemon RPC surface (Phase 1).
#
# Usage:
#   ./desktop/scripts/smoke.sh
#
# Requires:
#   - nc (netcat with Unix socket support: nc -U)  OR socat
#   - jq (optional, for pretty output)
#   - The daemon binary built and accessible: go build ./cmd/sshthing-daemon
#
# The script:
#   1. Starts the daemon in the background.
#   2. Reads the auth token from the token file.
#   3. Fires a series of RPC calls over the Unix socket.
#   4. Checks each response for absence of "error" field.
#   5. Kills the daemon and reports pass/fail.
#
set -euo pipefail

BINARY="${BINARY:-./cmd/sshthing-daemon/sshthing-daemon}"
SOCKET_PATH="${SSHTHING_SOCKET:-}"
DATA_DIR="${SSHTHING_DATA_DIR:-$(mktemp -d /tmp/sshthing-smoke-XXXXXX)}"

export SSHTHING_DATA_DIR="$DATA_DIR"

cleanup() {
    if [[ -n "${DAEMON_PID:-}" ]]; then
        kill "$DAEMON_PID" 2>/dev/null || true
        wait "$DAEMON_PID" 2>/dev/null || true
    fi
    rm -rf "$DATA_DIR"
}
trap cleanup EXIT

log() { echo "[smoke] $*" >&2; }
fail() { log "FAIL: $*"; exit 1; }
pass() { log "PASS: $*"; }

# ── Build ─────────────────────────────────────────────────────────────────────

if [[ ! -f "$BINARY" ]]; then
    log "Building daemon..."
    go build -o "$BINARY" ./cmd/sshthing-daemon/
fi

# ── Start daemon ──────────────────────────────────────────────────────────────

log "Starting daemon (DATA_DIR=$DATA_DIR)..."
"$BINARY" &
DAEMON_PID=$!
sleep 0.5  # Give it a moment to write the socket and token.

TOKEN_FILE="$DATA_DIR/daemon.token"
if [[ ! -f "$TOKEN_FILE" ]]; then
    fail "Token file not found at $TOKEN_FILE"
fi
TOKEN=$(cat "$TOKEN_FILE")
log "Auth token: ${TOKEN:0:8}..."

# Find socket path.
if [[ -z "$SOCKET_PATH" ]]; then
    # When SSHTHING_DATA_DIR is set (as it always is in this script), the daemon
    # writes daemon.sock inside that directory.
    SOCKET_PATH="$DATA_DIR/daemon.sock"
fi
log "Socket path: $SOCKET_PATH"

# ── RPC helper ────────────────────────────────────────────────────────────────

rpc() {
    local method="$1"
    local params="${2:-}"
    if [[ -z "$params" ]]; then params="{}"; fi
    local payload
    payload=$(printf '{"jsonrpc":"2.0","id":1,"auth":"%s","method":"%s","params":%s}\n' \
        "$TOKEN" "$method" "$params")

    local response
    if command -v socat &>/dev/null; then
        response=$(printf '%s\n' "$payload" | socat -T2 - "UNIX-CONNECT:$SOCKET_PATH" 2>/dev/null || true)
    else
        # BSD nc (macOS): -U = Unix socket, -w2 = 2-second idle timeout.
        # GNU nc (Linux): replace -w2 with -q1 if BSD nc is unavailable.
        response=$(printf '%s\n' "$payload" | nc -U -w2 "$SOCKET_PATH" 2>/dev/null || true)
    fi

    if [[ -z "$response" ]]; then
        fail "$method: empty response"
    fi

    if echo "$response" | grep -q '"error"'; then
        fail "$method: got error: $response"
    fi

    if command -v jq &>/dev/null; then
        echo "$response" | jq -c .result
    else
        echo "$response"
    fi
}

# ── Tests ─────────────────────────────────────────────────────────────────────

log "--- daemon.version ---"
rpc "daemon.version"
pass "daemon.version"

log "--- daemon.health ---"
rpc "daemon.health"
pass "daemon.health"

log "--- vault.status (locked) ---"
rpc "vault.status"
pass "vault.status"

log "--- vault.create ---"
rpc "vault.create" '{"password":"smoketest123"}'
pass "vault.create"

log "--- vault.status (unlocked) ---"
rpc "vault.status"
pass "vault.status"

log "--- hosts.list ---"
rpc "hosts.list" '{}'
pass "hosts.list"

log "--- groups.list ---"
rpc "groups.list"
pass "groups.list"

log "--- groups.create ---"
rpc "groups.create" '{"name":"smoke-group"}'
pass "groups.create"

log "--- groups.list (after create) ---"
rpc "groups.list"
pass "groups.list after create"

log "--- groups.rename ---"
rpc "groups.rename" '{"old":"smoke-group","new":"smoke-group-2"}'
pass "groups.rename"

log "--- groups.delete ---"
rpc "groups.delete" '{"name":"smoke-group-2"}'
pass "groups.delete"

log "--- settings.get ---"
rpc "settings.get"
pass "settings.get"

log "--- session.list ---"
rpc "session.list"
pass "session.list"

log "--- health.list ---"
rpc "health.list"
pass "health.list"

log "--- mount.list ---"
rpc "mount.list"
pass "mount.list"

log "--- tokens.list ---"
rpc "tokens.list"
pass "tokens.list"

log "--- teams.list (not signed in — expect not_signed_in error) ---"
TEAMS_RESP=$(printf '{"jsonrpc":"2.0","id":1,"auth":"%s","method":"teams.list","params":{}}\n' "$TOKEN" \
    | (command -v socat &>/dev/null && socat -T2 - "UNIX-CONNECT:$SOCKET_PATH" || nc -U -w2 "$SOCKET_PATH") 2>/dev/null || true)
if echo "$TEAMS_RESP" | grep -q '"code":-32030\|"not signed in"'; then
    pass "teams.list → not_signed_in (expected)"
else
    fail "teams.list: unexpected response: $TEAMS_RESP"
fi

log "--- auth.session (not signed in — expect null session) ---"
AUTH_SESS_RESP=$(printf '{"jsonrpc":"2.0","id":1,"auth":"%s","method":"auth.session","params":{}}\n' "$TOKEN" \
    | (command -v socat &>/dev/null && socat -T2 - "UNIX-CONNECT:$SOCKET_PATH" || nc -U -w2 "$SOCKET_PATH") 2>/dev/null || true)
if echo "$AUTH_SESS_RESP" | grep -q '"session":null\|"session":{}'; then
    pass "auth.session → null session (expected)"
elif echo "$AUTH_SESS_RESP" | grep -q '"result"'; then
    pass "auth.session → result present (not signed in, session field may vary)"
else
    fail "auth.session: unexpected response: $AUTH_SESS_RESP"
fi

log "--- sync.status (vault unlocked) ---"
rpc "sync.status"
pass "sync.status"

log "--- vault.lock ---"
rpc "vault.lock"
pass "vault.lock"

log "--- vault.status (locked again) ---"
rpc "vault.status"
pass "vault.status after lock"

log "--- keyring.healthCheck ---"
rpc "keyring.healthCheck"
pass "keyring.healthCheck"

log "--- daemon.shutdown ---"
rpc "daemon.shutdown" || true  # Connection may close before response.
pass "daemon.shutdown"

log ""
log "All smoke tests passed."
