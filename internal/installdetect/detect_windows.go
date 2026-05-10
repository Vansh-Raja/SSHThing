//go:build windows

package installdetect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func detectPlatform(ctx context.Context) []Install {
	var out []Install
	if cli, ok := detectCLI(ctx); ok {
		out = append(out, cli)
	}
	if gui, ok := detectGUI(ctx); ok {
		out = append(out, gui)
	}
	return out
}

func detectCLI(ctx context.Context) (Install, bool) {
	resolved, err := exec.LookPath("sshthing")
	if err != nil {
		// Fall back to sshthing.exe explicitly in case PATHEXT is weird.
		resolved, err = exec.LookPath("sshthing.exe")
		if err != nil {
			return Install{}, false
		}
	}
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		canonical = resolved
	}

	if winget := detectWingetCLI(ctx); winget {
		return Install{
			Kind:      KindCLI,
			Channel:   ChannelWinget,
			Path:      canonical,
			Updatable: true,
			Detail:    "winget-managed (Vansh-Raja.SSHThing)",
		}, true
	}
	if choco := detectChocoCLI(ctx); choco {
		return Install{
			Kind:      KindCLI,
			Channel:   ChannelChoco,
			Path:      canonical,
			Updatable: true,
			Detail:    "choco-managed (sshthing)",
		}, true
	}

	return Install{
		Kind:      KindCLI,
		Channel:   ChannelStandaloneZip,
		Path:      canonical,
		Updatable: true,
		Detail:    "standalone .exe",
	}, true
}

func detectWingetCLI(ctx context.Context) bool {
	if !hasTool("winget") {
		return false
	}
	out, err := runCmdOutput(ctx, "winget", "list", "sshthing", "--exact", "--accept-source-agreements")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(out), "sshthing")
}

func detectChocoCLI(ctx context.Context) bool {
	if !hasTool("choco") {
		return false
	}
	out, err := runCmdOutput(ctx, "choco", "list", "--local-only", "sshthing", "--limit-output")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(out), "sshthing")
}

func detectGUI(_ context.Context) (Install, bool) {
	// Standard NSIS install paths. The installer writes to one of these
	// per-user vs per-machine; both are worth probing.
	candidates := []string{}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "SSHThing", "SSHThing.exe"))
	}
	if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "SSHThing", "SSHThing.exe"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "sshthing", "SSHThing.exe"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return Install{
				Kind:      KindGUI,
				Channel:   ChannelNSIS,
				Path:      p,
				Updatable: true,
				Detail:    "NSIS install",
			}, true
		}
	}
	return Install{}, false
}
