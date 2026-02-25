package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Apply(ctx context.Context, check CheckResult, currentExe string) (ApplyResult, error) {
	if !check.UpdateAvailable {
		return ApplyResult{Success: false, Message: "already up to date"}, nil
	}

	switch check.ApplyMode {
	case ApplyModeCommand:
		return applyCommandMode(ctx, check, currentExe)
	case ApplyModeInstaller:
		return applyInstallerMode(ctx, check, currentExe)
	case ApplyModeReplaceBin:
		return applyReplaceMode(ctx, check, currentExe)
	case ApplyModeGuidance:
		return ApplyResult{Success: false, Message: strings.Join(check.Guidance, " ")}, nil
	default:
		return ApplyResult{Success: false, Message: "no update apply mode available"}, nil
	}
}

func applyCommandMode(ctx context.Context, check CheckResult, currentExe string) (ApplyResult, error) {
	if len(check.ApplyCommand) == 0 {
		return ApplyResult{}, fmt.Errorf("missing apply command")
	}

	out, err := runCommand(ctx, check.ApplyCommand)
	lowerOut := strings.ToLower(out)
	if err == nil && runtime.GOOS == "windows" {
		if strings.Contains(lowerOut, "no installed package found") ||
			strings.Contains(lowerOut, "no package found") ||
			strings.Contains(lowerOut, "no applicable update found") {
			err = fmt.Errorf("delegated package update not applicable")
		}
	}
	if err != nil {
		if runtime.GOOS == "windows" {
			if strings.Contains(lowerOut, "no installed package found") || strings.Contains(lowerOut, "no package found") || strings.Contains(lowerOut, "no applicable update found") {
				if hasTool("choco") {
					fallback := []string{"choco", "upgrade", "sshthing", "-y"}
					fout, ferr := runCommand(ctx, fallback)
					if ferr == nil {
						return ApplyResult{Success: true, Message: strings.TrimSpace(fout), NeedsRelaunch: true, RelaunchPath: currentExe}, nil
					}
				}
			}
			if check.Asset.URL != "" {
				installerResult, instErr := applyInstallerMode(ctx, check, currentExe)
				if instErr == nil {
					return installerResult, nil
				}
			}
		}
		return ApplyResult{}, fmt.Errorf("update command failed: %w (%s)", err, strings.TrimSpace(out))
	}

	return ApplyResult{Success: true, Message: strings.TrimSpace(out), NeedsRelaunch: true, RelaunchPath: currentExe}, nil
}

func applyInstallerMode(ctx context.Context, check CheckResult, currentExe string) (ApplyResult, error) {
	if check.Asset.URL == "" || check.Checksums.URL == "" {
		return ApplyResult{}, fmt.Errorf("missing installer/checksum asset")
	}
	tmpDir, err := os.MkdirTemp("", "sshthing-update-")
	if err != nil {
		return ApplyResult{}, err
	}

	installerPath := filepath.Join(tmpDir, check.Asset.Name)
	checksumsPath := filepath.Join(tmpDir, check.Checksums.Name)
	if err := downloadToFile(ctx, check.Asset.URL, installerPath); err != nil {
		return ApplyResult{}, err
	}
	if err := downloadToFile(ctx, check.Checksums.URL, checksumsPath); err != nil {
		return ApplyResult{}, err
	}
	if err := verifyChecksum(installerPath, checksumsPath, check.Asset.Name); err != nil {
		return ApplyResult{}, err
	}

	relaunchPath := check.InstallerExe
	if strings.TrimSpace(relaunchPath) == "" {
		relaunchPath = currentExe
	}

	action := &HandoffAction{
		Type:          HandoffInstaller,
		ParentPID:     os.Getpid(),
		InstallerPath: installerPath,
		InstallerArgs: []string{"/SP-", "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART"},
		RelaunchPath:  relaunchPath,
		CleanupDir:    tmpDir,
	}
	return ApplyResult{Success: true, Message: "installer staged", NeedsRelaunch: true, Handoff: action}, nil
}

func applyReplaceMode(ctx context.Context, check CheckResult, currentExe string) (ApplyResult, error) {
	if check.Asset.URL == "" || check.Checksums.URL == "" {
		return ApplyResult{}, fmt.Errorf("missing archive/checksum asset")
	}
	tmpDir, err := os.MkdirTemp("", "sshthing-update-")
	if err != nil {
		return ApplyResult{}, err
	}

	archivePath := filepath.Join(tmpDir, check.Asset.Name)
	checksumsPath := filepath.Join(tmpDir, check.Checksums.Name)
	if err := downloadToFile(ctx, check.Asset.URL, archivePath); err != nil {
		return ApplyResult{}, err
	}
	if err := downloadToFile(ctx, check.Checksums.URL, checksumsPath); err != nil {
		return ApplyResult{}, err
	}
	if err := verifyChecksum(archivePath, checksumsPath, check.Asset.Name); err != nil {
		return ApplyResult{}, err
	}

	binaryName := "sshthing"
	if runtime.GOOS == "windows" {
		binaryName = "sshthing.exe"
	}
	newBinaryPath := filepath.Join(tmpDir, binaryName)
	if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		err = extractBinaryFromTarGz(archivePath, binaryName, newBinaryPath)
	} else {
		err = extractBinaryFromZip(archivePath, binaryName, newBinaryPath)
	}
	if err != nil {
		return ApplyResult{}, err
	}

	action := &HandoffAction{
		Type:             HandoffReplace,
		ParentPID:        os.Getpid(),
		NewBinaryPath:    newBinaryPath,
		TargetBinaryPath: currentExe,
		RelaunchPath:     currentExe,
		CleanupDir:       tmpDir,
	}
	return ApplyResult{Success: true, Message: "binary replacement staged", NeedsRelaunch: true, Handoff: action}, nil
}

func runCommand(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
