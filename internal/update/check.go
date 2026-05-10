package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Check fetches the latest release for the given channel and returns a
// CheckResult describing whether an update is available + how to apply it.
//
// As of the v10 config schema this function is *stateless* with respect to
// `cfg` — it neither reads ETags from it nor writes them back. Callers (the
// daemon's once-a-week nudge, the `sshthing update` CLI) decide separately
// whether to persist `LastCheckedAt` / `LastSeenVersion` afterwards. The
// previous ETag-cache optimisation existed to keep the auto-apply check
// loop cheap; with auto-apply removed the overall call rate is far lower
// (one CLI invocation, or one ping per week), so dropping the cache is fine
// and removes a class of subtle bugs around per-channel cache desync.
func Check(ctx context.Context, currentVersion string, channel ReleaseChannel) (CheckResult, error) {
	if channel != ReleaseChannelBeta {
		channel = ReleaseChannelStable
	}

	result := CheckResult{
		CheckedAt:      time.Now(),
		CurrentVersion: normalizeVersionString(currentVersion),
		ReleaseChannel: channel,
		Channel:        ChannelUnknown,
		ApplyMode:      ApplyModeNone,
	}

	chKind, detail, installerExe, err := detectChannel(ctx)
	if err != nil {
		result.Channel = ChannelUnknown
		result.ChannelDetail = err.Error()
	} else {
		result.Channel = chKind
		result.ChannelDetail = detail
		result.InstallerExe = installerExe
	}

	pathHealth, _ := detectPathHealth(ctx, installerExe)
	if strings.TrimSpace(installerExe) == "" {
		if exe, err := os.Executable(); err == nil {
			pathHealth, _ = detectPathHealth(ctx, exe)
		}
	}
	result.PathHealth = pathHealth

	rel, _, _, err := fetchReleaseForChannel(ctx, channel, "")
	if err != nil {
		return result, err
	}

	result.LatestTag = rel.TagName
	result.LatestVersion = normalizeVersionString(rel.TagName)
	result.ReleaseURL = rel.HTMLURL
	result.Checksums = findAsset(rel.Assets, "SHA256SUMS")
	result.Asset = resolveReleaseAsset(rel.Assets)
	result.AllAssets = make([]AssetInfo, 0, len(rel.Assets))
	for _, a := range rel.Assets {
		result.AllAssets = append(result.AllAssets, AssetInfo{Name: a.Name, URL: a.URL})
	}

	if strings.EqualFold(result.CurrentVersion, "dev") || result.CurrentVersion == "" {
		result.UpdateAvailable = strings.TrimSpace(result.LatestVersion) != ""
	} else {
		result.UpdateAvailable = compareVersions(result.CurrentVersion, result.LatestVersion) < 0
	}

	selectApplyMode(&result)
	return result, nil
}

func resolveReleaseAsset(assets []githubReleaseAsset) AssetInfo {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "windows" {
		return findAsset(assets, "sshthing-setup-windows-amd64.exe")
	}
	if goos == "darwin" {
		if goarch == "arm64" {
			return findAsset(assets, "sshthing-macos-arm64.zip")
		}
		return findAsset(assets, "sshthing-macos-amd64.zip")
	}
	if goos == "linux" {
		if goarch == "arm64" {
			return findAsset(assets, "sshthing-linux-arm64.tar.gz")
		}
		return findAsset(assets, "sshthing-linux-amd64.tar.gz")
	}
	return AssetInfo{}
}

func selectApplyMode(result *CheckResult) {
	if result == nil {
		return
	}
	if !result.UpdateAvailable {
		result.ApplyMode = ApplyModeNone
		return
	}

	switch runtime.GOOS {
	case "windows":
		if result.ReleaseChannel == ReleaseChannelBeta && result.Channel != ChannelWindowsInstaller && (hasTool("winget") || hasTool("choco")) {
			result.ApplyMode = ApplyModeGuidance
			result.Guidance = []string{"Beta updates require a standalone install. Download the prerelease asset from GitHub Releases."}
			return
		}
		if hasTool("winget") {
			result.ApplyMode = ApplyModeCommand
			result.ApplyCommand = []string{"winget", "upgrade", "--name", "sshthing", "--silent", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
			return
		}
		if hasTool("choco") {
			result.ApplyMode = ApplyModeCommand
			result.ApplyCommand = []string{"choco", "upgrade", "sshthing", "-y"}
			return
		}
		if result.Asset.URL != "" {
			result.ApplyMode = ApplyModeInstaller
			return
		}
		result.ApplyMode = ApplyModeGuidance
		result.Guidance = []string{"Download latest Windows installer from GitHub Releases and run it."}
	case "darwin":
		if isBrewManaged() {
			if result.ReleaseChannel == ReleaseChannelBeta {
				result.ApplyMode = ApplyModeGuidance
				result.Guidance = []string{"Beta updates require a standalone install. Download the prerelease asset from GitHub Releases."}
				return
			}
			result.ApplyMode = ApplyModeCommand
			result.ApplyCommand = []string{"brew", "upgrade", "sshthing"}
			return
		}
		if result.Asset.URL != "" {
			result.ApplyMode = ApplyModeReplaceBin
			return
		}
		result.ApplyMode = ApplyModeGuidance
		result.Guidance = []string{"Download latest macOS zip from GitHub Releases and replace your binary."}
	case "linux":
		if result.Asset.URL != "" {
			result.ApplyMode = ApplyModeReplaceBin
		} else {
			result.ApplyMode = ApplyModeGuidance
			result.Guidance = []string{"No Linux binary found in this release. Rebuild from source."}
		}
	default:
		result.ApplyMode = ApplyModeGuidance
		result.Guidance = []string{"Auto-update is not available on this platform. Rebuild from source."}
	}
}

func detectChannel(ctx context.Context) (Channel, string, string, error) {
	_ = ctx
	switch runtime.GOOS {
	case "windows":
		installerExe, display, err := findWindowsInstallExe()
		if err == nil && installerExe != "" {
			return ChannelWindowsInstaller, display, installerExe, nil
		}
		if hasTool("winget") || hasTool("choco") {
			return ChannelStandalone, "windows package-manager fallback available", "", nil
		}
		return ChannelStandalone, "windows standalone", "", nil
	case "darwin":
		if formula := installedBrewFormula(); formula != "" {
			return ChannelMacOSBrew, fmt.Sprintf("homebrew-managed install (%s)", formula), "", nil
		}
		return ChannelStandalone, "macOS standalone binary", "", nil
	case "linux":
		return ChannelStandalone, "linux standalone", "", nil
	default:
		return ChannelUnknown, "unsupported platform", "", nil
	}
}

func isBrewManaged() bool {
	if !hasTool("brew") {
		return false
	}
	return installedBrewFormula() != ""
}

func installedBrewFormula() string {
	if !hasTool("brew") {
		return ""
	}
	for _, formula := range []string{"sshthing", "sshthing-beta"} {
		cmd := exec.Command("brew", "list", "--versions", formula)
		out, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(out)) != "" {
			return formula
		}
	}
	return ""
}

func hasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func detectPathHealth(ctx context.Context, desiredExe string) (PathHealth, error) {
	switch runtime.GOOS {
	case "windows":
		return detectWindowsPathHealth(desiredExe)
	default:
		return PathHealth{Healthy: true, Message: "PATH health checks currently implemented on Windows"}, nil
	}
}

func PathHealthLabel(ph PathHealth) string {
	if ph.Healthy {
		return "Healthy"
	}
	if strings.TrimSpace(ph.Message) == "" {
		return "Conflict"
	}
	return ph.Message
}

func ChannelLabel(ch Channel, detail string) string {
	base := string(ch)
	if strings.TrimSpace(detail) == "" {
		return base
	}
	if len(detail) > 44 {
		detail = detail[:44]
	}
	return fmt.Sprintf("%s (%s)", base, detail)
}
