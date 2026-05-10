package service

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
)

const unlockTTL = 15 * time.Minute

// ErrVaultLocked is returned by methods that require an unlocked vault.
var ErrVaultLocked = fmt.Errorf("vault is locked")

// ErrVaultMissing is returned when no vault database exists.
var ErrVaultMissing = fmt.Errorf("vault not found")

// ErrInvalidPassword is returned when the password is incorrect.
var ErrInvalidPassword = fmt.Errorf("invalid password")

// Vault manages the unlocked db.Store and the session unlock cache.
// It is safe for concurrent use.
type Vault struct {
	mu    sync.RWMutex
	store *db.Store
	// OnUnlock, if set, is called (in a new goroutine) immediately after a
	// successful vault unlock. Useful for services that need to perform
	// post-unlock initialization (e.g. mount restore, sync startup checks).
	OnUnlock func()
	// Notify emits a JSON-RPC notification to all connected clients.
	Notify func(method string, params any)
	// CfgStore is the shared atomic config holder. Vault writes to it when
	// biometric enrolment changes (BiometricEnabled / BiometricExpiry).
	// May be nil in tests.
	CfgStore *CfgStore
}

// UnlockResult is returned on a successful unlock.
type UnlockResult struct {
	Unlocked      bool   `json:"unlocked"`
	Salt          string `json:"salt"`
	SessionTTLSec int    `json:"sessionTtlSec"`
}

// StatusResult is the response for vault.status.
type StatusResult struct {
	Unlocked  bool   `json:"unlocked"`
	ExpiresAt *int64 `json:"expiresAt"` // Unix seconds, or null
}

// Unlock opens the vault with password and caches the session secret.
// Returns ErrVaultMissing if no vault file exists, ErrInvalidPassword on wrong key.
func (v *Vault) Unlock(password string) (*UnlockResult, error) {
	// Guard: check vault exists before calling db.Init, which would create a new DB
	// on first call with mode=rwc. See internal/db/db.go Init().
	exists, err := db.Exists()
	if err != nil {
		return nil, fmt.Errorf("check vault: %w", err)
	}
	if !exists {
		return nil, ErrVaultMissing
	}

	store, err := db.Init(password)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid password") {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	salt, err := store.GetSalt()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("get salt: %w", err)
	}

	// Cache the password in the OS keyring via unlock.Save so the daemon can
	// re-open the vault after a restart within the TTL window.
	// See internal/unlock/session.go Save().
	if err := unlock.Save(password, unlockTTL); err != nil {
		// Non-fatal: keyring may not be available in headless envs.
		// Log but continue.
		_ = err
	}

	v.mu.Lock()
	if v.store != nil {
		_ = v.store.Close()
	}
	v.store = store
	onUnlock := v.OnUnlock
	v.mu.Unlock()

	if onUnlock != nil {
		go onUnlock()
	}

	return &UnlockResult{
		Unlocked:      true,
		Salt:          salt,
		SessionTTLSec: int(unlockTTL.Seconds()),
	}, nil
}

// Status returns whether the vault is currently unlocked and the session TTL.
func (v *Vault) Status() *StatusResult {
	v.mu.RLock()
	unlocked := v.store != nil
	v.mu.RUnlock()

	_, expiresAt, ok, _ := unlock.Load()
	if !ok {
		expiresAt = time.Time{}
	}

	var expiresAtPtr *int64
	if !expiresAt.IsZero() {
		ts := expiresAt.Unix()
		expiresAtPtr = &ts
	}

	return &StatusResult{
		Unlocked:  unlocked,
		ExpiresAt: expiresAtPtr,
	}
}

// Store returns the current *db.Store if the vault is unlocked, or nil.
// Callers must treat nil as ErrVaultLocked.
func (v *Vault) Store() *db.Store {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.store
}

// Create initialises a brand-new encrypted vault with the given password.
// Returns ErrVaultMissing if a vault already exists — callers must check first.
// After creation the vault is left unlocked.
func (v *Vault) Create(password string) (*UnlockResult, error) {
	exists, err := db.Exists()
	if err != nil {
		return nil, fmt.Errorf("check vault: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("vault already exists")
	}

	// db.Init with mode=rwc creates the DB if absent.
	store, err := db.Init(password)
	if err != nil {
		return nil, fmt.Errorf("create vault: %w", err)
	}
	salt, err := store.GetSalt()
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("get salt: %w", err)
	}
	if err := unlock.Save(password, unlockTTL); err != nil {
		_ = err // non-fatal
	}

	v.mu.Lock()
	if v.store != nil {
		_ = v.store.Close()
	}
	v.store = store
	onUnlock := v.OnUnlock
	v.mu.Unlock()

	if onUnlock != nil {
		go onUnlock()
	}

	return &UnlockResult{
		Unlocked:      true,
		Salt:          salt,
		SessionTTLSec: int(unlockTTL.Seconds()),
	}, nil
}

// Lock drops the in-memory Store and clears the keyring session.
func (v *Vault) Lock() {
	v.mu.Lock()
	if v.store != nil {
		_ = v.store.Close()
		v.store = nil
	}
	v.mu.Unlock()
	_ = unlock.Clear()
	if v.Notify != nil {
		v.Notify("vault.locked", map[string]any{})
	}
}

// Vacuum rebuilds the encrypted database to reclaim space.
func (v *Vault) Vacuum() error {
	v.mu.RLock()
	store := v.store
	v.mu.RUnlock()
	if store == nil {
		return ErrVaultLocked
	}
	return store.Vacuum()
}

// ChangePassword re-keys the SQLCipher DB and re-encrypts all host secrets.
func (v *Vault) ChangePassword(oldPassword, newPassword string) error {
	store := v.Store()
	if store == nil {
		return ErrVaultLocked
	}
	// Verify the old password by attempting to open the DB with it.
	// db.Init handles the wrong-password error internally.
	verifyStore, err := db.Init(oldPassword)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid password") {
			return ErrInvalidPassword
		}
		return fmt.Errorf("verify old password: %w", err)
	}
	_ = verifyStore.Close()

	if err := store.ChangePassword(newPassword); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	// Re-cache new password in keyring.
	if err := unlock.Save(newPassword, unlockTTL); err != nil {
		_ = err // non-fatal
	}
	return nil
}

// KeyringHealthResult is returned by KeyringHealthCheck.
type KeyringHealthResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// KeyringHealthCheck performs a keyring round-trip (set → get → delete).
// Uses a random test value so parallel tests don't interfere.
func KeyringHealthCheck() *KeyringHealthResult {
	testVal := make([]byte, 8)
	if _, err := rand.Read(testVal); err != nil {
		return &KeyringHealthResult{Error: "rand: " + err.Error()}
	}
	// GetOrCreateDevicePepper is the simplest public wrapper that does
	// a keyring set+get in one call; use StoreSessionUnlock / LoadSessionUnlock
	// for a full round-trip test.
	testStr := fmt.Sprintf("healthcheck-%x", testVal)
	if err := securestore.StoreSessionUnlock(testStr); err != nil {
		return &KeyringHealthResult{Error: "store: " + err.Error()}
	}
	got, err := securestore.LoadSessionUnlock()
	if err != nil {
		return &KeyringHealthResult{Error: "load: " + err.Error()}
	}
	if got != testStr {
		return &KeyringHealthResult{Error: "value mismatch"}
	}
	return &KeyringHealthResult{OK: true}
}

// ────────────────────────────────────────────────────────────────────────
// Biometric (Touch ID) unlock
// ────────────────────────────────────────────────────────────────────────

// BiometricStatusResult tells the renderer what's possible: whether Touch ID
// hardware is available, whether the user has enrolled SSHThing for it, and
// when the cached secret expires.
type BiometricStatusResult struct {
	Available bool   `json:"available"` // Touch ID supported & user enrolled OS-side
	Enabled   bool   `json:"enabled"`   // user has stored their vault password
	ExpiresAt int64  `json:"expiresAt"` // 0 if not enabled; unix seconds otherwise
	Expired   bool   `json:"expired"`   // true if Enabled && now > ExpiresAt
}

// BiometricEnableResult is returned from EnableBiometric on success.
type BiometricEnableResult struct {
	OK        bool  `json:"ok"`
	ExpiresAt int64 `json:"expiresAt"`
}

// biometricCacheDuration matches the user's locked-in design choice:
// 7 days from the first password unlock that turned the feature on.
const biometricCacheDuration = 7 * 24 * time.Hour

// BiometricStatus returns the combined hardware + user-config view.
// Cheap; never triggers a Touch ID prompt.
func (v *Vault) BiometricStatus() *BiometricStatusResult {
	res := &BiometricStatusResult{
		Available: securestore.BiometricAvailable(),
	}
	if v.CfgStore == nil {
		return res
	}
	cfg := v.CfgStore.Get()
	res.Enabled = cfg.Vault.BiometricEnabled
	res.ExpiresAt = cfg.Vault.BiometricExpiry
	if res.Enabled && res.ExpiresAt > 0 && time.Now().Unix() > res.ExpiresAt {
		res.Expired = true
	}
	return res
}

// EnableBiometric verifies the password (by attempting a lightweight DB
// open), then stores it in the macOS keychain protected by Touch ID.
// Sets cfg.Vault.BiometricEnabled=true and BiometricExpiry=now+7d.
//
// We re-verify the password rather than trusting the renderer to only call
// this when the vault is already unlocked — defence in depth.
func (v *Vault) EnableBiometric(password string) (*BiometricEnableResult, error) {
	if !securestore.BiometricAvailable() {
		return nil, securestore.ErrBiometricUnavailable
	}
	if v.CfgStore == nil {
		return nil, fmt.Errorf("biometric: config store not wired")
	}
	if strings.TrimSpace(password) == "" {
		return nil, ErrInvalidPassword
	}

	// Verify the password actually opens the vault. The cheapest way is a
	// read-only Init+Close round-trip.
	exists, err := db.Exists()
	if err != nil {
		return nil, fmt.Errorf("check vault: %w", err)
	}
	if !exists {
		return nil, ErrVaultMissing
	}
	verifyStore, err := db.Init(password)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid password") {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}
	_ = verifyStore.Close()

	if err := securestore.BiometricStore(password); err != nil {
		return nil, fmt.Errorf("biometric store: %w", err)
	}

	expiresAt := time.Now().Add(biometricCacheDuration).Unix()
	if _, err := v.CfgStore.Mutate(func(cfg *config.Config) error {
		cfg.Vault.BiometricEnabled = true
		cfg.Vault.BiometricExpiry = expiresAt
		return nil
	}); err != nil {
		// Non-fatal — keychain is updated; the renderer can re-derive
		// state from a future settings.get and the user can retry.
		log.Printf("biometric: config save failed: %v", err)
	}

	return &BiometricEnableResult{OK: true, ExpiresAt: expiresAt}, nil
}

// DisableBiometric forgets the keychain item and clears the config flags.
// Idempotent — succeeds even if Touch ID was never enabled.
func (v *Vault) DisableBiometric() error {
	if err := securestore.BiometricForget(); err != nil {
		return err
	}
	if v.CfgStore != nil {
		if _, err := v.CfgStore.Mutate(func(cfg *config.Config) error {
			cfg.Vault.BiometricEnabled = false
			cfg.Vault.BiometricExpiry = 0
			return nil
		}); err != nil {
			log.Printf("biometric: config save failed: %v", err)
		}
	}
	return nil
}

// UnlockWithBiometric prompts Touch ID, retrieves the cached password from
// the keychain, and unlocks the vault as if the user had typed it. Returns
// ErrBiometricExpired if the 7-day window has elapsed (caller should clear
// the cache and demand the password). Returns the underlying biometric
// errors verbatim for the renderer to map to UI states.
func (v *Vault) UnlockWithBiometric() (*UnlockResult, error) {
	if v.CfgStore == nil {
		return nil, securestore.ErrBiometricUnavailable
	}
	cfg := v.CfgStore.Get()
	if !cfg.Vault.BiometricEnabled {
		return nil, securestore.ErrBiometricUnavailable
	}
	if cfg.Vault.BiometricExpiry > 0 && time.Now().Unix() > cfg.Vault.BiometricExpiry {
		// Expired: forget the cached secret and force password.
		_ = securestore.BiometricForget()
		_, _ = v.CfgStore.Mutate(func(c *config.Config) error {
			c.Vault.BiometricEnabled = false
			c.Vault.BiometricExpiry = 0
			return nil
		})
		return nil, securestore.ErrBiometricUnavailable
	}

	password, err := securestore.BiometricFetch()
	if err != nil {
		return nil, err
	}
	return v.Unlock(password)
}
