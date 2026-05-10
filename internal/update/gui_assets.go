package update

import (
	"runtime"
	"strings"
)

// FindGUIAsset picks the right GUI artefact for the current platform out
// of a release's asset list. Returns AssetInfo{} if no matching asset is
// available (e.g. a Linux user querying a release that only published
// macOS+Windows builds).
//
// Asset name conventions follow electron-builder defaults:
//
//	macOS DMG:        SSHThing-<ver>.dmg, SSHThing-<ver>-arm64.dmg, SSHThing-<ver>-x64.dmg
//	Windows NSIS:     SSHThing-Setup-<ver>.exe, SSHThing Setup <ver>.exe
//	Linux AppImage:   SSHThing-<ver>.AppImage, SSHThing-<ver>-arm64.AppImage
//
// For multi-arch macOS releases we prefer the arch-specific asset over a
// universal one (smaller download). When only one DMG exists we take it
// regardless of arch — electron-builder's universal target uses no arch
// suffix.
func FindGUIAsset(assets []AssetInfo) AssetInfo {
	switch runtime.GOOS {
	case "darwin":
		return findMacGUIAsset(assets)
	case "windows":
		return findWindowsGUIAsset(assets)
	case "linux":
		return findLinuxGUIAsset(assets)
	}
	return AssetInfo{}
}

func findMacGUIAsset(assets []AssetInfo) AssetInfo {
	wantArch := runtime.GOARCH // "arm64" or "amd64"; electron-builder uses "arm64" / "x64"
	if wantArch == "amd64" {
		wantArch = "x64"
	}

	var archMatch, universal AssetInfo
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if !strings.HasSuffix(name, ".dmg") {
			continue
		}
		if !strings.HasPrefix(name, "sshthing") {
			continue
		}
		switch {
		case strings.Contains(name, "-"+wantArch+"."):
			archMatch = a
		case !strings.Contains(name, "-arm64.") && !strings.Contains(name, "-x64."):
			universal = a
		}
	}
	if archMatch.URL != "" {
		return archMatch
	}
	return universal
}

func findWindowsGUIAsset(assets []AssetInfo) AssetInfo {
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if !strings.HasSuffix(name, ".exe") {
			continue
		}
		// "Setup" appears in both NSIS conventions (`SSHThing Setup x.y.z.exe`
		// and `SSHThing-Setup-x.y.z.exe`); the daemon binary asset is
		// distinct (`sshthing-setup-windows-amd64.exe`) — we differentiate
		// by checking for the "sshthing" prefix and excluding the daemon
		// flavour names.
		if !strings.Contains(name, "sshthing") {
			continue
		}
		if !strings.Contains(name, "setup") {
			continue
		}
		// Skip the standalone CLI Windows installer if it ever lives in
		// the same release; it's matched by `resolveReleaseAsset`.
		if strings.Contains(name, "-windows-amd64") || strings.Contains(name, "-windows-arm64") {
			continue
		}
		return AssetInfo{Name: a.Name, URL: a.URL}
	}
	return AssetInfo{}
}

func findLinuxGUIAsset(assets []AssetInfo) AssetInfo {
	wantArch := runtime.GOARCH
	if wantArch == "amd64" {
		wantArch = "x86_64"
	}

	var archMatch, universal AssetInfo
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if !strings.HasSuffix(name, ".appimage") {
			continue
		}
		if !strings.HasPrefix(name, "sshthing") {
			continue
		}
		switch {
		case strings.Contains(name, "-"+wantArch+"."), strings.Contains(name, "-"+strings.ToLower(wantArch)+"."):
			archMatch = a
		case !strings.Contains(name, "-arm64.") && !strings.Contains(name, "-x86_64."):
			universal = a
		}
	}
	if archMatch.URL != "" {
		return archMatch
	}
	return universal
}
