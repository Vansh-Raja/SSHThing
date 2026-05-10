//go:build linux

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
		return Install{}, false
	}
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		canonical = resolved
	}

	// Snap: paths look like /snap/sshthing/<rev>/bin/sshthing.
	if strings.HasPrefix(canonical, "/snap/sshthing/") {
		return Install{
			Kind:      KindCLI,
			Channel:   ChannelUnknown,
			Path:      canonical,
			Updatable: false,
			Detail:    "snap package — update via `snap refresh sshthing`",
		}, true
	}

	// dpkg-managed (Debian/Ubuntu .deb): asking dpkg-query is the only
	// reliable way; the path can be /usr/bin/sshthing which doesn't
	// give it away.
	if hasTool("dpkg-query") {
		if out, err := runCmdOutput(ctx, "dpkg-query", "-W", "-f=${Version}", "sshthing"); err == nil && out != "" {
			return Install{
				Kind:      KindCLI,
				Channel:   ChannelUnknown,
				Path:      canonical,
				Updatable: false,
				Detail:    "dpkg-managed — update via your distro package manager",
				Version:   strings.TrimSpace(out),
			}, true
		}
	}

	// Anything else is a hand-installed binary.
	return Install{
		Kind:      KindCLI,
		Channel:   ChannelStandaloneZip,
		Path:      canonical,
		Updatable: true,
		Detail:    "standalone binary",
	}, true
}

func detectGUI(_ context.Context) (Install, bool) {
	candidates := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Applications"),
			filepath.Join(home, ".local", "bin"),
		)
	}
	candidates = append(candidates,
		"/opt/SSHThing",
		"/opt",
	)

	// Look for a file matching SSHThing*.AppImage in each candidate dir.
	for _, dir := range candidates {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(strings.ToLower(name), "sshthing") {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".appimage") {
				continue
			}
			full := filepath.Join(dir, name)
			return Install{
				Kind:      KindGUI,
				Channel:   ChannelAppImage,
				Path:      full,
				Updatable: true,
				Detail:    "AppImage at " + dir,
			}, true
		}
	}

	return Install{}, false
}
