//go:build windows

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ApplyGUIWindows runs the freshly-downloaded NSIS installer in silent
// mode against the existing install. NSIS handles the in-place upgrade
// (preserving config) when the installer name and product code match the
// previously-installed version, which they will since electron-builder
// produces a stable AppId.
func ApplyGUIWindows(ctx context.Context, asset, checksums AssetInfo, targetExe string) error {
	if asset.URL == "" || checksums.URL == "" {
		return fmt.Errorf("missing GUI installer asset or SHA256SUMS in release")
	}
	if running, _ := guiRunningWindows(); running {
		return fmt.Errorf("SSHThing is running — close it and re-run `sshthing update`")
	}

	tmpDir, err := os.MkdirTemp("", "sshthing-gui-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	installerPath := filepath.Join(tmpDir, asset.Name)
	checksumPath := filepath.Join(tmpDir, checksums.Name)
	if err := downloadToFile(ctx, asset.URL, installerPath); err != nil {
		return fmt.Errorf("download installer: %w", err)
	}
	if err := downloadToFile(ctx, checksums.URL, checksumPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(installerPath, checksumPath, asset.Name); err != nil {
		return fmt.Errorf("checksum verify: %w", err)
	}

	// `/S` is NSIS silent mode; `/D` would override the install dir but we
	// want to upgrade in place, so omit it. `--allusers` mirrors what the
	// initial install did (electron-builder defaults to per-user but
	// honours either flag).
	args := []string{"/S"}
	cmd := exec.CommandContext(ctx, installerPath, args...)
	cmd.SysProcAttr = windowsHiddenSysProcAttr()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("installer exited %v: %s", err, strings.TrimSpace(string(out)))
	}

	// Sanity check: the target executable should still exist after the
	// installer ran.
	if targetExe != "" {
		if _, err := os.Stat(targetExe); err != nil {
			return fmt.Errorf("installer ran but target %s is missing afterwards: %w", targetExe, err)
		}
	}
	return nil
}

func guiRunningWindows() (bool, error) {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq SSHThing.exe", "/NH", "/FO", "CSV")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(string(out)), "sshthing.exe"), nil
}
