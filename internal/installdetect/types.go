// Package installdetect locates SSHThing installs on the local machine
// (CLI binaries and the Electron desktop app) and classifies them by
// installation channel (Homebrew, Winget/Choco, AppImage, DMG, NSIS,
// standalone zip, or "bundled inside the GUI").
//
// The `sshthing update` CLI uses this to plan which artifacts to update
// and via what mechanism — e.g. delegate to `brew upgrade sshthing` for
// brew-managed CLIs, but mount-and-swap a fresh DMG for `/Applications/
// SSHThing.app`. Each Install carries enough metadata for the caller to
// pick the right apply path without re-doing detection work.
package installdetect

import "context"

// Kind distinguishes the running-shell CLI from the Electron desktop app.
type Kind string

const (
	KindCLI Kind = "cli"
	KindGUI Kind = "gui"
)

// Channel records HOW an install got there, which determines the update
// mechanism. Each channel maps to a specific apply path in the update
// command (delegate to a package manager, run an installer, swap a binary,
// mount and copy a DMG, etc.).
type Channel string

const (
	ChannelBrew          Channel = "brew"           // CLI installed via Homebrew (any tap or formula)
	ChannelWinget        Channel = "winget"         // CLI installed via winget on Windows
	ChannelChoco         Channel = "choco"          // CLI installed via Chocolatey on Windows
	ChannelStandaloneZip Channel = "standalone_zip" // CLI dropped manually (a tarball/zip extract)
	ChannelBundled       Channel = "bundled"        // CLI bundled inside the GUI's .app — not directly updatable
	ChannelDMG           Channel = "dmg"            // GUI installed from a macOS DMG
	ChannelNSIS          Channel = "nsis"           // GUI installed via the Windows NSIS installer
	ChannelAppImage      Channel = "appimage"       // GUI shipped as a Linux AppImage
	ChannelUnknown       Channel = "unknown"        // detector found something but couldn't classify it
)

// Install describes a single SSHThing artifact found on disk. The Path
// is the canonical filesystem path of the binary or .app bundle (for GUI)
// — e.g. `/usr/local/bin/sshthing`, `/Applications/SSHThing.app`.
//
// `Updatable` is false for `KindCLI` installs that are bundled inside a
// GUI .app: those get replaced when the parent .app is updated, and the
// CLI's update command refuses to touch them directly.
type Install struct {
	Kind      Kind    `json:"kind"`
	Channel   Channel `json:"channel"`
	Path      string  `json:"path"`
	Version   string  `json:"version,omitempty"`
	Updatable bool    `json:"updatable"`
	// Detail is a free-form human-readable note ("homebrew formula sshthing",
	// "AppImage in ~/Applications", etc.) shown by `sshthing update --doctor`.
	Detail string `json:"detail,omitempty"`
}

// Detect runs the platform-appropriate detector chain and returns every
// SSHThing install it can find. Returns at most one CLI Install and at
// most one GUI Install — the chain stops at the first match per slot so
// e.g. a bundled CLI inside the .app is only reported when no other CLI
// can be found.
//
// The function never errors out wholesale; individual detector failures
// (e.g. a missing `brew` command, a registry key that doesn't exist) are
// silently treated as "this channel didn't match" and the next detector
// runs.
func Detect(ctx context.Context) []Install {
	return detectPlatform(ctx)
}
