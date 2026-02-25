package securestore

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	serviceName          = "sshthing"
	devicePepperUser     = "device-pepper-v1"
	vaultUnlockCacheUser = "session-unlock-cache-v1"
)

func GetDevicePepper() ([]byte, error) {
	v, err := keyring.Get(serviceName, devicePepperUser)
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
	if serr := keyring.Set(serviceName, devicePepperUser, enc); serr != nil {
		return nil, serr
	}
	return b, nil
}

func StoreSessionUnlock(value string) error {
	return keyring.Set(serviceName, vaultUnlockCacheUser, value)
}

func LoadSessionUnlock() (string, error) {
	return keyring.Get(serviceName, vaultUnlockCacheUser)
}

func ClearSessionUnlock() error {
	err := keyring.Delete(serviceName, vaultUnlockCacheUser)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
