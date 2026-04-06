//go:build !windows

package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensurePrivateKeyDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

func writePrivateKeyFile(path string, privateKey string) error {
	dir := filepath.Dir(path)
	if err := ensurePrivateKeyDir(dir); err != nil {
		return fmt.Errorf("failed to create private key directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(normalizePrivateKeyForFile(privateKey)); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

func secureDeleteFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err == nil {
		info, statErr := file.Stat()
		if statErr == nil && info != nil {
			zeros := make([]byte, info.Size())
			_, _ = file.Write(zeros)
		}
		_ = file.Close()
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
