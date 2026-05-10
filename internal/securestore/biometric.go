package securestore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Biometric helper service. Wraps the bundled `sshthing-biometric` Swift CLI
// which stores secrets in the macOS keychain protected by Touch ID and
// returns them after a successful biometric prompt.
//
// macOS-only. On other platforms every operation returns ErrBiometricUnavailable.

var (
	// ErrBiometricUnavailable is returned when the platform / build / hardware
	// can't perform biometric auth (no Touch ID enrolled, helper missing, etc.).
	ErrBiometricUnavailable = errors.New("biometric authentication unavailable")
	// ErrBiometricCancelled is returned when the user cancelled the prompt.
	ErrBiometricCancelled = errors.New("biometric prompt cancelled")
	// ErrBiometricAuthFailed is returned when the user failed authentication
	// (wrong fingerprint repeatedly, fallback exhausted).
	ErrBiometricAuthFailed = errors.New("biometric authentication failed")
	// ErrBiometricNotFound is returned when fetch is called but no item exists.
	ErrBiometricNotFound = errors.New("no stored biometric secret")
)

// The keychain "service" / "account" pair that uniquely identifies the
// SSHThing master-password item. Exposed so the daemon can pass them
// consistently between store/fetch/forget.
const (
	biometricService = "com.sshthing.desktop.vault"
	biometricAccount = "master-password"
	biometricReason  = "Unlock your SSHThing vault"
)

// biometricBinary returns the absolute path of the bundled
// `sshthing-biometric` CLI. Resolution order:
//   1. SSHTHING_BIOMETRIC_BIN env var
//   2. sibling of the running daemon executable (production .app bundle)
//   3. PATH lookup
func biometricBinary() (string, error) {
	if env := strings.TrimSpace(os.Getenv("SSHTHING_BIOMETRIC_BIN")); env != "" {
		return env, nil
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "sshthing-biometric")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath("sshthing-biometric"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("sshthing-biometric helper not found: %w", ErrBiometricUnavailable)
}

// BiometricAvailable returns true if Touch ID is supported and at least one
// fingerprint is enrolled. False otherwise — any specific failure reason is
// suppressed to keep this a cheap "should we offer the feature?" check.
func BiometricAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	bin, err := biometricBinary()
	if err != nil {
		return false
	}
	cmd := exec.Command(bin, "available")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// BiometricStore writes secret to the keychain under the SSHThing biometric
// item, replacing any existing value. The keychain ACL requires Touch ID to
// read it back. Idempotent.
func BiometricStore(secret string) error {
	if runtime.GOOS != "darwin" {
		return ErrBiometricUnavailable
	}
	if secret == "" {
		return errors.New("refusing to store empty secret")
	}
	bin, err := biometricBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "store", "--service", biometricService, "--account", biometricAccount)
	cmd.Stdin = strings.NewReader(secret)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("biometric store: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// BiometricFetch triggers the macOS Touch ID prompt and returns the stored
// secret on success. Maps non-zero exit codes to typed errors so callers can
// branch on cancellation vs auth-failure vs missing-item.
func BiometricFetch() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", ErrBiometricUnavailable
	}
	bin, err := biometricBinary()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(bin, "fetch",
		"--service", biometricService,
		"--account", biometricAccount,
		"--reason", biometricReason,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}

	// Map exit codes from main.swift:
	//   1  unavailable / cancelled
	//   2  auth failed
	//   3  not found
	//   4  other I/O error
	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	switch exitCode {
	case 1:
		return "", ErrBiometricCancelled
	case 2:
		return "", ErrBiometricAuthFailed
	case 3:
		return "", ErrBiometricNotFound
	default:
		return "", fmt.Errorf("biometric fetch failed (exit %d): %s", exitCode, strings.TrimSpace(stderr.String()))
	}
}

// BiometricForget removes the keychain item. Idempotent — succeeds even if
// no item exists.
func BiometricForget() error {
	if runtime.GOOS != "darwin" {
		return nil // no-op on other platforms; nothing to forget
	}
	bin, err := biometricBinary()
	if err != nil {
		return nil // helper missing — nothing to forget
	}
	cmd := exec.Command(bin, "forget",
		"--service", biometricService,
		"--account", biometricAccount,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("biometric forget: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
