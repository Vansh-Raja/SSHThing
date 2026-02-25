package securestore

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

const (
	serviceName          = "sshthing"
	devicePepperUser     = "device-pepper-v1"
	vaultUnlockCacheUser = "session-unlock-cache-v1"
)

// errNotFound is a sentinel returned by the file store when a key is absent.
var errNotFound = errors.New("secret not found in file store")

var (
	keyringUnavailable     bool
	keyringUnavailableOnce sync.Once
)

// isKeyringUnavailableErr returns true for errors that indicate the OS keyring
// daemon is missing or unreachable (as opposed to a simple "not found").
func isKeyringUnavailableErr(err error) bool {
	if err == nil || err == keyring.ErrNotFound {
		return false
	}
	return true
}

func markKeyringUnavailable() {
	keyringUnavailableOnce.Do(func() {
		keyringUnavailable = true
	})
}

// kGet tries the OS keyring, falling back to the file store if unavailable.
func kGet(service, user string) (string, error) {
	if !keyringUnavailable {
		val, err := keyring.Get(service, user)
		if err == nil {
			return val, nil
		}
		if err == keyring.ErrNotFound {
			// Key doesn't exist in OS keyring — also check file store in case
			// it was previously written there during a fallback.
			fs, ferr := getFileStore()
			if ferr == nil {
				if fval, ferr2 := fs.Get(service, user); ferr2 == nil {
					return fval, nil
				}
			}
			return "", keyring.ErrNotFound
		}
		// Keyring daemon unavailable — switch to file store.
		markKeyringUnavailable()
	}

	fs, err := getFileStore()
	if err != nil {
		return "", err
	}
	val, err := fs.Get(service, user)
	if err == errNotFound {
		return "", keyring.ErrNotFound
	}
	return val, err
}

// kSet tries the OS keyring, falling back to the file store if unavailable.
func kSet(service, user, value string) error {
	if !keyringUnavailable {
		err := keyring.Set(service, user, value)
		if err == nil {
			return nil
		}
		markKeyringUnavailable()
	}

	fs, err := getFileStore()
	if err != nil {
		return err
	}
	return fs.Set(service, user, value)
}

// kDelete tries the OS keyring, falling back to the file store if unavailable.
func kDelete(service, user string) error {
	if !keyringUnavailable {
		err := keyring.Delete(service, user)
		if err == nil {
			return nil
		}
		if err == keyring.ErrNotFound {
			// Also try file store.
			fs, ferr := getFileStore()
			if ferr == nil {
				return fs.Delete(service, user)
			}
			return keyring.ErrNotFound
		}
		markKeyringUnavailable()
	}

	fs, err := getFileStore()
	if err != nil {
		return err
	}
	ferr := fs.Delete(service, user)
	if ferr == errNotFound {
		return keyring.ErrNotFound
	}
	return ferr
}

func GetDevicePepper() ([]byte, error) {
	v, err := kGet(serviceName, devicePepperUser)
	if err != nil {
		return nil, err
	}
	b, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(v))
	if err != nil {
		return nil, err
	}
	if len(b) < 16 {
		return nil, fmt.Errorf("device pepper too short")
	}
	return b, nil
}

func GetOrCreateDevicePepper(randReader io.Reader) ([]byte, error) {
	b, err := GetDevicePepper()
	if err == nil {
		return b, nil
	}
	if err != keyring.ErrNotFound {
		return nil, err
	}

	b = make([]byte, 32)
	if _, rerr := io.ReadFull(randReader, b); rerr != nil {
		return nil, rerr
	}
	enc := base64.RawStdEncoding.EncodeToString(b)
	if serr := kSet(serviceName, devicePepperUser, enc); serr != nil {
		return nil, serr
	}
	return b, nil
}

func StoreSessionUnlock(value string) error {
	return kSet(serviceName, vaultUnlockCacheUser, value)
}

func LoadSessionUnlock() (string, error) {
	return kGet(serviceName, vaultUnlockCacheUser)
}

func ClearSessionUnlock() error {
	err := kDelete(serviceName, vaultUnlockCacheUser)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
