//go:build windows

package paths

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func dataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "SSHThing"), nil
}

func socketPath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`\\.\pipe\sshthing-%s`, u.Username), nil
}
