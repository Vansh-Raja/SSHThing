// `sshthing update` — single CLI entry point for self-updates of both
// the standalone `sshthing` binary and the SSHThing.app desktop bundle.
//
// Behaviour summary:
//   - Detects which install(s) live on disk via internal/installdetect.
//   - Refuses gracefully when invoked from inside the bundled CLI of a
//     running .app (the user must quit and run from a real terminal).
//   - For brew/winget/choco-managed CLIs, delegates to the package manager.
//   - For standalone CLI binaries, downloads the platform asset, verifies
//     the SHA256, and uses the existing handoff trick for the in-place swap.
//   - For the GUI, downloads the platform installer (DMG/NSIS/AppImage),
//     verifies, optionally code-signing-checks (macOS), and replaces.
//   - Each artefact is updated independently; one failing doesn't roll back
//     the other.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/installdetect"
	"github.com/Vansh-Raja/SSHThing/internal/update"
)

// Exit codes for `--check`. The plain `update` invocation uses 0 / 1
// for success / generic failure (subset of these).
const (
	exitUpToDate    = 0 // no update available (only meaningful with --check)
	exitUpdateAvail = 1 // (--check only) at least one install can be updated
	exitError       = 2 // network / IO / permission / partial failure
)

type updateFlags struct {
	check   bool // --check: print availability, don't apply
	beta    bool // --beta: use the beta release channel for this run
	cliOnly bool // --cli: only update the CLI install
	guiOnly bool // --gui: only update the GUI install
	yes     bool // --yes: skip the [Y/n] prompt
	doctor  bool // --doctor: print detected installs and exit
}

func runUpdate(args []string) error {
	flags, err := parseUpdateArgs(args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	installs := installdetect.Detect(ctx)

	if flags.doctor {
		printDoctor(installs, flags)
		return nil
	}

	if len(installs) == 0 {
		return fmt.Errorf("no SSHThing installs detected on this machine")
	}

	// Refuse-class: if the only install we found is the bundled CLI inside
	// the .app, the user is running this command from a context that can't
	// safely update anything.
	if onlyBundledCLI(installs) {
		fmt.Println("This `sshthing` is bundled inside SSHThing.app and can't update itself in place.")
		fmt.Println("Quit SSHThing and run `sshthing update` from a regular terminal.")
		return nil
	}

	// Channel selection: explicit `--beta` always wins; otherwise infer
	// from the detected install (a `sshthing-beta` brew install defaults
	// to the beta channel, anything else to stable).
	channel := inferChannel(installs, flags.beta)

	currentVersion := strings.TrimSpace(version)
	if currentVersion == "" {
		currentVersion = "dev"
	}

	fmt.Println("Checking releases…")
	check, err := update.Check(ctx, currentVersion, channel)
	if err != nil {
		return fmt.Errorf("could not reach the release feed: %w", err)
	}

	plan := buildPlan(installs, check, flags)
	if len(plan) == 0 {
		fmt.Printf("Already on latest %s release (%s).\n", channel, check.LatestVersion)
		return nil
	}

	printPlan(check, plan)

	if flags.check {
		// `--check` is a dry run: anything in the plan means an update is
		// available. Exit code communicates that to scripts.
		os.Exit(exitUpdateAvail)
		return nil // unreachable
	}

	if !flags.yes {
		if !isInteractive() {
			return fmt.Errorf("refusing to update non-interactively without --yes (stdin is not a TTY)")
		}
		if !confirm("Continue?") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	return execPlan(ctx, plan)
}

// parseUpdateArgs is intentionally hand-rolled (no `flag` package) so the
// update subcommand stays consistent with the existing `sshthing exec`,
// `sshthing cp`, etc. parsing style. Unknown flags are an error.
func parseUpdateArgs(args []string) (updateFlags, error) {
	var f updateFlags
	for _, a := range args {
		switch a {
		case "--check":
			f.check = true
		case "--beta":
			f.beta = true
		case "--cli":
			f.cliOnly = true
		case "--gui":
			f.guiOnly = true
		case "--yes", "-y":
			f.yes = true
		case "--doctor":
			f.doctor = true
		case "-h", "--help":
			printUpdateHelp()
			os.Exit(0)
		default:
			return f, fmt.Errorf("unknown flag %q (run `sshthing update --help`)", a)
		}
	}
	if f.cliOnly && f.guiOnly {
		return f, fmt.Errorf("--cli and --gui are mutually exclusive")
	}
	return f, nil
}

func printUpdateHelp() {
	fmt.Println("sshthing update — self-update SSHThing's CLI and/or GUI install.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sshthing update [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --check    Print availability and exit (0=current, 1=update avail, 2=error).")
	fmt.Println("  --beta     Use the beta release channel for this run only.")
	fmt.Println("  --cli      Only update the CLI install.")
	fmt.Println("  --gui      Only update the GUI install.")
	fmt.Println("  --yes,-y   Skip the [Y/n] confirmation prompt.")
	fmt.Println("  --doctor   Print detected installs + paths and exit.")
}

// planEntry describes one artefact's update operation.
type planEntry struct {
	install   installdetect.Install
	asset     update.AssetInfo // chosen asset for the platform; empty for package-manager delegation
	checksums update.AssetInfo
	latest    string // human-readable target version
	how       string // short description shown in the plan ("via brew", "via DMG (~150 MB)", etc.)
}

func buildPlan(installs []installdetect.Install, check update.CheckResult, flags updateFlags) []planEntry {
	var out []planEntry
	for _, ins := range installs {
		if !ins.Updatable {
			continue
		}
		switch ins.Kind {
		case installdetect.KindCLI:
			if flags.guiOnly {
				continue
			}
			how := describeCLIChannel(ins.Channel)
			entry := planEntry{install: ins, latest: check.LatestVersion, how: how}
			if needsCLIDownload(ins.Channel) {
				entry.asset = check.Asset
				entry.checksums = check.Checksums
				if entry.asset.URL == "" {
					// No CLI asset for this platform — skip rather than
					// blocking the GUI update behind a missing file.
					continue
				}
			}
			if entry.asset.URL == "" && !packageManagerCanUpdate(ins.Channel) {
				continue
			}
			if !cliNeedsUpdate(ins.Version, check.LatestVersion, check.UpdateAvailable) {
				continue
			}
			out = append(out, entry)
		case installdetect.KindGUI:
			if flags.cliOnly {
				continue
			}
			asset := update.FindGUIAsset(check.AllAssets)
			if asset.URL == "" {
				continue
			}
			if !guiNeedsUpdate(ins.Version, check.LatestVersion, check.UpdateAvailable) {
				continue
			}
			out = append(out, planEntry{
				install:   ins,
				asset:     asset,
				checksums: check.Checksums,
				latest:    check.LatestVersion,
				how:       describeGUIChannel(ins.Channel, asset.Name),
			})
		}
	}
	return out
}

func describeCLIChannel(c installdetect.Channel) string {
	switch c {
	case installdetect.ChannelBrew:
		return "via `brew upgrade`"
	case installdetect.ChannelWinget:
		return "via `winget upgrade`"
	case installdetect.ChannelChoco:
		return "via `choco upgrade`"
	case installdetect.ChannelStandaloneZip:
		return "via direct binary swap"
	}
	return string(c)
}

func describeGUIChannel(c installdetect.Channel, assetName string) string {
	switch c {
	case installdetect.ChannelDMG:
		return "via DMG (" + assetName + ")"
	case installdetect.ChannelNSIS:
		return "via Windows installer (" + assetName + ")"
	case installdetect.ChannelAppImage:
		return "via AppImage replace (" + assetName + ")"
	}
	return string(c)
}

// needsCLIDownload returns true for CLI channels that fall back to the
// raw zip/binary asset rather than delegating to a package manager.
func needsCLIDownload(c installdetect.Channel) bool {
	return c == installdetect.ChannelStandaloneZip
}

func packageManagerCanUpdate(c installdetect.Channel) bool {
	switch c {
	case installdetect.ChannelBrew, installdetect.ChannelWinget, installdetect.ChannelChoco:
		return true
	}
	return false
}

// cliNeedsUpdate decides whether to add a CLI install to the plan. We
// MUST never propose a downgrade — e.g. a user on `sshthing-beta`
// (3.0.0-beta.1) running `--check` without `--beta` would otherwise be
// told to "upgrade" to whatever the latest stable is (2.0.2). Compare
// versions semver-aware and return true only when the latest is
// strictly greater than what's installed.
//
// `latestNewer` is the broader Check() result against the *running
// binary*'s version, used as a fallback when the install itself
// doesn't expose a parsed version (e.g. a stripped Linux dpkg entry).
func cliNeedsUpdate(installed, latest string, latestNewer bool) bool {
	if strings.TrimSpace(latest) == "" {
		return false
	}
	if strings.TrimSpace(installed) == "" {
		return latestNewer
	}
	return update.CompareVersions(installed, latest) < 0
}

// guiNeedsUpdate is the same shape as cliNeedsUpdate but for a GUI
// install. Same downgrade-refusal rule applies.
func guiNeedsUpdate(installed, latest string, latestNewer bool) bool {
	return cliNeedsUpdate(installed, latest, latestNewer)
}

// inferChannel guesses the right release channel from the detected
// installs when the user didn't pass `--beta` explicitly. A user on the
// `sshthing-beta` brew formula expects beta updates by default;
// otherwise we default to stable.
func inferChannel(installs []installdetect.Install, betaFlag bool) update.ReleaseChannel {
	if betaFlag {
		return update.ReleaseChannelBeta
	}
	for _, ins := range installs {
		if ins.Kind != installdetect.KindCLI {
			continue
		}
		// The brew detector encodes the formula name into Detail (e.g.
		// "homebrew formula sshthing-beta"). Falling back to a substring
		// match keeps this resilient to small phrasing tweaks.
		if strings.Contains(ins.Detail, "sshthing-beta") {
			return update.ReleaseChannelBeta
		}
	}
	return update.ReleaseChannelStable
}

func onlyBundledCLI(installs []installdetect.Install) bool {
	if len(installs) != 1 {
		return false
	}
	return installs[0].Kind == installdetect.KindCLI && installs[0].Channel == installdetect.ChannelBundled
}

func printDoctor(installs []installdetect.Install, flags updateFlags) {
	fmt.Println("SSHThing install detection:")
	fmt.Println()
	if len(installs) == 0 {
		fmt.Println("  (no installs found)")
		return
	}
	for _, ins := range installs {
		updatable := "yes"
		if !ins.Updatable {
			updatable = "no"
		}
		ver := strings.TrimSpace(ins.Version)
		if ver == "" {
			ver = "(unknown version)"
		}
		fmt.Printf("  %-3s  %-14s  %s\n", ins.Kind, ins.Channel, ins.Path)
		fmt.Printf("       version: %s\n", ver)
		fmt.Printf("       updatable: %s\n", updatable)
		if ins.Detail != "" {
			fmt.Printf("       %s\n", ins.Detail)
		}
		fmt.Println()
	}
	// Report the channel that an actual `sshthing update` invocation
	// would use, including the inferred-from-install case. Previously
	// this read flags.beta directly, so a user on `sshthing-beta`
	// running `--doctor` without `--beta` was told "Release channel:
	// stable" even though the real update would correctly use beta.
	channel := inferChannel(installs, flags.beta)
	fmt.Printf("Release channel: %s\n", channel)
}

func printPlan(check update.CheckResult, plan []planEntry) {
	fmt.Println()
	fmt.Println("Updates available:")
	for _, p := range plan {
		from := strings.TrimSpace(p.install.Version)
		if from == "" {
			from = "?"
		}
		fmt.Printf("  %-3s  %s → %s  %s\n", p.install.Kind, from, p.latest, p.how)
	}
	fmt.Println()
	if check.ReleaseURL != "" {
		fmt.Printf("Release notes: %s\n", check.ReleaseURL)
		fmt.Println()
	}
}

func confirm(prompt string) bool {
	fmt.Printf("%s [Y/n] ", prompt)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		// Empty input is a yes (capital Y default); only an explicit "n"
		// or "no" aborts.
		return true
	}
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "" || input == "y" || input == "yes"
}

// execPlan runs each planned update in order, printing status as it goes.
// Per the design doc: artefacts are independent — a CLI failure doesn't
// block the GUI attempt, and the overall process exits non-zero only if
// at least one entry failed.
func execPlan(ctx context.Context, plan []planEntry) error {
	var anyFailed bool
	for _, p := range plan {
		fmt.Printf("[%s] starting…\n", p.install.Kind)
		err := applyOne(ctx, p)
		if err != nil {
			fmt.Printf("[%s] FAILED: %v\n", p.install.Kind, err)
			anyFailed = true
			continue
		}
		fmt.Printf("[%s] updated to %s ✓\n", p.install.Kind, p.latest)
	}
	if anyFailed {
		return fmt.Errorf("one or more artefacts failed to update — see above")
	}
	return nil
}

func applyOne(ctx context.Context, p planEntry) error {
	switch p.install.Kind {
	case installdetect.KindCLI:
		return applyCLI(ctx, p)
	case installdetect.KindGUI:
		return applyGUI(ctx, p)
	}
	return fmt.Errorf("unknown install kind: %s", p.install.Kind)
}

func applyCLI(ctx context.Context, p planEntry) error {
	switch p.install.Channel {
	case installdetect.ChannelBrew:
		// `brew upgrade` is idempotent and self-contained: it pulls a new
		// bottle, swaps the keg, updates the symlinks. No handoff trick
		// needed because brew operates on its own paths, not on /usr/bin.
		formula := "sshthing"
		if strings.Contains(p.install.Detail, "sshthing-beta") {
			formula = "sshthing-beta"
		}
		cmd := exec.CommandContext(ctx, "brew", "upgrade", formula)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("brew upgrade %s: %w (%s)", formula, err, strings.TrimSpace(string(out)))
		}
		return nil
	case installdetect.ChannelWinget:
		args := []string{"upgrade", "--name", "sshthing", "--silent",
			"--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
		out, err := exec.CommandContext(ctx, "winget", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("winget upgrade: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	case installdetect.ChannelChoco:
		out, err := exec.CommandContext(ctx, "choco", "upgrade", "sshthing", "-y").CombinedOutput()
		if err != nil {
			return fmt.Errorf("choco upgrade: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	case installdetect.ChannelStandaloneZip:
		// Use the existing applyReplaceMode pathway: it knows about the
		// handoff trick for swapping a running binary, downloading the
		// platform zip, and verifying SHA256. We synthesize a minimal
		// CheckResult so we can reuse that code without duplicating it.
		check := update.CheckResult{
			UpdateAvailable: true,
			ApplyMode:       update.ApplyModeReplaceBin,
			Asset:           p.asset,
			Checksums:       p.checksums,
		}
		exe := p.install.Path
		result, err := update.Apply(ctx, check, exe)
		if err != nil {
			return err
		}
		if result.Handoff != nil {
			if err := update.LaunchHandoff(result.Handoff); err != nil {
				return fmt.Errorf("launch handoff: %w", err)
			}
			fmt.Println("       (handoff helper launched — the binary will be replaced when this process exits)")
		}
		return nil
	}
	return fmt.Errorf("unsupported CLI channel %q", p.install.Channel)
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
