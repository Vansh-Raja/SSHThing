package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/health"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
)

// HealthService provides host health probe operations.
type HealthService struct {
	Vault       *Vault
	CfgStore    *CfgStore
	TeamsClient *teamsclient.Client
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
// Supports both personal hosts (integer IDs from local DB) and team hosts
// (string IDs — fetches connect config from cloud API).
func (hs *HealthService) Probe(ctx context.Context, hostID string) (*HealthResult, error) {
	// Try to parse as int — personal host.
	intID, err := strconv.Atoi(hostID)
	if err == nil {
		return hs.probePersonal(ctx, intID, hostID)
	}
	// String ID — team host.
	return hs.probeTeam(ctx, hostID)
}

func (hs *HealthService) probePersonal(ctx context.Context, intID int, hostID string) (*HealthResult, error) {
	store := hs.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
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
	hr := resultToHealthResult(hostID, result)

	// Persist to DB — best effort.
	_ = store.UpsertHostHealth(dbhealthFromResult(intID, result))

	return hr, nil
}

func (hs *HealthService) probeTeam(ctx context.Context, hostID string) (*HealthResult, error) {
	if hs.TeamsClient == nil || !hs.TeamsClient.Enabled() {
		return nil, fmt.Errorf("teams client not configured")
	}

	accessToken, err := accessToken(ctx, hs.TeamsClient)
	if err != nil {
		return nil, fmt.Errorf("team auth: %w", err)
	}

	connectConfig, err := hs.TeamsClient.GetTeamHostConnectConfig(ctx, accessToken, hostID)
	if err != nil {
		return nil, normalizeTeamHealthError(err, hostID)
	}

	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	if hs.CfgStore != nil {
		cfg := hs.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
	}

	conn := ssh.Connection{
		Hostname:         connectConfig.Hostname,
		Username:         connectConfig.Username,
		Port:             connectConfig.Port,
		HostKeyPolicy:    hostKeyPolicy,
		KeepAliveSeconds: keepAlive,
	}

	var authMode health.AuthMode
	switch connectConfig.CredentialType {
	case "private_key":
		if strings.TrimSpace(connectConfig.Secret) == "" {
			return nil, fmt.Errorf("private key not configured for %s", connectConfig.Label)
		}
		if err := ssh.ValidatePrivateKey(connectConfig.Secret); err != nil {
			return nil, fmt.Errorf("team private key is invalid format: %v", err)
		}
		conn.PrivateKey = connectConfig.Secret
		authMode = health.AuthModeKey
	case "password":
		if connectConfig.Secret == "" {
			return nil, fmt.Errorf("password not configured for %s", connectConfig.Label)
		}
		conn.Password = connectConfig.Secret
		authMode = health.AuthModePassword
	}

	result := health.Probe(ctx, conn, health.ProbeOptions{AuthMode: authMode})
	return resultToHealthResult(hostID, result), nil
}

func resultToHealthResult(hostID string, result health.Result) *HealthResult {
	return &HealthResult{
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
}

func normalizeTeamHealthError(err error, label string) error {
	switch err.Error() {
	case "personal_credential_not_configured":
		return fmt.Errorf("personal credential not configured for %s", strings.TrimSpace(label))
	case "shared_credential_not_configured":
		return fmt.Errorf("shared credential not configured for %s", strings.TrimSpace(label))
	default:
		return err
	}
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
