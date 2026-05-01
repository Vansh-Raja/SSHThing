package health

import (
	"context"
	"errors"
	"testing"
)

func TestParseProbeOutputValid(t *testing.T) {
	stats, err := ParseProbeOutput(`
status=ok
uptime_seconds=3600
cpu_percent=17
mem_total_bytes=8589934592
mem_available_bytes=4294967296
disk_total_bytes=107374182400
disk_available_bytes=53687091200
gpu_present=true
gpu_name=NVIDIA RTX 4090
`)
	if err != nil {
		t.Fatalf("ParseProbeOutput returned error: %v", err)
	}
	if stats.Status != StatusOnline {
		t.Fatalf("expected online status, got %q", stats.Status)
	}
	if stats.UptimeSeconds != 3600 || stats.CPUPercent != 17 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if !stats.GPUPresent || stats.GPUName != "NVIDIA RTX 4090" {
		t.Fatalf("unexpected gpu stats: %+v", stats)
	}
}

func TestParseProbeOutputUnsupported(t *testing.T) {
	stats, err := ParseProbeOutput(`
status=unsupported
error=linux /proc metrics are unavailable
`)
	if err != nil {
		t.Fatalf("ParseProbeOutput returned error: %v", err)
	}
	if stats.Status != StatusUnsupported {
		t.Fatalf("expected unsupported, got %q", stats.Status)
	}
	if stats.Error == "" {
		t.Fatalf("expected unsupported error message")
	}
}

func TestParseProbeOutputRejectsMalformedNumericValue(t *testing.T) {
	_, err := ParseProbeOutput(`
status=ok
uptime_seconds=bad
cpu_percent=17
mem_total_bytes=1
mem_available_bytes=1
disk_total_bytes=1
disk_available_bytes=1
gpu_present=false
`)
	if err == nil {
		t.Fatalf("expected malformed uptime to fail")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		stderr string
		want   Status
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: StatusTimeout},
		{name: "auth", err: errors.New("exit status 255"), stderr: "Permission denied (publickey).", want: StatusAuthFailed},
		{name: "offline", err: errors.New("exit status 255"), stderr: "ssh: connect to host example.com port 22: Connection refused", want: StatusOffline},
		{name: "host key", err: errors.New("exit status 255"), stderr: "Host key verification failed.", want: StatusError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyError(tt.err, tt.stderr); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
