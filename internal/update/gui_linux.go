//go:build linux

package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// ApplyGUILinux replaces the user's AppImage with a freshly-downloaded
// copy. AppImages are single self-contained files, so no bundle copy
// dance — just verify-checksum, write to a sibling temp file, then
// rename onto the install path with execute bits preserved.
func ApplyGUILinux(ctx context.Context, asset, checksums AssetInfo, targetPath string) error {
	if asset.URL == "" || checksums.URL == "" {
		return fmt.Errorf("missing AppImage asset or SHA256SUMS in release")
	}
	if running, _ := guiRunningLinux(); running {
		return fmt.Errorf("SSHThing is running — quit it and re-run `sshthing update`")
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(targetPath), ".sshthing-gui-update-")
	if err != nil {
		// Fall back to system temp if the install dir isn't writable —
		// the rename across filesystems will be slower but functional.
		tmpDir, err = os.MkdirTemp("", "sshthing-gui-update-")
		if err != nil {
			return err
		}
	}
	defer os.RemoveAll(tmpDir)

	imgPath := filepath.Join(tmpDir, asset.Name)
	checksumPath := filepath.Join(tmpDir, checksums.Name)
	if err := downloadToFile(ctx, asset.URL, imgPath); err != nil {
		return fmt.Errorf("download AppImage: %w", err)
	}
	if err := downloadToFile(ctx, checksums.URL, checksumPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(imgPath, checksumPath, asset.Name); err != nil {
		return fmt.Errorf("checksum verify: %w", err)
	}
	if err := os.Chmod(imgPath, 0o755); err != nil {
		return fmt.Errorf("chmod new AppImage: %w", err)
	}

	// Try a same-filesystem rename first (atomic). If the temp dir was
	// forced to system /tmp because the install dir is on a different
	// filesystem (rename returns EXDEV), fall back to copy-then-replace.
	if err := os.Rename(imgPath, targetPath); err != nil {
		if copyErr := copyFileLinux(imgPath, targetPath, 0o755); copyErr != nil {
			return fmt.Errorf("install AppImage: rename failed (%v) and copy failed: %w", err, copyErr)
		}
	}
	return nil
}

func guiRunningLinux() (bool, error) {
	if err := exec.Command("pgrep", "-x", "SSHThing").Run(); err == nil {
		return true, nil
	}
	// AppImages on Linux often have lowercase or arch-suffixed exe names;
	// best-effort second pass.
	if err := exec.Command("pgrep", "-f", "SSHThing.*AppImage").Run(); err == nil {
		return true, nil
	}
	return false, nil
}

func copyFileLinux(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(mode)
}
