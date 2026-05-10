package mount

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
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

	keyPath   string
	cmd       *exec.Cmd
	stderrBuf bytes.Buffer

	// passwordClean is set when WrapForPasswordAuth allocated an askpass
	// server / sshpass pipe. It must be called when the prepared cmd exits
	// (success or failure) to release fds and shut the askpass listener.
	passwordClean func()
}

// runPasswordCleanup invokes passwordClean once if set, then nils it out.
func (p *PreparedMount) runPasswordCleanup() {
	if p.passwordClean != nil {
		p.passwordClean()
		p.passwordClean = nil
	}
}

// fuseFilesystemAvailable reports whether at least one of the known macOS FUSE
// backends is installed. fuse-t (NFS-based, no kext) lives under
// /Library/Application Support/fuse-t/. macFUSE (kext-based) lives at
// /Library/Filesystems/macfuse.fs.
func fuseFilesystemAvailable() bool {
	candidates := []string{
		"/Library/Application Support/fuse-t/lib/libfuse3.dylib",
		"/Library/Application Support/fuse-t/lib/libfuse-t-1.2.1.dylib",
		"/Library/Filesystems/fuse-t.fs",
		"/Library/Filesystems/macfuse.fs",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// detectFuseBackend returns "fuse-t", "macfuse", or "" depending on which
// dynamic libraries the sshfs binary links against. Best-effort; ignores
// errors so it never blocks a mount attempt.
func detectFuseBackend(sshfsPath string) string {
	out, err := exec.Command("otool", "-L", sshfsPath).Output()
	if err != nil {
		return ""
	}
	s := string(out)
	if strings.Contains(s, "fuse-t/lib/libfuse") || strings.Contains(s, "libfuse3") {
		return "fuse-t"
	}
	if strings.Contains(s, "/usr/local/lib/libfuse.2") || strings.Contains(s, "libosxfuse") || strings.Contains(s, "macfuse") {
		return "macfuse"
	}
	return ""
}

func (p *PreparedMount) Cmd() *exec.Cmd     { return p.cmd }
func (p *PreparedMount) RemotePath() string { return p.remotePath }
func (p *PreparedMount) Stderr() string     { return strings.TrimSpace(p.stderrBuf.String()) }

type Manager struct {
	mu         sync.Mutex
	active     map[int]*Mount
	sshfsBin   string
	diskutil   string
	fusermount string
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
	switch runtime.GOOS {
	case "darwin":
		return m.checkPrereqsDarwin()
	case "linux":
		return m.checkPrereqsLinux()
	case "windows":
		return fmt.Errorf("Mount feature is not yet available on Windows.\nThis feature requires FUSE filesystem support.")
	default:
		return fmt.Errorf("Mount feature is not supported on %s", runtime.GOOS)
	}
}

func (m *Manager) checkPrereqsDarwin() error {
	// On macOS, both the legacy macFUSE sshfs and the fuse-t cask install
	// a binary called `sshfs` at /usr/local/bin/sshfs. They overwrite one
	// another. We just take whatever's on PATH and rely on the user having
	// the working variant installed.
	for _, name := range []string{"sshfs"} {
		if p, err := exec.LookPath(name); err == nil {
			m.sshfsBin = p
			break
		}
	}
	if m.sshfsBin != "" {
		// Note which FUSE backend the binary links against for diagnostics.
		// fuse-t links libfuse3 (in /Library/Application Support/fuse-t/lib);
		// macFUSE links its own kext-based libfuse 2.x.
		if backend := detectFuseBackend(m.sshfsBin); backend != "" {
			log.Printf("mount: detected FUSE backend: %s (sshfs=%s)", backend, m.sshfsBin)
		}
	}
	if m.sshfsBin == "" {
		return errors.New("⚠ Mount (beta) requires SSHFS.\nInstall FUSE-T (recommended):\n  brew install --cask fuse-t\n  brew tap macos-fuse-t/homebrew-cask\n  brew install --cask fuse-t-sshfs\n\nOr macFUSE + SSHFS:\n  brew install --cask macfuse\n  brew install sshfs")
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

func (m *Manager) checkPrereqsLinux() error {
	if p, err := exec.LookPath("sshfs"); err == nil {
		m.sshfsBin = p
	}
	if m.sshfsBin == "" {
		return errors.New("⚠ Mount requires sshfs.\nInstall:\n  apt install sshfs        (Debian/Ubuntu)\n  dnf install fuse-sshfs   (Fedora/RHEL)\n  pacman -S sshfs          (Arch)")
	}

	if _, err := exec.LookPath("umount"); err != nil {
		return fmt.Errorf("⚠ missing required tool: umount")
	}

	// fusermount3 preferred, fusermount as fallback.
	for _, name := range []string{"fusermount3", "fusermount"} {
		if p, err := exec.LookPath(name); err == nil {
			m.fusermount = p
			break
		}
	}
	if m.fusermount == "" {
		return fmt.Errorf("⚠ missing required tool: fusermount (install fuse or fuse3)")
	}
	return nil
}

func mountRoot() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sshthing", "mounts"), nil
}

func mountKeyDir() (string, error) {
	// We deliberately do NOT use os.UserConfigDir() on macOS. That returns
	// `~/Library/Application Support` which contains a space, and sshfs
	// passes the IdentityFile path verbatim to ssh's `-o IdentityFile=`
	// parser. The OpenSSH config-line tokenizer splits on whitespace and
	// rejects the path with "keyword identityfile extra arguments at end
	// of line", which sshfs reports back as the cryptic "remote host has
	// disconnected".
	//
	// The cache dir on macOS is `~/Library/Caches`, which has no spaces.
	// On Linux it's $XDG_CACHE_HOME or `~/.cache` (also no spaces). On
	// Windows it's %LocalAppData% which can have spaces, but we don't
	// reach this path on Windows (mount is unsupported there).
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, "sshthing", "mount-keys")
	if strings.ContainsAny(dir, " \t") {
		// Last-resort fallback. /tmp is world-writable but each file is
		// chmod 0600 so the secret is still user-private; the directory
		// itself is the user's own subdir.
		dir = filepath.Join(os.TempDir(), "sshthing-mount-keys")
	}
	return dir, nil
}

func mountKeyPathFor(hostID int) (string, error) {
	dir, err := mountKeyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("host_%d.key", hostID)), nil
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
	if err := ssh.WritePrivateKeyFile(keyPath, privateKey); err != nil {
		return "", err
	}
	return keyPath, nil
}

func cleanupKeyFile(path string) {
	_ = ssh.SecureDeleteFile(path)
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

func (m *Manager) PrepareMount(hostID int, conn ssh.Connection, remotePath string, displayName string, localMountBase string) (*PreparedMount, error) {
	m.mu.Lock()
	_, alreadyMounted := m.active[hostID]
	m.mu.Unlock()
	if alreadyMounted {
		return nil, fmt.Errorf("⚠ host is already mounted")
	}

	if err := m.CheckPrereqs(); err != nil {
		return nil, err
	}

	root := strings.TrimSpace(localMountBase)
	if root == "" {
		var err error
		root, err = mountRoot()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("cannot create mount directory %s: %v", root, err)
	}

	// Prefer display name (server label) for the folder; fall back to hostname.
	label := strings.TrimSpace(displayName)
	if label == "" {
		label = conn.Hostname
	}
	mountName := safeMountName(label, conn.Port)
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

	var mountOpts []string
	if runtime.GOOS == "linux" {
		mountOpts = []string{"reconnect"}
	} else {
		mountOpts = []string{
			"reconnect",
			fmt.Sprintf("volname=%s", strings.TrimSpace(displayName)),
			"defer_permissions",
		}
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

	// Opt-in extra debug output via env var — set MOUNT_DEBUG=1 to get
	// sshfs internal trace + ssh protocol verbosity. Useful when "remote
	// host has disconnected" is the only error visible.
	if os.Getenv("MOUNT_DEBUG") == "1" {
		args = append(args, "-o", "sshfs_debug")
		args = append(args, "-o", "LogLevel=DEBUG3")
	}

	// For password-auth hosts we wrap sshfs with the same sshpass / askpass
	// plumbing ssh.Connect uses. Without this, sshfs tries to prompt for a
	// password and the daemon-spawned process has no terminal, so ssh drops
	// the connection ("remote host has disconnected").
	cmd, pwCleanup, err := ssh.WrapForPasswordAuth(m.sshfsBin, args, conn)
	if err != nil {
		return nil, fmt.Errorf("configure sshfs auth: %w", err)
	}
	cmd.Stdout = os.Stdout
	// IMPORTANT: do NOT inherit os.Stdin. In the TUI's tea.ExecProcess flow
	// the Bubble Tea runtime attaches a terminal anyway; in the daemon's
	// background spawn, inheriting stdin gives ssh an empty pipe to read
	// password prompts from, which then fails with "remote host has
	// disconnected". Leaving Stdin nil routes prompts through askpass.
	cmd.Stdin = nil

	// For key-auth mounts, disable any user-side ssh-agent on the way in.
	// macOS's system ssh-agent may contain other keys; mixing them with our
	// explicit IdentityFile can confuse ssh's auth ordering. For password
	// mounts the WrapForPasswordAuth path already set the askpass env, so
	// we leave it alone there.
	if conn.Password == "" && cmd.Env != nil {
		// Replace SSH_AUTH_SOCK with empty in the existing env slice rather
		// than rebuilding from os.Environ() (which would clobber any
		// askpass / sshpass plumbing the wrap helper installed).
		newEnv := make([]string, 0, len(cmd.Env)+1)
		seen := false
		for _, e := range cmd.Env {
			if strings.HasPrefix(e, "SSH_AUTH_SOCK=") {
				newEnv = append(newEnv, "SSH_AUTH_SOCK=")
				seen = true
				continue
			}
			newEnv = append(newEnv, e)
		}
		if !seen {
			newEnv = append(newEnv, "SSH_AUTH_SOCK=")
		}
		cmd.Env = newEnv
	}

	// Diagnostic logging — sshfs failures are notoriously opaque so we log
	// the binary, the resolved args (key file path / port / StrictHostKeyChecking
	// / sshfs options) and the auth mode. The actual command's argv lives on
	// cmd.Args (which differs from `args` when sshpass wraps the call).
	authMode := "key"
	if conn.Password != "" && conn.PrivateKey == "" {
		authMode = "password"
	} else if conn.Password == "" && conn.PrivateKey == "" {
		authMode = "agent"
	}
	log.Printf("mount: sshfs bin=%s authMode=%s remote=%s local=%s args=%v cmdArgs=%v",
		m.sshfsBin, authMode, remoteSpec, localPath, args, cmd.Args)

	p := &PreparedMount{
		HostID:        hostID,
		Hostname:      conn.Hostname,
		LocalPath:     localPath,
		remoteSpec:    remoteSpec,
		remotePath:    strings.TrimSpace(remotePath),
		display:       strings.TrimSpace(displayName),
		keyPath:       keyPath,
		cmd:           cmd,
		passwordClean: pwCleanup,
	}
	cmd.Stderr = &p.stderrBuf

	return p, nil
}

func (m *Manager) AbortMount(p *PreparedMount) {
	if p == nil {
		return
	}
	// Kill the sshfs cmd if it's still running. fuse-t spawns a userspace
	// NFS daemon (go-nfsv4) as a child of sshfs; if sshfs dies cleanly the
	// daemon usually goes with it. But on failure paths sshfs may have
	// exited while go-nfsv4 lingered, so we also force-kill any go-nfsv4
	// holding our specific mount point.
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_, _ = p.cmd.Process.Wait()
	}
	killStaleNFSDaemonForPath(p.LocalPath)
	// MOUNT_DEBUG=1 keeps the temp key around so we can reproduce the
	// failing sshfs invocation manually with the same args.
	if os.Getenv("MOUNT_DEBUG") != "1" {
		cleanupKeyFile(p.keyPath)
	} else if p.keyPath != "" {
		log.Printf("mount: MOUNT_DEBUG=1 keeping key file at %s for manual reproduction", p.keyPath)
	}
	p.runPasswordCleanup()
}

// killStaleNFSDaemonForPath finds any go-nfsv4 process holding the given
// mount path and force-kills it. fuse-t doesn't always tear its NFS
// userspace daemon down when sshfs dies on a failed mount, leaving a
// zombie that blocks future mounts at the same path.
func killStaleNFSDaemonForPath(localPath string) {
	if runtime.GOOS != "darwin" {
		return
	}
	out, err := exec.Command("ps", "-Ao", "pid,command").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "go-nfsv4") {
			continue
		}
		if !strings.Contains(line, localPath) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pid := fields[0]
		log.Printf("mount: cleanup killing stale go-nfsv4 pid=%s for %s", pid, localPath)
		_ = exec.Command("kill", "-9", pid).Run()
	}
}

func (m *Manager) FinalizeMount(p *PreparedMount) error {
	if p == nil {
		return fmt.Errorf("internal error: missing prepared mount")
	}

	// fuse-t spawns a userspace NFS daemon then runs through SSH auth before
	// the mount appears in the kernel mount table. 2 s was too tight for
	// some networks (the connection negotiates SSH + NFSv4 + handshakes).
	// Bump to 8 s; if mount still hasn't appeared, sshfs has either died
	// or is stuck and the error message is more useful than a quick retry.
	ok, err := waitMounted(p.LocalPath, 8*time.Second)
	if err != nil {
		log.Printf("mount: waitMounted error for %s: %v (stderr=%q)", p.LocalPath, err, p.Stderr())
		m.AbortMount(p)
		return err
	}
	if !ok {
		stderr := p.Stderr()
		hint := ""
		if !fuseFilesystemAvailable() {
			hint = "\n→ FUSE filesystem extension is missing. Install or reinstall fuse-t:" +
				"\n   brew tap macos-fuse-t/homebrew-cask" +
				"\n   brew reinstall --cask fuse-t fuse-t-sshfs" +
				"\n  (the cask may have downloaded the .pkg without running it; macOS will" +
				"\n   prompt to allow the system extension on the next install)."
		}
		log.Printf("mount: did not appear at %s; stderr=%q hint=%s", p.LocalPath, stderr, hint)
		m.AbortMount(p)
		errMsg := fmt.Sprintf("⚠ mount did not appear at %s", p.LocalPath)
		if stderr != "" {
			errMsg += "\n" + stderr
		}
		if hint != "" {
			errMsg += hint
		}
		return fmt.Errorf("%s", errMsg)
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

	// Mount is up — askpass server / sshpass pipe no longer needed; the
	// inner ssh transport has already negotiated its auth. Releasing now
	// keeps fd / port count tight while sshfs keeps running.
	p.runPasswordCleanup()

	// Open in file manager. If this fails, treat as non-fatal.
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", p.LocalPath).Run()
	case "linux":
		if xdgOpen, err := exec.LookPath("xdg-open"); err == nil {
			_ = exec.Command(xdgOpen, p.LocalPath).Run()
		}
	}
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

	var cmd *exec.Cmd
	if runtime.GOOS == "linux" && m.fusermount != "" {
		cmd = exec.Command(m.fusermount, "-u", mnt.LocalPath)
	} else {
		cmd = exec.Command("umount", mnt.LocalPath)
	}
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

	// If umount failed, try platform-specific fallback.
	if primaryErr != nil {
		switch runtime.GOOS {
		case "darwin":
			if m.diskutil != "" {
				_ = exec.Command(m.diskutil, "unmount", mnt.LocalPath).Run()
				stillMounted, _ := waitMounted(mnt.LocalPath, 200*time.Millisecond)
				if stillMounted {
					_ = exec.Command(m.diskutil, "unmount", "force", mnt.LocalPath).Run()
				}
			}
		case "linux":
			if m.fusermount != "" {
				// Lazy unmount as force fallback.
				_ = exec.Command(m.fusermount, "-uz", mnt.LocalPath).Run()
			}
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
	// Linux fast-path: read /proc/mounts directly instead of spawning mount(8).
	if runtime.GOOS == "linux" {
		return isMountedProc(localPath)
	}

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

func isMountedProc(localPath string) (bool, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		// Fall back to mount(8) if /proc/mounts is unavailable.
		out, err2 := exec.Command("mount").Output()
		if err2 != nil {
			return false, err2
		}
		return strings.Contains(string(out), " "+localPath+" "), nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// /proc/mounts format: device mountpoint fstype options ...
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == localPath {
			return true, nil
		}
	}
	return false, scanner.Err()
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
