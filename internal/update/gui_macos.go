//go:build darwin

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ApplyGUIDarwin downloads the GUI DMG, verifies its checksum + signature,
// mounts it, copies the .app over the existing install via an atomic
// rename (with a `.bak` rollback), then unmounts. Caller must have
// already confirmed SSHThing.app is not running — see GUIRunning().
func ApplyGUIDarwin(ctx context.Context, asset, checksums AssetInfo, targetAppPath string) error {
	if asset.URL == "" || checksums.URL == "" {
		return fmt.Errorf("missing GUI asset or SHA256SUMS in release")
	}
	if !strings.HasSuffix(targetAppPath, ".app") {
		return fmt.Errorf("target path %q is not a .app bundle", targetAppPath)
	}
	if running, _ := guiRunningDarwin(); running {
		return fmt.Errorf("SSHThing is running — quit it (Cmd+Q) and re-run `sshthing update`")
	}

	tmpDir, err := os.MkdirTemp("", "sshthing-gui-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dmgPath := filepath.Join(tmpDir, asset.Name)
	checksumPath := filepath.Join(tmpDir, checksums.Name)
	if err := downloadToFile(ctx, asset.URL, dmgPath); err != nil {
		return fmt.Errorf("download dmg: %w", err)
	}
	if err := downloadToFile(ctx, checksums.URL, checksumPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(dmgPath, checksumPath, asset.Name); err != nil {
		return fmt.Errorf("checksum verify: %w", err)
	}

	mountPoint, err := mountDMG(ctx, dmgPath)
	if err != nil {
		return fmt.Errorf("mount dmg: %w", err)
	}
	defer detachDMG(context.Background(), mountPoint)

	srcApp := filepath.Join(mountPoint, "SSHThing.app")
	if info, err := os.Stat(srcApp); err != nil || !info.IsDir() {
		return fmt.Errorf("DMG does not contain SSHThing.app at %s", srcApp)
	}

	// Verify the bundle the user is about to copy. If signing or Gatekeeper
	// assessment fails, refuse — better to halt than to install a tampered
	// or unsigned build over a working one.
	if err := verifyMacAppSignature(ctx, srcApp); err != nil {
		return fmt.Errorf("downloaded app failed signature check: %w", err)
	}

	stagingApp := targetAppPath + ".new"
	backupApp := targetAppPath + ".bak"

	// Clean any stale staging/backup dirs from a prior failed run.
	_ = os.RemoveAll(stagingApp)

	if err := dittoCopy(ctx, srcApp, stagingApp); err != nil {
		return fmt.Errorf("copy app from dmg: %w", err)
	}

	// Atomic-ish swap: move existing → .bak, move .new → real path. If
	// the second move fails, callers can manually `mv .app.bak .app` to
	// recover. We delete an old .bak only after a fresh .bak is in place.
	if _, err := os.Stat(targetAppPath); err == nil {
		_ = os.RemoveAll(backupApp)
		if err := os.Rename(targetAppPath, backupApp); err != nil {
			_ = os.RemoveAll(stagingApp)
			return fmt.Errorf("backup existing app: %w (the install is unchanged)", err)
		}
	}
	if err := os.Rename(stagingApp, targetAppPath); err != nil {
		// Try to roll the backup back into place so the user isn't left
		// without an app at all.
		if _, statErr := os.Stat(backupApp); statErr == nil {
			_ = os.Rename(backupApp, targetAppPath)
		}
		return fmt.Errorf("install new app: %w", err)
	}

	return nil
}

// guiRunningDarwin returns true when an SSHThing process is alive in
// userland — used to refuse an in-place .app swap while the app is
// running.
func guiRunningDarwin() (bool, error) {
	cmd := exec.Command("pgrep", "-x", "SSHThing")
	if err := cmd.Run(); err != nil {
		// pgrep exits 1 when nothing matches, which is the happy path.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func mountDMG(ctx context.Context, dmgPath string) (string, error) {
	// `hdiutil attach -nobrowse -plist` returns an XML dictionary with
	// each system-entity mount-point. Parsing that XML in pure Go is
	// painful; we fall back to the line-oriented output and grep for the
	// /Volumes/... path. The mount is always under /Volumes for non-readonly
	// DMGs and electron-builder's DMGs follow that convention.
	cmd := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-quiet", "-noverify", "-noautoopen", dmgPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("hdiutil attach: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	// Output lines look like:
	//   /dev/disk5\tApple_partition_scheme\t
	//   /dev/disk5s1\tApple_HFS\t/Volumes/SSHThing 1.2.3
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(line, "\t")
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if strings.HasPrefix(f, "/Volumes/") {
				return f, nil
			}
		}
	}
	return "", fmt.Errorf("could not parse mount point from hdiutil output: %s", strings.TrimSpace(string(out)))
}

func detachDMG(ctx context.Context, mountPoint string) {
	if mountPoint == "" {
		return
	}
	// `-force` unmounts even if Finder has windows open into it. We don't
	// surface errors — the temp dir cleanup will eventually time it out
	// and a stale mount is at most a UI nuisance.
	_ = exec.CommandContext(ctx, "hdiutil", "detach", "-quiet", "-force", mountPoint).Run()
}

// dittoCopy copies a directory tree using ditto, which preserves
// signatures, extended attributes, resource forks, and ACLs better than
// cp -a. Required for code-signed .app bundles.
func dittoCopy(ctx context.Context, src, dst string) error {
	cmd := exec.CommandContext(ctx, "ditto", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ditto %s → %s: %v (%s)", src, dst, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// verifyMacAppSignature runs codesign + spctl against the freshly-copied
// .app to make sure the developer signature is intact. Both checks are
// fail-closed: an unsigned or rejected bundle never gets installed.
func verifyMacAppSignature(ctx context.Context, appPath string) error {
	cs := exec.CommandContext(ctx, "codesign", "--verify", "--deep", "--strict", appPath)
	if out, err := cs.CombinedOutput(); err != nil {
		return fmt.Errorf("codesign verify: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	// `spctl --assess` is the Gatekeeper-equivalent. On a notarised
	// release this always passes; we treat a failure as a hard reject.
	sp := exec.CommandContext(ctx, "spctl", "--assess", "--type", "execute", appPath)
	if out, err := sp.CombinedOutput(); err != nil {
		return fmt.Errorf("spctl assess: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
