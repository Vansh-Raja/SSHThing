package mount

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

type Mount struct {
	HostID     int
	Hostname   string
	LocalPath  string
	RemotePath string
	KeyPath    string
	PID        int
}

type PreparedMount struct {
	HostID    int
	Hostname  string
	LocalPath string

	remoteSpec string
	remotePath string
	display    string

	keyPath string
	cmd     *exec.Cmd
}

func (p *PreparedMount) Cmd() *exec.Cmd { return p.cmd }

type Manager struct {
	mu       sync.Mutex
	active   map[int]*Mount
	sshfsBin string
	diskutil string
}

func NewManager() *Manager {
	return &Manager{
		active: make(map[int]*Mount),
	}
}

func (m *Manager) IsMounted(hostID int) (bool, *Mount) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mnt, ok := m.active[hostID]
	return ok, mnt
}

func (m *Manager) CheckPrereqs() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("⚠ Finder mounts are currently supported only on macOS (darwin)")
	}

	// Prefer the standard `sshfs` name, but allow variants.
	for _, name := range []string{"sshfs", "sshfs-fuse-t"} {
		if p, err := exec.LookPath(name); err == nil {
			m.sshfsBin = p
			break
		}
	}
	if m.sshfsBin == "" {
		return errors.New("⚠ Mount (beta) requires FUSE-T + SSHFS.\nInstall:\n  brew install --cask fuse-t\n  brew tap macos-fuse-t/homebrew-cask\n  brew install --cask fuse-t-sshfs")
	}

	if _, err := exec.LookPath("umount"); err != nil {
		return fmt.Errorf("⚠ missing required tool: umount")
	}
	if _, err := exec.LookPath("open"); err != nil {
		return fmt.Errorf("⚠ missing required tool: open")
	}
	if _, err := exec.LookPath("mount"); err != nil {
		return fmt.Errorf("⚠ missing required tool: mount")
	}
	if p, err := exec.LookPath("diskutil"); err == nil {
		m.diskutil = p
	}
	return nil
}

func mountRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sshthing", "mounts"), nil
}

func mountKeyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sshthing", "mount-keys"), nil
}

func mountKeyPathFor(hostID int) (string, error) {
	dir, err := mountKeyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("host_%d.key", hostID)), nil
}

func normalizeKeyForFile(privateKey string) string {
	privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
	privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
	if privateKey != "" && !strings.HasSuffix(privateKey, "\n") {
		privateKey += "\n"
	}
	return privateKey
}

func writeMountKeyFile(hostID int, privateKey string) (string, error) {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return "", nil
	}
	keyPath, err := mountKeyPathFor(hostID)
	if err != nil {
		return "", err
	}
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return "", err
	}
	content := []byte(normalizeKeyForFile(privateKey))
	if err := os.WriteFile(keyPath, content, 0600); err != nil {
		return "", err
	}
	return keyPath, nil
}

func cleanupKeyFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	size := info.Size()
	if size > 0 {
		f, err := os.OpenFile(path, os.O_WRONLY, 0600)
		if err == nil {
			zeros := make([]byte, size)
			_, _ = f.Write(zeros)
			_ = f.Close()
		}
	}
	_ = os.Remove(path)
}

func safeMountName(hostname string, port int) string {
	base := strings.TrimSpace(hostname)
	if base == "" {
		base = "host"
	}
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, base)
	base = strings.Trim(base, "._-")
	if base == "" {
		base = "host"
	}
	if port != 0 && port != 22 {
		base = fmt.Sprintf("%s_%d", base, port)
	}
	return base
}

func remoteSpecFor(conn ssh.Connection, remotePath string) string {
	target := conn.Username + "@" + conn.Hostname
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		// Omit dir to mount remote home (sshfs treats missing dir as home).
		return target + ":"
	}
	return target + ":" + remotePath
}

func (m *Manager) PrepareMount(hostID int, conn ssh.Connection, remotePath string, displayName string) (*PreparedMount, error) {
	m.mu.Lock()
	_, alreadyMounted := m.active[hostID]
	m.mu.Unlock()
	if alreadyMounted {
		return nil, fmt.Errorf("⚠ host is already mounted")
	}

	if err := m.CheckPrereqs(); err != nil {
		return nil, err
	}

	root, err := mountRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}

	mountName := safeMountName(conn.Hostname, conn.Port)
	localPath := filepath.Join(root, mountName)
	if err := os.MkdirAll(localPath, 0700); err != nil {
		return nil, err
	}

	keyPath, err := writeMountKeyFile(hostID, conn.PrivateKey)
	if err != nil {
		return nil, err
	}

	remoteSpec := remoteSpecFor(conn, remotePath)

	// Build sshfs args:
	// sshfs [user@]host:[dir] mountpoint [options]
	args := []string{remoteSpec, localPath}

	mountOpts := []string{
		"reconnect",
		fmt.Sprintf("volname=%s", strings.TrimSpace(displayName)),
		"defer_permissions",
	}
	args = append(args, "-o", strings.Join(mountOpts, ","))

	// SSH options passed through.
	args = append(args, "-o", "StrictHostKeyChecking="+strictHostKeyChecking(conn.HostKeyPolicy))
	args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", keepAliveSeconds(conn.KeepAliveSeconds)))

	// Port: sshfs supports -p in many builds; this is the most explicit form.
	if conn.Port != 0 && conn.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", conn.Port))
	}

	if keyPath != "" {
		args = append(args, "-o", fmt.Sprintf("IdentityFile=%s", keyPath))
	}

	cmd := exec.Command(m.sshfsBin, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return &PreparedMount{
		HostID:     hostID,
		Hostname:   conn.Hostname,
		LocalPath:  localPath,
		remoteSpec: remoteSpec,
		remotePath: strings.TrimSpace(remotePath),
		display:    strings.TrimSpace(displayName),
		keyPath:    keyPath,
		cmd:        cmd,
	}, nil
}

func (m *Manager) AbortMount(p *PreparedMount) {
	if p == nil {
		return
	}
	cleanupKeyFile(p.keyPath)
}

func (m *Manager) FinalizeMount(p *PreparedMount) error {
	if p == nil {
		return fmt.Errorf("internal error: missing prepared mount")
	}

	ok, err := waitMounted(p.LocalPath, 2*time.Second)
	if err != nil {
		m.AbortMount(p)
		return err
	}
	if !ok {
		m.AbortMount(p)
		return fmt.Errorf("⚠ mount did not appear at %s", p.LocalPath)
	}

	pid := 0
	if p.cmd != nil && p.cmd.Process != nil {
		pid = p.cmd.Process.Pid
	}

	m.mu.Lock()
	m.active[p.HostID] = &Mount{
		HostID:     p.HostID,
		Hostname:   p.Hostname,
		LocalPath:  p.LocalPath,
		RemotePath: p.remotePath,
		KeyPath:    p.keyPath,
		PID:        pid,
	}
	m.mu.Unlock()

	// Open in Finder. If this fails, treat as non-fatal.
	_ = exec.Command("open", p.LocalPath).Run()
	return nil
}

func (m *Manager) PrepareUnmount(hostID int) (*exec.Cmd, string, error) {
	if err := m.CheckPrereqs(); err != nil {
		return nil, "", err
	}
	m.mu.Lock()
	mnt, ok := m.active[hostID]
	m.mu.Unlock()
	if !ok {
		return nil, "", fmt.Errorf("⚠ host is not mounted")
	}

	cmd := exec.Command("umount", mnt.LocalPath)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, mnt.LocalPath, nil
}

func (m *Manager) FinalizeUnmount(hostID int, primaryErr error) error {
	m.mu.Lock()
	mnt, ok := m.active[hostID]
	m.mu.Unlock()
	if !ok {
		// Already gone; treat as success.
		return nil
	}

	// If umount failed, try diskutil fallback (if available).
	if primaryErr != nil && m.diskutil != "" {
		_ = exec.Command(m.diskutil, "unmount", mnt.LocalPath).Run()
		stillMounted, _ := waitMounted(mnt.LocalPath, 200*time.Millisecond)
		if stillMounted {
			_ = exec.Command(m.diskutil, "unmount", "force", mnt.LocalPath).Run()
		}
	}

	// Wait for the filesystem to actually disappear before cleaning up the key.
	unmounted, err := waitUnmounted(mnt.LocalPath, 3*time.Second)
	if err != nil {
		return err
	}
	if !unmounted {
		// Keep the record + temp key in place so the mount can continue to function.
		if primaryErr != nil {
			return fmt.Errorf("⚠ unmount failed (mount still present): %v", primaryErr)
		}
		return fmt.Errorf("⚠ unmount did not complete (mount still present)")
	}

	cleanupKeyFile(mnt.KeyPath)

	m.mu.Lock()
	delete(m.active, hostID)
	m.mu.Unlock()

	return nil
}

func (m *Manager) UnmountAll() {
	m.mu.Lock()
	ids := make([]int, 0, len(m.active))
	for id := range m.active {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		cmd, _, err := m.PrepareUnmount(id)
		if err == nil {
			runErr := cmd.Run()
			_ = m.FinalizeUnmount(id, runErr)
			continue
		}
		_ = m.FinalizeUnmount(id, err)
	}
}

func (m *Manager) ListActive() []Mount {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Mount, 0, len(m.active))
	for _, v := range m.active {
		if v != nil {
			out = append(out, *v)
		}
	}
	return out
}

// RestoreMounted marks mounts as active if they are still mounted on the system.
// This is used on startup when the previous app session chose to keep mounts open.
func (m *Manager) RestoreMounted(records []Mount) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range records {
		ok, _ := isMounted(r.LocalPath)
		if ok {
			// key path is deterministic by host id; if it doesn't exist, leave empty.
			if r.KeyPath == "" {
				if kp, err := mountKeyPathFor(r.HostID); err == nil {
					if _, err2 := os.Stat(kp); err2 == nil {
						r.KeyPath = kp
					}
				}
			}
			cp := r
			m.active[r.HostID] = &cp
		} else {
			// Clean up stale key file if present.
			if kp, err := mountKeyPathFor(r.HostID); err == nil {
				cleanupKeyFile(kp)
			}
		}
	}
}

func waitMounted(localPath string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		mounted, err := isMounted(localPath)
		if err != nil {
			return false, err
		}
		if mounted {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func waitUnmounted(localPath string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		mounted, err := isMounted(localPath)
		if err != nil {
			return false, err
		}
		if !mounted {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func isMounted(localPath string) (bool, error) {
	out, err := exec.Command("mount").Output()
	if err != nil {
		return false, err
	}
	needle := " on " + localPath + " "
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, needle) || strings.HasSuffix(strings.TrimSpace(line), " on "+localPath) || strings.Contains(line, " on "+localPath+"(") {
			return true, nil
		}
		if strings.Contains(line, localPath) && strings.Contains(line, " on ") {
			// Fallback: best-effort match.
			if strings.Contains(line, " on "+localPath) {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsMounted reports whether a given local mount path is currently mounted.
func IsMounted(localPath string) (bool, error) {
	return isMounted(localPath)
}

func strictHostKeyChecking(policy string) string {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case "strict", "yes":
		return "yes"
	case "off", "no":
		return "no"
	default:
		return "accept-new"
	}
}

func keepAliveSeconds(v int) int {
	if v <= 0 {
		return 60
	}
	if v > 600 {
		return 600
	}
	return v
}
