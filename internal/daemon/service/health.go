package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/health"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// HealthService provides host health probe operations.
type HealthService struct {
	Vault    *Vault
	CfgStore *CfgStore
}

// HealthResult is the wire-safe health result.
type HealthResult struct {
	HostID    string    `json:"hostId"`
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checkedAt"`
	LatencyMs int64     `json:"latencyMs"`
	Error     string    `json:"error,omitempty"`
	// Resource fields populated when available.
	UptimeSecs     int64   `json:"uptimeSecs,omitempty"`
	CPUPercent     float64 `json:"cpuPercent,omitempty"`
	MemTotalBytes  int64   `json:"memTotalBytes,omitempty"`
	MemAvailBytes  int64   `json:"memAvailBytes,omitempty"`
	DiskTotalBytes int64   `json:"diskTotalBytes,omitempty"`
	DiskAvailBytes int64   `json:"diskAvailBytes,omitempty"`
	GPUPresent     bool    `json:"gpuPresent,omitempty"`
	GPUName        string  `json:"gpuName,omitempty"`
}

// Probe runs a health probe against a single host.
// Mirrors: internal/app/backend.go beginPersonalHealthRefreshWithOptions.
func (hs *HealthService) Probe(ctx context.Context, hostID string) (*HealthResult, error) {
	store := hs.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	intID, err := strconv.Atoi(hostID)
	if err != nil {
		return nil, fmt.Errorf("invalid host id %q: %w", hostID, err)
	}
	model, err := store.GetHostByID(intID)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	if hs.CfgStore != nil {
		cfg := hs.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
	}

	conn := ssh.Connection{
		Hostname:         model.Hostname,
		Username:         model.Username,
		Port:             model.Port,
		HostKeyPolicy:    hostKeyPolicy,
		KeepAliveSeconds: keepAlive,
	}
	if model.KeyType == "password" {
		conn.Password = secret
	} else if secret != "" {
		conn.PrivateKey = secret
	}

	result := health.Probe(ctx, conn, health.ProbeOptions{})
	hr := &HealthResult{
		HostID:         hostID,
		Status:         string(result.Status),
		CheckedAt:      result.CheckedAt,
		LatencyMs:      result.Latency.Milliseconds(),
		Error:          result.Error,
		UptimeSecs:     int64(result.Uptime.Seconds()),
		CPUPercent:     result.CPUPercent,
		MemTotalBytes:  result.MemTotalBytes,
		MemAvailBytes:  result.MemAvailableBytes,
		DiskTotalBytes: result.DiskTotalBytes,
		DiskAvailBytes: result.DiskAvailableBytes,
		GPUPresent:     result.GPUPresent,
		GPUName:        result.GPUName,
	}

	// Persist to DB — best effort.
	_ = store.UpsertHostHealth(dbhealthFromResult(intID, result))

	return hr, nil
}

func dbhealthFromResult(hostID int, r health.Result) db.HostHealth {
	return db.HostHealth{
		HostID:             hostID,
		Status:             string(r.Status),
		CheckedAt:          r.CheckedAt,
		LatencyMS:          r.Latency.Milliseconds(),
		UptimeSeconds:      int64(r.Uptime.Seconds()),
		CPUPercent:         r.CPUPercent,
		MemTotalBytes:      r.MemTotalBytes,
		MemAvailableBytes:  r.MemAvailableBytes,
		DiskTotalBytes:     r.DiskTotalBytes,
		DiskAvailableBytes: r.DiskAvailableBytes,
		GPUPresent:         r.GPUPresent,
		GPUName:            r.GPUName,
		Error:              r.Error,
	}
}

// List returns the last stored health check result for every host.
func (hs *HealthService) List() ([]HealthResult, error) {
	store := hs.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	rowMap, err := store.ListHostHealth()
	if err != nil {
		return nil, fmt.Errorf("list host health: %w", err)
	}
	out := make([]HealthResult, 0, len(rowMap))
	for _, r := range rowMap {
		out = append(out, HealthResult{
			HostID:         strconv.Itoa(r.HostID),
			Status:         r.Status,
			CheckedAt:      r.CheckedAt,
			LatencyMs:      r.LatencyMS,
			Error:          r.Error,
			UptimeSecs:     r.UptimeSeconds,
			CPUPercent:     r.CPUPercent,
			MemTotalBytes:  r.MemTotalBytes,
			MemAvailBytes:  r.MemAvailableBytes,
			DiskTotalBytes: r.DiskTotalBytes,
			DiskAvailBytes: r.DiskAvailableBytes,
			GPUPresent:     r.GPUPresent,
			GPUName:        r.GPUName,
		})
	}
	return out, nil
}
