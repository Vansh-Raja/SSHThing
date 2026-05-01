package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

type Status string

const (
	StatusUnknown     Status = "unknown"
	StatusChecking    Status = "checking"
	StatusOnline      Status = "online"
	StatusOffline     Status = "offline"
	StatusTimeout     Status = "timeout"
	StatusAuthFailed  Status = "auth_failed"
	StatusUnsupported Status = "unsupported"
	StatusError       Status = "error"
)

type AuthMode string

const (
	AuthModeDefault  AuthMode = "default"
	AuthModeKey      AuthMode = "key"
	AuthModePassword AuthMode = "password"
)

type ProbeOptions struct {
	Timeout        time.Duration
	ConnectTimeout time.Duration
	AuthMode       AuthMode
}

type Result struct {
	TargetKey          string
	Status             Status
	CheckedAt          time.Time
	Latency            time.Duration
	Uptime             time.Duration
	CPUPercent         float64
	MemTotalBytes      int64
	MemAvailableBytes  int64
	DiskTotalBytes     int64
	DiskAvailableBytes int64
	GPUPresent         bool
	GPUName            string
	Error              string
}

type Stats struct {
	Status             Status
	UptimeSeconds      int64
	CPUPercent         float64
	MemTotalBytes      int64
	MemAvailableBytes  int64
	DiskTotalBytes     int64
	DiskAvailableBytes int64
	GPUPresent         bool
	GPUName            string
	Error              string
}

func Probe(ctx context.Context, conn ssh.Connection, opts ProbeOptions) Result {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.ConnectTimeout <= 0 {
		opts.ConnectTimeout = 5 * time.Second
	}

	started := time.Now()
	execResult, err := ssh.ConnectExecCaptured(ctx, conn, BuildLinuxProbeCommand(), ssh.ExecOptions{
		Timeout:           opts.Timeout,
		ConnectTimeout:    opts.ConnectTimeout,
		AllowPasswordAuth: opts.AuthMode == AuthModePassword,
		BatchMode:         opts.AuthMode != AuthModePassword,
	})
	result := Result{
		Status:    StatusOnline,
		CheckedAt: time.Now(),
		Latency:   time.Since(started),
	}
	if execResult.Duration > 0 {
		result.Latency = execResult.Duration
	}
	if err != nil {
		result.Status = ClassifyError(err, execResult.Stderr)
		result.Error = cleanErrorMessage(err, execResult.Stderr)
		return result
	}

	stats, err := ParseProbeOutput(execResult.Stdout)
	if err != nil {
		result.Status = StatusError
		result.Error = err.Error()
		return result
	}
	result.Status = stats.Status
	if result.Status == "" {
		result.Status = StatusOnline
	}
	result.Uptime = time.Duration(stats.UptimeSeconds) * time.Second
	result.CPUPercent = stats.CPUPercent
	result.MemTotalBytes = stats.MemTotalBytes
	result.MemAvailableBytes = stats.MemAvailableBytes
	result.DiskTotalBytes = stats.DiskTotalBytes
	result.DiskAvailableBytes = stats.DiskAvailableBytes
	result.GPUPresent = stats.GPUPresent
	result.GPUName = stats.GPUName
	result.Error = stats.Error
	return result
}

func ParseProbeOutput(output string) (Stats, error) {
	values := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if len(values) == 0 {
		return Stats{}, fmt.Errorf("empty health probe output")
	}

	status := Status(strings.TrimSpace(values["status"]))
	if status == "" || status == "ok" {
		status = StatusOnline
	}
	stats := Stats{Status: status, GPUName: values["gpu_name"], Error: values["error"]}
	if status == StatusUnsupported {
		return stats, nil
	}

	var err error
	if stats.UptimeSeconds, err = parseRequiredInt(values, "uptime_seconds"); err != nil {
		return Stats{}, err
	}
	if stats.CPUPercent, err = parseRequiredFloat(values, "cpu_percent"); err != nil {
		return Stats{}, err
	}
	if stats.MemTotalBytes, err = parseRequiredInt(values, "mem_total_bytes"); err != nil {
		return Stats{}, err
	}
	if stats.MemAvailableBytes, err = parseRequiredInt(values, "mem_available_bytes"); err != nil {
		return Stats{}, err
	}
	if stats.DiskTotalBytes, err = parseRequiredInt(values, "disk_total_bytes"); err != nil {
		return Stats{}, err
	}
	if stats.DiskAvailableBytes, err = parseRequiredInt(values, "disk_available_bytes"); err != nil {
		return Stats{}, err
	}
	stats.GPUPresent = strings.EqualFold(values["gpu_present"], "true") || values["gpu_present"] == "1" || strings.EqualFold(values["gpu_present"], "yes")
	return stats, nil
}

func BuildLinuxProbeCommand() string {
	return strings.TrimSpace(`
if [ ! -r /proc/stat ] || [ ! -r /proc/meminfo ] || [ ! -r /proc/uptime ]; then
  echo "status=unsupported"
  echo "error=linux /proc metrics are unavailable"
  exit 0
fi

read cpu user nice system idle iowait irq softirq steal guest guest_nice < /proc/stat
idle1=$((idle + iowait))
total1=$((user + nice + system + idle + iowait + irq + softirq + steal))
sleep 1
read cpu user nice system idle iowait irq softirq steal guest guest_nice < /proc/stat
idle2=$((idle + iowait))
total2=$((user + nice + system + idle + iowait + irq + softirq + steal))
diff_idle=$((idle2 - idle1))
diff_total=$((total2 - total1))
if [ "$diff_total" -gt 0 ]; then
  cpu_percent=$((100 * (diff_total - diff_idle) / diff_total))
else
  cpu_percent=0
fi

echo "status=ok"
awk '{printf "uptime_seconds=%d\n", $1}' /proc/uptime
printf 'cpu_percent=%s\n' "$cpu_percent"
awk '
  /MemTotal:/ { total=$2 * 1024 }
  /MemAvailable:/ { avail=$2 * 1024 }
  END {
    if (total == "") total = 0
    if (avail == "") avail = 0
    printf "mem_total_bytes=%.0f\n", total
    printf "mem_available_bytes=%.0f\n", avail
  }
' /proc/meminfo
df -P -k / 2>/dev/null | awk 'NR==2 { printf "disk_total_bytes=%.0f\n", $2 * 1024; printf "disk_available_bytes=%.0f\n", $4 * 1024 }'

gpu_present=false
gpu_name=
if command -v nvidia-smi >/dev/null 2>&1; then
  gpu_name=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | sed -n '1p')
  if [ -n "$gpu_name" ]; then
    gpu_present=true
  fi
elif command -v lspci >/dev/null 2>&1 && lspci | grep -Eiq 'vga|3d|display'; then
  gpu_present=true
  gpu_name=$(lspci | grep -Ei 'vga|3d|display' | sed -n '1p')
fi
printf 'gpu_present=%s\n' "$gpu_present"
printf 'gpu_name=%s\n' "$gpu_name"
`)
}

func ClassifyError(err error, stderr string) Status {
	if err == nil {
		return StatusOnline
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return StatusTimeout
	}
	msg := strings.ToLower(err.Error() + "\n" + stderr)
	switch {
	case strings.Contains(msg, "permission denied"),
		strings.Contains(msg, "authentication failed"),
		strings.Contains(msg, "too many authentication failures"):
		return StatusAuthFailed
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no route to host"),
		strings.Contains(msg, "could not resolve hostname"),
		strings.Contains(msg, "name or service not known"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "operation timed out"),
		isNetworkTimeout(err):
		return StatusOffline
	case strings.Contains(msg, "host key verification failed"):
		return StatusError
	default:
		return StatusError
	}
}

func parseRequiredInt(values map[string]string, key string) (int64, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return 0, fmt.Errorf("missing %s in health probe output", key)
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s in health probe output: %w", key, err)
	}
	return n, nil
}

func parseRequiredFloat(values map[string]string, key string) (float64, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return 0, fmt.Errorf("missing %s in health probe output", key)
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s in health probe output: %w", key, err)
	}
	return n, nil
}

func cleanErrorMessage(err error, stderr string) string {
	msg := strings.TrimSpace(stderr)
	if msg == "" && err != nil {
		msg = strings.TrimSpace(err.Error())
	}
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > 160 {
		msg = msg[:157] + "..."
	}
	return msg
}

func isNetworkTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
