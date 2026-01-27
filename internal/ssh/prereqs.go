package ssh

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// RequiredTools lists the OpenSSH tools needed by SSHThing
var RequiredTools = []string{"ssh", "sftp", "ssh-keygen"}

// CheckPrereqs verifies that required OpenSSH tools are available in PATH.
// Returns nil if all tools are found, otherwise returns an error with
// installation instructions appropriate for the current platform.
func CheckPrereqs() error {
	var missing []string
	for _, tool := range RequiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	var instructions string
	switch runtime.GOOS {
	case "windows":
		instructions = `On Windows, install OpenSSH:
  1. Open Settings > Apps > Optional Features
  2. Click "Add a feature"
  3. Find and install "OpenSSH Client"

Alternatively, if you have winget:
  winget install Microsoft.OpenSSH.Client`
	case "darwin":
		instructions = "OpenSSH should be pre-installed on macOS. Check your PATH."
	default:
		instructions = `Install OpenSSH using your package manager:
  Ubuntu/Debian: sudo apt install openssh-client
  Fedora/RHEL:   sudo dnf install openssh-clients
  Arch:          sudo pacman -S openssh`
	}

	return fmt.Errorf("missing required tools: %s\n\n%s",
		strings.Join(missing, ", "), instructions)
}

// CheckSSH verifies that ssh is available
func CheckSSH() error {
	if _, err := exec.LookPath("ssh"); err != nil {
		return fmt.Errorf("ssh not found in PATH")
	}
	return nil
}

// CheckSFTP verifies that sftp is available
func CheckSFTP() error {
	if _, err := exec.LookPath("sftp"); err != nil {
		return fmt.Errorf("sftp not found in PATH")
	}
	return nil
}

// CheckSSHKeygen verifies that ssh-keygen is available
func CheckSSHKeygen() error {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return fmt.Errorf("ssh-keygen not found in PATH")
	}
	return nil
}
