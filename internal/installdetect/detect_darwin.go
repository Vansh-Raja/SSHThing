//go:build darwin

package installdetect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// detectPlatform locates the CLI + GUI installs on macOS. The CLI side
// has a priority chain — bundled-inside-GUI is special-cased so we never
// try to "update" the CLI symlink that points into the .app bundle.
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
	// LookPath returns the first match on $PATH; resolve symlinks so the
	// `/usr/local/bin/sshthing → /Applications/SSHThing.app/...` symlink
	// from `installCliSymlink` classifies as "bundled" rather than as a
	// standalone install at /usr/local/bin.
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		canonical = resolved
	}

	// Bundled inside the GUI .app — refuse-class. We still report it so
	// `sshthing update` can show a helpful message instead of silently
	// finding nothing.
	if strings.Contains(canonical, "/SSHThing.app/Contents/") {
		return Install{
			Kind:      KindCLI,
			Channel:   ChannelBundled,
			Path:      canonical,
			Updatable: false,
			Detail:    "bundled inside SSHThing.app — quit the app to update",
		}, true
	}

	// Brew Cellar: /opt/homebrew/Cellar/<formula>/<ver>/bin/sshthing on
	// Apple Silicon, /usr/local/Cellar/... on Intel. Either way the path
	// segment "/Cellar/" is brew-specific.
	if strings.Contains(canonical, "/Cellar/") || strings.Contains(canonical, "/homebrew/") {
		formula := installedBrewFormula(ctx)
		if formula == "" {
			formula = "sshthing"
		}
		return Install{
			Kind:      KindCLI,
			Channel:   ChannelBrew,
			Path:      canonical,
			Updatable: true,
			Detail:    "homebrew formula " + formula,
			Version:   brewFormulaVersion(ctx, formula),
		}, true
	}

	// Anything else is a hand-installed binary — zip extract, manual cp,
	// `go install`, etc. We can do the in-place swap for these.
	return Install{
		Kind:      KindCLI,
		Channel:   ChannelStandaloneZip,
		Path:      canonical,
		Updatable: true,
		Detail:    "standalone binary",
	}, true
}

func detectGUI(ctx context.Context) (Install, bool) {
	// Most-common paths first; mdfind fallback catches users who moved
	// the .app to a non-standard location like ~/Desktop/Apps/.
	candidates := []string{
		"/Applications/SSHThing.app",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Applications", "SSHThing.app"))
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return Install{
				Kind:      KindGUI,
				Channel:   ChannelDMG,
				Path:      p,
				Updatable: true,
				Detail:    "macOS app bundle",
				Version:   readAppBundleVersion(p),
			}, true
		}
	}

	// Fallback: mdfind (Spotlight) by bundle identifier. Only run when
	// the well-known paths missed; mdfind is slow on first cold call.
	if hasTool("mdfind") {
		if out, err := runCmdOutput(ctx, "mdfind", "kMDItemCFBundleIdentifier == 'com.sshthing.desktop'"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				p := strings.TrimSpace(line)
				if p == "" {
					continue
				}
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					return Install{
						Kind:      KindGUI,
						Channel:   ChannelDMG,
						Path:      p,
						Updatable: true,
						Detail:    "macOS app bundle (located via Spotlight)",
						Version:   readAppBundleVersion(p),
					}, true
				}
			}
		}
	}

	return Install{}, false
}

func installedBrewFormula(ctx context.Context) string {
	if !hasTool("brew") {
		return ""
	}
	for _, formula := range []string{"sshthing", "sshthing-beta"} {
		out, err := runCmdOutput(ctx, "brew", "list", "--versions", formula)
		if err != nil || out == "" {
			continue
		}
		return formula
	}
	return ""
}

func brewFormulaVersion(ctx context.Context, formula string) string {
	if formula == "" || !hasTool("brew") {
		return ""
	}
	out, err := runCmdOutput(ctx, "brew", "list", "--versions", formula)
	if err != nil {
		return ""
	}
	// Output: "sshthing 1.2.3" — take the second whitespace-separated field.
	fields := strings.Fields(out)
	if len(fields) >= 2 {
		return fields[1]
	}
	return ""
}

// readAppBundleVersion pulls CFBundleShortVersionString out of the .app's
// Info.plist using `defaults read`. Best-effort: returns "" on any error,
// which is fine because the CLI's update planner falls back on the
// release-feed version comparison.
func readAppBundleVersion(appPath string) string {
	plistPath := filepath.Join(appPath, "Contents", "Info")
	out, err := runCmdOutput(context.Background(), "defaults", "read", plistPath, "CFBundleShortVersionString")
	if err != nil {
		return ""
	}
	return out
}
