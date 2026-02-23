//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func findWindowsInstallExe() (string, string, error) {
	paths := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, p := range paths {
		k, err := registry.OpenKey(p.root, p.path, registry.READ)
		if err != nil {
			continue
		}
		names, _ := k.ReadSubKeyNames(-1)
		_ = k.Close()

		for _, sub := range names {
			sk, err := registry.OpenKey(p.root, p.path+`\`+sub, registry.READ)
			if err != nil {
				continue
			}
			displayName, _, _ := sk.GetStringValue("DisplayName")
			if !strings.Contains(strings.ToLower(displayName), "sshthing") {
				_ = sk.Close()
				continue
			}
			installLoc, _, _ := sk.GetStringValue("InstallLocation")
			_ = sk.Close()

			installLoc = strings.TrimSpace(installLoc)
			if installLoc == "" {
				continue
			}
			exe := filepath.Join(installLoc, "sshthing.exe")
			if _, err := os.Stat(exe); err == nil {
				return exe, displayName, nil
			}
		}
	}

	return "", "", fmt.Errorf("installer install not found")
}

func detectWindowsPathHealth(desiredExe string) (PathHealth, error) {
	ph := PathHealth{Healthy: true}
	resolved := firstWhereHit()
	ph.ResolvedPath = resolved
	ph.DesiredPath = desiredExe

	if strings.TrimSpace(desiredExe) == "" {
		ph.Message = "No desired install path detected"
		return ph, nil
	}

	if strings.EqualFold(strings.TrimSpace(resolved), strings.TrimSpace(desiredExe)) {
		ph.Message = "PATH resolves to installed binary"
		return ph, nil
	}

	ph.Healthy = false
	ph.Message = "PATH resolves to a different sshthing binary"
	if resolved != "" {
		ph.Conflicts = append(ph.Conflicts, resolved)
	}
	return ph, nil
}

func firstWhereHit() string {
	cmd := exec.Command("where", "sshthing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(strings.ToLower(line), "sshthing.exe") {
			return line
		}
	}
	return ""
}
