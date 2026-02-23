//go:build windows

package update

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func fixPathConflicts(_ context.Context, desiredExe string) (PathHealth, error) {
	desiredExe = strings.TrimSpace(desiredExe)
	if desiredExe == "" {
		return PathHealth{Healthy: false, Message: "missing desired install path"}, fmt.Errorf("missing desired executable path")
	}
	desiredDir := filepath.Dir(desiredExe)

	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return PathHealth{}, err
	}
	defer k.Close()

	pathValue, _, _ := k.GetStringValue("Path")
	parts := strings.Split(pathValue, ";")

	newParts := make([]string, 0, len(parts)+1)
	newParts = append(newParts, desiredDir)
	seen := map[string]bool{strings.ToLower(strings.TrimSpace(desiredDir)): true}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		k := strings.ToLower(p)
		if seen[k] {
			continue
		}
		if strings.HasSuffix(strings.ToLower(p), `\sshthing`) || strings.HasSuffix(strings.ToLower(p), `/sshthing`) {
			if !strings.EqualFold(p, desiredDir) {
				continue
			}
		}
		seen[k] = true
		newParts = append(newParts, p)
	}

	newPath := strings.Join(newParts, ";")
	if err := k.SetStringValue("Path", newPath); err != nil {
		return PathHealth{}, err
	}

	return detectWindowsPathHealth(desiredExe)
}
