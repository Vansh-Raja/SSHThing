#!/usr/bin/env bash
# Build the macOS biometric helper as a universal (arm64 + x86_64) binary.
# Output goes to desktop/bin/sshthing-biometric.
#
# Skips silently on non-macOS hosts so the daemon:build script remains portable.

set -euo pipefail

if [[ "$(uname)" != "Darwin" ]]; then
  echo "[biometric] non-macOS host, skipping" >&2
  exit 0
fi

if ! command -v swiftc >/dev/null 2>&1; then
  echo "[biometric] swiftc not found, skipping (Touch ID won't work in this build)" >&2
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="${ROOT}/../mac-helpers/sshthing-biometric/main.swift"
OUT_DIR="${ROOT}/bin"
OUT="${OUT_DIR}/sshthing-biometric"

mkdir -p "${OUT_DIR}"

if [[ ! -f "${SRC}" ]]; then
  echo "[biometric] source not found: ${SRC}" >&2
  exit 1
fi

# Build per-arch then lipo. swiftc doesn't accept multiple -target flags in
# one invocation reliably across versions, so we just compile twice.
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

echo "[biometric] compiling arm64..."
swiftc -O \
  -target arm64-apple-macos11 \
  -framework Foundation -framework Security -framework LocalAuthentication \
  -o "${TMP}/biometric-arm64" \
  "${SRC}"

echo "[biometric] compiling x86_64..."
swiftc -O \
  -target x86_64-apple-macos11 \
  -framework Foundation -framework Security -framework LocalAuthentication \
  -o "${TMP}/biometric-x86_64" \
  "${SRC}" 2>/dev/null || {
    # Some Swift toolchains lack the x86_64 SDK on Apple Silicon. Fall back
    # to arm64-only — the resulting bundle just won't run on Intel Macs.
    echo "[biometric] x86_64 compile failed; emitting arm64-only binary" >&2
    cp "${TMP}/biometric-arm64" "${OUT}"
    chmod +x "${OUT}"
    echo "[biometric] wrote ${OUT} (arm64 only)"
    exit 0
  }

lipo -create -output "${OUT}" "${TMP}/biometric-arm64" "${TMP}/biometric-x86_64"
chmod +x "${OUT}"

# Plain ad-hoc sign — no entitlements. We're using the legacy keychain
# (no SecAccessControl), so we don't need keychain-access-groups.
echo "[biometric] codesigning ad-hoc..."
codesign --force --sign - "${OUT}" 2>&1 || {
  echo "[biometric] WARNING: codesign failed" >&2
}

echo "[biometric] wrote ${OUT} (universal)"
