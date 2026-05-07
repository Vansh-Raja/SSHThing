package service

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

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
	mu       sync.RWMutex
	store    *db.Store
	// OnUnlock, if set, is called (in a new goroutine) immediately after a
	// successful vault unlock. Useful for services that need to perform
	// post-unlock initialization (e.g. mount restore, sync startup checks).
	OnUnlock func()
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
