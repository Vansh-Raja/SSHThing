package update

import "time"

type Channel string

const (
	ChannelUnknown          Channel = "unknown"
	ChannelWindowsInstaller Channel = "windows_installer"
	ChannelMacOSBrew        Channel = "macos_brew"
	ChannelStandalone       Channel = "standalone"
	ChannelLinuxGuidance    Channel = "linux_guidance"
)

type ApplyMode string

const (
	ApplyModeNone       ApplyMode = "none"
	ApplyModeCommand    ApplyMode = "command"
	ApplyModeInstaller  ApplyMode = "installer"
	ApplyModeReplaceBin ApplyMode = "replace_binary"
	ApplyModeGuidance   ApplyMode = "guidance"
)

type PathHealth struct {
	Healthy      bool
	ResolvedPath string
	DesiredPath  string
	Message      string
	Conflicts    []string
}

type AssetInfo struct {
	Name string
	URL  string
}

type CheckResult struct {
	CheckedAt       time.Time
	ETag            string
	CurrentVersion  string
	LatestVersion   string
	LatestTag       string
	ReleaseURL      string
	UpdateAvailable bool
	Channel         Channel
	ChannelDetail   string
	PathHealth      PathHealth
	ApplyMode       ApplyMode
	ApplyCommand    []string
	Guidance        []string
	Asset           AssetInfo
	Checksums       AssetInfo
	InstallerExe    string
}

type HandoffActionType string

const (
	HandoffInstaller HandoffActionType = "installer"
	HandoffReplace   HandoffActionType = "replace"
)

type HandoffAction struct {
	Type             HandoffActionType `json:"type"`
	ParentPID        int               `json:"parent_pid"`
	InstallerPath    string            `json:"installer_path,omitempty"`
	InstallerArgs    []string          `json:"installer_args,omitempty"`
	NewBinaryPath    string            `json:"new_binary_path,omitempty"`
	TargetBinaryPath string            `json:"target_binary_path,omitempty"`
	RelaunchPath     string            `json:"relaunch_path"`
	RelaunchArgs     []string          `json:"relaunch_args,omitempty"`
	CleanupDir       string            `json:"cleanup_dir,omitempty"`
}

type ApplyResult struct {
	Success       bool
	Message       string
	NeedsRelaunch bool
	Handoff       *HandoffAction
	RelaunchPath  string
	RelaunchArgs  []string
}
