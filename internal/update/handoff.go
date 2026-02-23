package update

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const HandoffArg = "--update-handoff"

func LaunchHandoff(action *HandoffAction) error {
	if action == nil {
		return fmt.Errorf("missing handoff action")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "sshthing-handoff-")
	if err != nil {
		return err
	}
	actionPath := filepath.Join(tmpDir, "action.json")
	b, err := json.Marshal(action)
	if err != nil {
		return err
	}
	if err := os.WriteFile(actionPath, b, 0600); err != nil {
		return err
	}

	helperExe := filepath.Join(tmpDir, filepath.Base(exe))
	if err := copyFile(exe, helperExe, 0755); err != nil {
		return err
	}

	cmd := exec.Command(helperExe, HandoffArg, actionPath)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = windowsHiddenSysProcAttr()
	}
	return cmd.Start()
}

func RunHandoffFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var action HandoffAction
	if err := json.Unmarshal(b, &action); err != nil {
		return err
	}

	waitForParent(action.ParentPID)

	switch action.Type {
	case HandoffInstaller:
		cmd := exec.Command(action.InstallerPath, action.InstallerArgs...)
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = windowsHiddenSysProcAttr()
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	case HandoffReplace:
		tmpTarget := action.TargetBinaryPath + ".new"
		if err := copyFile(action.NewBinaryPath, tmpTarget, 0755); err != nil {
			return err
		}
		if err := os.Rename(tmpTarget, action.TargetBinaryPath); err != nil {
			_ = os.Remove(tmpTarget)
			return err
		}
	default:
		return fmt.Errorf("unknown handoff action: %s", action.Type)
	}

	if action.RelaunchPath != "" {
		cmd := exec.Command(action.RelaunchPath, action.RelaunchArgs...)
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = windowsHiddenSysProcAttr()
		}
		_ = cmd.Start()
	}

	if action.CleanupDir != "" {
		_ = os.RemoveAll(action.CleanupDir)
	}
	_ = os.RemoveAll(filepath.Dir(path))
	return nil
}

func waitForParent(pid int) {
	if pid <= 0 {
		return
	}
	for i := 0; i < 200; i++ {
		if !isProcessRunning(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false
		}
		return string(out) != "" && (containsIgnoreCase(string(out), strconv.Itoa(pid)) && !containsIgnoreCase(string(out), "No tasks are running"))
	}
	err = proc.Signal(os.Signal(nil))
	return err == nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func containsIgnoreCase(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (stringContainsFold(s, sub)))
}

func stringContainsFold(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexFold(s, sub) >= 0)
}

func indexFold(s, sep string) int {
	for i := 0; i+len(sep) <= len(s); i++ {
		if equalFoldASCII(s[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
