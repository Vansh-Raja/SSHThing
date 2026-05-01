package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	hpkg "github.com/Vansh-Raja/SSHThing/internal/health"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	healthMaxConcurrent = 3
	healthProbeTimeout  = 10 * time.Second
	healthConnectTime   = 5 * time.Second
	healthStaleAfter    = 15 * time.Minute
)

type healthTargetKind string

const (
	healthTargetLocal healthTargetKind = "local"
	healthTargetTeam  healthTargetKind = "team"
)

type healthTarget struct {
	kind         healthTargetKind
	key          string
	hostID       int
	teamHostID   string
	label        string
	conn         ssh.Connection
	authMode     hpkg.AuthMode
	preflightErr error
}

type healthProbeDeps struct {
	teamsClient *teamsclient.Client
	accessToken string
	cfg         config.Config
}

type healthRefreshOptions struct {
	SilentIfEmpty bool
	Source        string
}

func localHealthKey(hostID int) string {
	return fmt.Sprintf("local:%d", hostID)
}

func teamHealthKey(hostID string) string {
	return "team:" + strings.TrimSpace(hostID)
}

func runHealthProbeCmd(runID int, target healthTarget, deps healthProbeDeps) tea.Cmd {
	return func() tea.Msg {
		var result hpkg.Result
		if target.preflightErr != nil {
			result = hpkg.Result{
				TargetKey: target.key,
				Status:    hpkg.StatusAuthFailed,
				CheckedAt: time.Now(),
				Error:     target.preflightErr.Error(),
			}
		} else {
			conn := target.conn
			authMode := target.authMode
			if target.kind == healthTargetTeam {
				var err error
				conn, authMode, err = buildTeamHealthConnection(context.Background(), target, deps)
				if err != nil {
					result = hpkg.Result{
						TargetKey: target.key,
						Status:    hpkg.StatusAuthFailed,
						CheckedAt: time.Now(),
						Error:     err.Error(),
					}
					return hostHealthResultMsg{runID: runID, targetKey: target.key, hostID: target.hostID, result: result}
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), healthProbeTimeout+2*time.Second)
			defer cancel()
			result = hpkg.Probe(ctx, conn, hpkg.ProbeOptions{
				Timeout:        healthProbeTimeout,
				ConnectTimeout: healthConnectTime,
				AuthMode:       authMode,
			})
			result.TargetKey = target.key
		}
		return hostHealthResultMsg{
			runID:     runID,
			targetKey: target.key,
			hostID:    target.hostID,
			result:    result,
		}
	}
}

func (m *Model) loadStoredHostHealth() {
	if m.store == nil {
		return
	}
	if m.healthResults == nil {
		m.healthResults = map[string]hpkg.Result{}
	}
	results, err := m.store.ListHostHealth()
	if err != nil {
		m.err = err
		return
	}
	for hostID, result := range results {
		key := localHealthKey(hostID)
		if existing, ok := m.healthResults[key]; ok && existing.Status == hpkg.StatusChecking {
			continue
		}
		m.healthResults[key] = hostHealthFromDB(result)
	}
}

func (m *Model) beginPersonalHealthRefresh() tea.Cmd {
	return m.beginPersonalHealthRefreshWithOptions(healthRefreshOptions{Source: "manual"})
}

func (m *Model) beginPersonalHealthRefreshWithOptions(opts healthRefreshOptions) tea.Cmd {
	if m.store == nil {
		if !opts.SilentIfEmpty {
			m.err = fmt.Errorf("database is locked")
		}
		return nil
	}
	if len(m.hosts) == 0 {
		if !opts.SilentIfEmpty {
			m.err = fmt.Errorf("no hosts to refresh")
		}
		return nil
	}
	if m.healthResults == nil {
		m.healthResults = map[string]hpkg.Result{}
	}

	m.healthRunID++
	m.healthChecking = true
	m.healthCompleted = 0
	m.healthInFlight = 0
	m.healthTotal = len(m.hosts)
	m.healthQueue = make([]healthTarget, 0, len(m.hosts))
	m.healthActiveKeys = map[string]bool{}
	m.healthProbeDeps = healthProbeDeps{}

	now := time.Now()
	for _, host := range m.hosts {
		target := m.healthTargetForHost(host)
		m.healthQueue = append(m.healthQueue, target)
		m.healthActiveKeys[target.key] = true
		m.healthResults[target.key] = hpkg.Result{
			TargetKey: target.key,
			Status:    hpkg.StatusChecking,
			CheckedAt: now,
		}
	}
	m.err = fmt.Errorf("ℹ checking health: 0/%d complete", m.healthTotal)
	return m.scheduleHealthProbes()
}

func (m *Model) maybeAutoRefreshTeamsHealthOnEnter() tea.Cmd {
	teamID := strings.TrimSpace(m.teamsCurrentTeamID)
	if teamID == "" {
		return nil
	}
	if m.teamsHealthAutoRefreshed == nil {
		m.teamsHealthAutoRefreshed = map[string]bool{}
	}
	if m.teamsHealthAutoRefreshed[teamID] {
		return nil
	}
	m.teamsHealthAutoRefreshed[teamID] = true
	if !m.profileSignedIn() {
		return nil
	}
	return m.beginTeamHealthRefreshWithOptions(healthRefreshOptions{SilentIfEmpty: true, Source: "teams_enter"})
}

func (m *Model) beginTeamHealthRefreshWithOptions(opts healthRefreshOptions) tea.Cmd {
	if m.healthResults == nil {
		m.healthResults = map[string]hpkg.Result{}
	}
	if len(m.teamsItems) == 0 {
		if !opts.SilentIfEmpty {
			m.err = fmt.Errorf("no team hosts to refresh")
		}
		return nil
	}
	accessToken, err := m.teamsAccessToken(context.Background())
	if err != nil {
		if !opts.SilentIfEmpty {
			m.err = fmt.Errorf("team health refresh failed: %v", err)
		}
		return nil
	}

	m.healthRunID++
	m.healthChecking = true
	m.healthCompleted = 0
	m.healthInFlight = 0
	m.healthTotal = len(m.teamsItems)
	m.healthQueue = make([]healthTarget, 0, len(m.teamsItems))
	m.healthActiveKeys = map[string]bool{}
	m.healthProbeDeps = healthProbeDeps{teamsClient: m.teamsClient, accessToken: accessToken, cfg: m.cfg}
	now := time.Now()
	for _, host := range m.teamsItems {
		target := m.healthTargetForTeamHost(host)
		m.healthQueue = append(m.healthQueue, target)
		m.healthActiveKeys[target.key] = true
		m.healthResults[target.key] = hpkg.Result{
			TargetKey: target.key,
			Status:    hpkg.StatusChecking,
			CheckedAt: now,
		}
	}
	m.err = fmt.Errorf("ℹ checking team health: 0/%d complete", m.healthTotal)
	return m.scheduleHealthProbes()
}

func (m *Model) scheduleHealthProbes() tea.Cmd {
	var cmds []tea.Cmd
	for m.healthInFlight < healthMaxConcurrent && len(m.healthQueue) > 0 {
		target := m.healthQueue[0]
		m.healthQueue = m.healthQueue[1:]
		m.healthInFlight++
		cmds = append(cmds, runHealthProbeCmd(m.healthRunID, target, m.healthProbeDeps))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) handleHealthResult(msg hostHealthResultMsg) tea.Cmd {
	if msg.runID != m.healthRunID {
		return nil
	}
	if m.healthResults == nil {
		m.healthResults = map[string]hpkg.Result{}
	}
	if m.healthInFlight > 0 {
		m.healthInFlight--
	}
	m.healthCompleted++
	msg.result.TargetKey = msg.targetKey
	m.healthResults[msg.targetKey] = msg.result
	if msg.hostID > 0 && msg.result.Status != hpkg.StatusChecking && m.store != nil {
		_ = m.store.UpsertHostHealth(hostHealthToDB(msg.hostID, msg.result))
	}

	if m.healthCompleted >= m.healthTotal {
		m.healthChecking = false
		m.healthQueue = nil
		m.healthInFlight = 0
		m.healthProbeDeps = healthProbeDeps{}
		m.err = fmt.Errorf("✓ health refresh complete: %s", m.healthSummary())
		return nil
	}
	m.err = fmt.Errorf("ℹ checking health: %d/%d complete", m.healthCompleted, m.healthTotal)
	return m.scheduleHealthProbes()
}

func (m *Model) healthTargetForHost(host Host) healthTarget {
	conn, authMode, err := m.healthConnectionForHost(host)
	return healthTarget{
		key:          localHealthKey(host.ID),
		kind:         healthTargetLocal,
		hostID:       host.ID,
		label:        hostDisplayName(host),
		conn:         conn,
		authMode:     authMode,
		preflightErr: err,
	}
}

func (m *Model) healthConnectionForHost(host Host) (ssh.Connection, hpkg.AuthMode, error) {
	conn, _, _ := m.buildSSHConn(host)
	authMode := hpkg.AuthModeDefault
	if host.HasKey && host.KeyType != "password" {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			return conn, authMode, fmt.Errorf("failed to decrypt key: %v", err)
		}
		if strings.TrimSpace(secret) == "" {
			return conn, authMode, fmt.Errorf("private key is not configured")
		}
		if err := ssh.ValidatePrivateKey(secret); err != nil {
			return conn, authMode, fmt.Errorf("stored private key is invalid format: %v", err)
		}
		conn.PrivateKey = secret
		authMode = hpkg.AuthModeKey
	}
	if host.KeyType == "password" {
		if !m.cfg.SSH.PasswordAutoLogin {
			return conn, authMode, fmt.Errorf("password auto-login is disabled")
		}
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			return conn, authMode, fmt.Errorf("failed to decrypt password: %v", err)
		}
		if secret == "" {
			return conn, authMode, fmt.Errorf("password is not configured")
		}
		conn.PrivateKey = ""
		conn.Password = secret
		authMode = hpkg.AuthModePassword
	}
	return conn, authMode, nil
}

func (m *Model) healthTargetForTeamHost(host teams.TeamHost) healthTarget {
	key := teamHealthKey(host.ID)
	return healthTarget{
		kind:       healthTargetTeam,
		key:        key,
		teamHostID: host.ID,
		label:      strings.TrimSpace(host.Label),
	}
}

func buildTeamHealthConnection(ctx context.Context, target healthTarget, deps healthProbeDeps) (ssh.Connection, hpkg.AuthMode, error) {
	var conn ssh.Connection
	if deps.teamsClient == nil {
		return conn, hpkg.AuthModeDefault, fmt.Errorf("teams client is not configured")
	}
	connectConfig, err := deps.teamsClient.GetTeamHostConnectConfig(ctx, deps.accessToken, target.teamHostID)
	if err != nil {
		return conn, hpkg.AuthModeDefault, normalizeTeamHealthError(err, target.label)
	}

	conn = ssh.Connection{
		Hostname:            connectConfig.Hostname,
		Username:            connectConfig.Username,
		Port:                connectConfig.Port,
		PasswordBackendUnix: string(deps.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(deps.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    deps.cfg.SSH.KeepAliveSeconds,
		Term:                sshTermFromConfig(deps.cfg),
	}
	switch connectConfig.CredentialType {
	case "private_key":
		if strings.TrimSpace(connectConfig.Secret) == "" {
			return conn, hpkg.AuthModeDefault, fmt.Errorf("private key not configured for %s", connectConfig.Label)
		}
		if err := ssh.ValidatePrivateKey(connectConfig.Secret); err != nil {
			return conn, hpkg.AuthModeDefault, fmt.Errorf("team private key is invalid format: %v", err)
		}
		conn.PrivateKey = connectConfig.Secret
		return conn, hpkg.AuthModeKey, nil
	case "password":
		if !deps.cfg.SSH.PasswordAutoLogin {
			return conn, hpkg.AuthModeDefault, fmt.Errorf("password auto-login is disabled")
		}
		if connectConfig.Secret == "" {
			return conn, hpkg.AuthModeDefault, fmt.Errorf("password not configured for %s", connectConfig.Label)
		}
		conn.Password = connectConfig.Secret
		return conn, hpkg.AuthModePassword, nil
	default:
		return conn, hpkg.AuthModeDefault, nil
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

func sshTermFromConfig(cfg config.Config) string {
	switch cfg.SSH.TermMode {
	case config.TermXterm:
		return "xterm-256color"
	case config.TermCustom:
		return strings.TrimSpace(cfg.SSH.TermCustom)
	default:
		return ""
	}
}

func (m *Model) healthSummary() string {
	online := 0
	offline := 0
	failed := 0
	for _, result := range m.healthResults {
		if len(m.healthActiveKeys) > 0 && !m.healthActiveKeys[result.TargetKey] {
			continue
		}
		if result.CheckedAt.IsZero() || result.Status == hpkg.StatusChecking {
			continue
		}
		switch result.Status {
		case hpkg.StatusOnline:
			online++
		case hpkg.StatusOffline, hpkg.StatusTimeout:
			offline++
		default:
			failed++
		}
	}
	return fmt.Sprintf("%d online, %d offline, %d other", online, offline, failed)
}

func (m Model) healthViewForKey(key string) *ui.HostHealthView {
	result, ok := m.healthResults[key]
	if !ok {
		return nil
	}
	view := &ui.HostHealthView{
		Status:         string(result.Status),
		StatusLabel:    healthStatusLabel(result.Status),
		CheckedLabel:   healthCheckedLabel(result),
		LatencyLabel:   healthLatencyLabel(result.Latency),
		UptimeLabel:    formatDuration(result.Uptime),
		CPULabel:       formatPercent(result.CPUPercent),
		CPUPercent:     int(result.CPUPercent),
		RAMLabel:       formatUsage(result.MemTotalBytes-result.MemAvailableBytes, result.MemTotalBytes),
		DiskLabel:      formatDisk(result.DiskAvailableBytes, result.DiskTotalBytes),
		RAMUsedPct:     percentUsed(result.MemTotalBytes, result.MemAvailableBytes),
		RAMUsedLabel:   formatBytesShort(result.MemTotalBytes - result.MemAvailableBytes),
		RAMTotalLabel:  formatBytesShort(result.MemTotalBytes),
		DiskUsedPct:    percentUsed(result.DiskTotalBytes, result.DiskAvailableBytes),
		DiskFreeLabel:  formatBytesShort(result.DiskAvailableBytes),
		DiskTotalLabel: formatBytesShort(result.DiskTotalBytes),
		GPULabel:       healthGPULabel(result),
		Error:          strings.TrimSpace(result.Error),
		Stale:          !result.CheckedAt.IsZero() && time.Since(result.CheckedAt) > healthStaleAfter,
	}
	view.SummaryLine = healthSummaryLine(result)
	view.ResourceLine = healthResourceLine(result)
	view.SystemLine = healthSystemLine(result)
	return view
}

func healthSummaryLine(result hpkg.Result) string {
	parts := []string{healthStatusLabel(result.Status)}
	checked := healthCheckedLabel(result)
	if checked != "" {
		parts = append(parts, checked)
	}
	if latency := healthLatencyLabel(result.Latency); latency != "n/a" {
		parts = append(parts, latency)
	}
	return strings.Join(parts, " · ")
}

func healthResourceLine(result hpkg.Result) string {
	parts := []string{}
	if result.Status == hpkg.StatusChecking {
		return "checking"
	}
	parts = append(parts, "cpu "+formatPercent(result.CPUPercent))
	if result.MemTotalBytes > 0 {
		parts = append(parts, fmt.Sprintf("ram %s/%s · %d%%", formatBytesShort(result.MemTotalBytes-result.MemAvailableBytes), formatBytesShort(result.MemTotalBytes), percentUsed(result.MemTotalBytes, result.MemAvailableBytes)))
	}
	if result.DiskTotalBytes > 0 {
		parts = append(parts, fmt.Sprintf("disk %s/%s free", formatBytesShort(result.DiskAvailableBytes), formatBytesShort(result.DiskTotalBytes)))
	}
	return strings.Join(parts, " · ")
}

func healthSystemLine(result hpkg.Result) string {
	parts := []string{}
	if uptime := formatDuration(result.Uptime); uptime != "n/a" {
		parts = append(parts, "up "+uptime)
	}
	gpu := healthGPULabel(result)
	if gpu != "" && gpu != "n/a" && gpu != "checking" {
		parts = append(parts, "gpu "+gpu)
	}
	return strings.Join(parts, " · ")
}

func percentUsed(total, available int64) int {
	if total <= 0 {
		return 0
	}
	used := total - available
	if used < 0 {
		used = 0
	}
	pct := int(used * 100 / total)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func hostHealthToDB(hostID int, result hpkg.Result) db.HostHealth {
	return db.HostHealth{
		HostID:             hostID,
		Status:             string(result.Status),
		CheckedAt:          result.CheckedAt,
		LatencyMS:          result.Latency.Milliseconds(),
		UptimeSeconds:      int64(result.Uptime.Seconds()),
		CPUPercent:         result.CPUPercent,
		MemTotalBytes:      result.MemTotalBytes,
		MemAvailableBytes:  result.MemAvailableBytes,
		DiskTotalBytes:     result.DiskTotalBytes,
		DiskAvailableBytes: result.DiskAvailableBytes,
		GPUPresent:         result.GPUPresent,
		GPUName:            result.GPUName,
		Error:              result.Error,
	}
}

func hostHealthFromDB(result db.HostHealth) hpkg.Result {
	return hpkg.Result{
		TargetKey:          localHealthKey(result.HostID),
		Status:             hpkg.Status(result.Status),
		CheckedAt:          result.CheckedAt,
		Latency:            time.Duration(result.LatencyMS) * time.Millisecond,
		Uptime:             time.Duration(result.UptimeSeconds) * time.Second,
		CPUPercent:         result.CPUPercent,
		MemTotalBytes:      result.MemTotalBytes,
		MemAvailableBytes:  result.MemAvailableBytes,
		DiskTotalBytes:     result.DiskTotalBytes,
		DiskAvailableBytes: result.DiskAvailableBytes,
		GPUPresent:         result.GPUPresent,
		GPUName:            result.GPUName,
		Error:              result.Error,
	}
}

func healthStatusLabel(status hpkg.Status) string {
	switch status {
	case hpkg.StatusChecking:
		return "checking"
	case hpkg.StatusOnline:
		return "online"
	case hpkg.StatusOffline:
		return "offline"
	case hpkg.StatusTimeout:
		return "timeout"
	case hpkg.StatusAuthFailed:
		return "auth failed"
	case hpkg.StatusUnsupported:
		return "unsupported"
	case hpkg.StatusError:
		return "error"
	default:
		return "unknown"
	}
}

func healthCheckedLabel(result hpkg.Result) string {
	if result.CheckedAt.IsZero() || result.Status == hpkg.StatusChecking {
		return "checking"
	}
	return ui.FormatTimeAgo(result.CheckedAt)
}

func healthLatencyLabel(v time.Duration) string {
	if v <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%dms", v.Milliseconds())
}

func healthGPULabel(result hpkg.Result) string {
	if result.GPUPresent {
		if strings.TrimSpace(result.GPUName) != "" {
			return result.GPUName
		}
		return "yes"
	}
	if result.Status == hpkg.StatusChecking {
		return "checking"
	}
	if result.Status != hpkg.StatusOnline {
		return "n/a"
	}
	return "no"
}

func formatPercent(v float64) string {
	if v <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", v)
}

func formatUsage(used, total int64) string {
	if total <= 0 {
		return "n/a"
	}
	if used < 0 {
		used = 0
	}
	return fmt.Sprintf("%s / %s (%d%%)", formatBytes(used), formatBytes(total), used*100/total)
}

func formatDisk(available, total int64) string {
	if total <= 0 {
		return "n/a"
	}
	if available < 0 {
		available = 0
	}
	return fmt.Sprintf("%s free / %s", formatBytes(available), formatBytes(total))
}

func formatDuration(v time.Duration) string {
	if v <= 0 {
		return "n/a"
	}
	days := int(v.Hours()) / 24
	hours := int(v.Hours()) % 24
	minutes := int(v.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatBytes(v int64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div := int64(unit)
	exp := 0
	for n := v / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(v)/float64(div), "KMGTPE"[exp])
}

func formatBytesShort(v int64) string {
	if v <= 0 {
		return "0 B"
	}
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div := int64(unit)
	exp := 0
	for n := v / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(v)/float64(div), "KMGTPE"[exp])
}
