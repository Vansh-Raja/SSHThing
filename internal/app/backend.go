package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/mount"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Host loading ──────────────────────────────────────────────────────

func (m *Model) loadHosts() {
	if m.store == nil {
		return
	}

	dbHosts, err := m.store.GetHosts()
	if err != nil {
		m.err = err
		return
	}

	m.hosts = make([]Host, len(dbHosts))
	for i, h := range dbHosts {
		hasKey := h.KeyData != ""
		if h.KeyType == "password" {
			hasKey = true
		}
		label := strings.TrimSpace(h.Label)
		m.hosts[i] = Host{
			ID:            h.ID,
			Label:         label,
			GroupName:     strings.TrimSpace(h.GroupName),
			Tags:          append([]string(nil), h.Tags...),
			Hostname:      h.Hostname,
			Username:      h.Username,
			Port:          h.Port,
			HasKey:        hasKey,
			KeyType:       h.KeyType,
			CreatedAt:     h.CreatedAt,
			LastConnected: h.LastConnected,
		}
	}

	m.loadGroups()
	m.rebuildListItems()
	m.syncTokenLabelsWithHosts()
}

func (m *Model) loadGroups() {
	if m.store == nil {
		return
	}
	groups, err := m.store.GetGroups()
	if err != nil {
		m.err = err
		return
	}
	m.groups = groups
}

func hostDisplayName(h Host) string {
	d := strings.TrimSpace(h.Label)
	if d == "" {
		d = strings.TrimSpace(h.Hostname)
	}
	return d
}

func hostVirtualGroupTag(h Host) string {
	if strings.TrimSpace(h.GroupName) == "" {
		return ""
	}
	return db.NormalizeTagToken(h.GroupName)
}

func hostSearchTags(h Host) []string {
	tags := db.NormalizeTags(h.Tags)
	if gt := hostVirtualGroupTag(h); gt != "" {
		tags = db.NormalizeTags(append(tags, gt))
	}
	return tags
}

func hostSearchCorpus(h Host) string {
	parts := []string{
		hostDisplayName(h),
		h.Hostname,
		h.Username,
		h.GroupName,
	}
	for _, tag := range hostSearchTags(h) {
		parts = append(parts, tag, "#"+tag)
	}
	return strings.Join(parts, " ")
}

func teamHostSearchCorpus(h teams.TeamHost) string {
	parts := []string{
		strings.TrimSpace(h.Label),
		h.Hostname,
		h.Username,
		h.Group,
	}
	for _, tag := range h.Tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		parts = append(parts, tag, "#"+tag)
	}
	return strings.Join(parts, " ")
}

// ── List building ─────────────────────────────────────────────────────

func (m *Model) rebuildListItems() {
	counts := make(map[string]int)
	hostsByGroup := make(map[string][]Host)
	for _, h := range m.hosts {
		g := strings.TrimSpace(h.GroupName)
		hostsByGroup[g] = append(hostsByGroup[g], h)
		counts[g]++
	}

	groupSet := make(map[string]struct{})
	for _, g := range m.groups {
		name := strings.TrimSpace(g)
		if name == "" {
			continue
		}
		groupSet[name] = struct{}{}
	}
	for g := range hostsByGroup {
		if g == "" {
			continue
		}
		groupSet[g] = struct{}{}
	}

	groups := make([]string, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i]) < strings.ToLower(groups[j])
	})

	for g := range hostsByGroup {
		sort.Slice(hostsByGroup[g], func(i, j int) bool {
			a := strings.ToLower(hostDisplayName(hostsByGroup[g][i]))
			b := strings.ToLower(hostDisplayName(hostsByGroup[g][j]))
			if a == b {
				return strings.ToLower(hostsByGroup[g][i].Hostname) < strings.ToLower(hostsByGroup[g][j].Hostname)
			}
			return a < b
		})
	}

	items := make([]ListItem, 0, len(m.hosts)+len(groups)+2)
	for _, g := range groups {
		items = append(items, ListItem{Kind: ListItemGroup, GroupName: g, Count: counts[g]})
		if !m.collapsed[g] {
			for _, h := range hostsByGroup[g] {
				items = append(items, ListItem{Kind: ListItemHost, GroupName: g, Host: h})
			}
		}
	}

	if len(hostsByGroup[""]) > 0 {
		items = append(items, ListItem{Kind: ListItemGroup, GroupName: "Ungrouped", Count: counts[""]})
		if !m.collapsed["Ungrouped"] {
			for _, h := range hostsByGroup[""] {
				items = append(items, ListItem{Kind: ListItemHost, GroupName: "", Host: h})
			}
		}
	}

	items = append(items, ListItem{Kind: ListItemNewGroup})
	m.listItems = items
	if len(m.listItems) == 0 {
		m.selectedIdx = 0
	} else if m.selectedIdx >= len(m.listItems) {
		m.selectedIdx = len(m.listItems) - 1
	}
}

// ── Selection helpers ─────────────────────────────────────────────────

func (m *Model) selectedListItem() (ListItem, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.listItems) {
		return ListItem{}, false
	}
	return m.listItems[m.selectedIdx], true
}

func (m *Model) selectedHost() (Host, bool) {
	item, ok := m.selectedListItem()
	if !ok || item.Kind != ListItemHost {
		return Host{}, false
	}
	return item.Host, true
}

func (m *Model) selectedGroup() (string, bool) {
	item, ok := m.selectedListItem()
	if !ok || item.Kind != ListItemGroup {
		return "", false
	}
	if item.GroupName == "Ungrouped" {
		return "", true
	}
	return item.GroupName, true
}

func (m *Model) hostCountForGroup(groupName string) int {
	count := 0
	norm := strings.TrimSpace(groupName)
	if strings.EqualFold(norm, "Ungrouped") {
		norm = ""
	}
	for _, h := range m.hosts {
		hg := strings.TrimSpace(h.GroupName)
		if norm == "" {
			if hg == "" {
				count++
			}
			continue
		}
		if strings.EqualFold(hg, norm) {
			count++
		}
	}
	return count
}

func (m *Model) selectGroupInList(groupName string) {
	m.rebuildListItems()
	for i, it := range m.listItems {
		if it.Kind != ListItemGroup {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(it.GroupName), strings.TrimSpace(groupName)) {
			m.selectedIdx = i
			return
		}
	}
}

func (m *Model) selectedSpotlightItem() (SpotlightItem, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.spotlightItems) {
		return SpotlightItem{}, false
	}
	return m.spotlightItems[m.selectedIdx], true
}

// ── Mount helpers ─────────────────────────────────────────────────────

func (m *Model) restoreMountsFromDB() {
	if m.store == nil || m.mountManager == nil {
		return
	}
	states, err := m.store.GetMountStates()
	if err != nil {
		return
	}

	byID := make(map[int]Host, len(m.hosts))
	for _, h := range m.hosts {
		byID[h.ID] = h
	}

	var toRestore []mount.Mount
	for _, st := range states {
		ok, err := mount.IsMounted(st.LocalPath)
		if err != nil {
			continue
		}
		if !ok {
			_ = m.store.DeleteMountState(st.HostID)
			continue
		}
		host, _ := byID[st.HostID]
		hostname := strings.TrimSpace(host.Hostname)
		if hostname == "" {
			hostname = fmt.Sprintf("host_%d", st.HostID)
		}
		toRestore = append(toRestore, mount.Mount{
			HostID:     st.HostID,
			Hostname:   hostname,
			LocalPath:  st.LocalPath,
			RemotePath: st.RemotePath,
		})
	}

	if len(toRestore) > 0 {
		m.mountManager.RestoreMounted(toRestore)
	}
}

// ── Sync helpers ──────────────────────────────────────────────────────

func (m *Model) initSyncManager() {
	if m.store == nil {
		return
	}
	syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, m.masterPassword)
	if err != nil {
		m.err = fmt.Errorf("sync init failed: %v", err)
		return
	}
	m.syncManager = syncMgr
}

// ── SSH connection ────────────────────────────────────────────────────

func (m *Model) buildSSHConn(host Host) (ssh.Connection, string, string) {
	var privateKey string
	var password string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err == nil {
			if err := ssh.ValidatePrivateKey(key); err == nil {
				privateKey = key
			}
		}
	}
	if host.KeyType == "password" && m.cfg.SSH.PasswordAutoLogin {
		secret, err := m.store.GetHostSecret(host.ID)
		if err == nil {
			password = secret
		}
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}

	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PrivateKey:          privateKey,
		Password:            password,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}
	return conn, privateKey, password
}

func (m *Model) currentSSHSessionTerm() string {
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		return "xterm-256color"
	case config.TermCustom:
		return strings.TrimSpace(m.cfg.SSH.TermCustom)
	default:
		return ""
	}
}

func (m Model) connectToHost(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false

	var privateKey string
	var password string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}
	if host.KeyType == "password" && m.cfg.SSH.PasswordAutoLogin {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt password: %v", err)
			return m, nil
		}
		password = secret
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PrivateKey:          privateKey,
		Password:            password,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}

	cmd, tempKey, err := ssh.Connect(conn)
	if err != nil {
		m.err = fmt.Errorf("failed to prepare SSH connection: %v", err)
		return m, nil
	}

	if m.store != nil {
		m.store.UpdateLastConnected(host.ID)
	}

	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			if tempKey != nil {
				tempKey.Cleanup()
			}
			return sshFinishedMsg{err: err, hostname: host.Hostname, proto: "SSH", keyType: host.KeyType}
		}),
	)
}

func (m Model) connectToTeamHost(host teams.TeamHost) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false

	ctx := context.Background()
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		m.err = err
		return m, nil
	}

	connectConfig, err := m.teamsClient.GetTeamHostConnectConfig(ctx, accessToken, host.ID)
	if err != nil {
		switch err.Error() {
		case "personal_credential_not_configured":
			m.err = fmt.Errorf("personal credential not configured for %s. Add it in the browser teams UI first.", hostDisplayName(Host{Label: host.Label, Hostname: host.Hostname}))
		case "shared_credential_not_configured":
			m.err = fmt.Errorf("shared credential not configured for %s", hostDisplayName(Host{Label: host.Label, Hostname: host.Hostname}))
		default:
			m.err = err
		}
		return m, nil
	}

	conn := ssh.Connection{
		Hostname:            connectConfig.Hostname,
		Username:            connectConfig.Username,
		Port:                connectConfig.Port,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                m.currentSSHSessionTerm(),
	}

	switch connectConfig.CredentialType {
	case "private_key":
		if strings.TrimSpace(connectConfig.Secret) == "" {
			m.err = fmt.Errorf("private key not configured for %s", connectConfig.Label)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(connectConfig.Secret); err != nil {
			m.err = fmt.Errorf("team private key is invalid format: %v", err)
			return m, nil
		}
		conn.PrivateKey = connectConfig.Secret
	case "password":
		if m.cfg.SSH.PasswordAutoLogin {
			conn.Password = connectConfig.Secret
		}
	}

	cmd, tempKey, err := ssh.Connect(conn)
	if err != nil {
		m.err = fmt.Errorf("failed to prepare SSH connection: %v", err)
		return m, nil
	}

	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			if tempKey != nil {
				tempKey.Cleanup()
			}
			return sshFinishedMsg{
				err:      err,
				hostname: connectConfig.Hostname,
				proto:    "SSH",
				keyType:  connectConfig.CredentialType,
			}
		}),
	)
}

func (m Model) connectToHostSFTP(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false

	var privateKey string
	var password string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}
	if host.KeyType == "password" && m.cfg.SSH.PasswordAutoLogin {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt password: %v", err)
			return m, nil
		}
		password = secret
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PrivateKey:          privateKey,
		Password:            password,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}

	cmd, tempKey, err := ssh.ConnectSFTP(conn)
	if err != nil {
		m.err = fmt.Errorf("failed to prepare SFTP session: %v", err)
		return m, nil
	}

	if m.store != nil {
		m.store.UpdateLastConnected(host.ID)
	}

	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			if tempKey != nil {
				tempKey.Cleanup()
			}
			return sshFinishedMsg{err: err, hostname: host.Hostname, proto: "SFTP", keyType: host.KeyType}
		}),
	)
}

func (m Model) handleMountEnter(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false

	if m.mountManager == nil {
		m.armedMount = false
		m.armedUnmount = false
		m.err = fmt.Errorf("\u26A0 mount manager not initialized")
		return m, nil
	}

	// Unmount flow
	if m.armedUnmount {
		m.armedUnmount = false
		cmd, localPath, err := m.mountManager.PrepareUnmount(host.ID)
		if err != nil {
			m.err = err
			return m, nil
		}
		_ = localPath
		return m, tea.Sequence(
			tea.ShowCursor,
			tea.ExecProcess(cmd, func(err error) tea.Msg {
				return mountFinishedMsg{action: "unmount", hostID: host.ID, local: localPath, err: err}
			}),
		)
	}

	// Mount flow
	m.armedMount = false

	var privateKey string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}

	remotePath := m.cfg.Mount.DefaultRemotePath
	display := strings.TrimSpace(host.Label)
	if display == "" {
		display = host.Hostname
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	prep, err := m.mountManager.PrepareMount(host.ID, ssh.Connection{
		Hostname:         host.Hostname,
		Username:         host.Username,
		Port:             host.Port,
		PrivateKey:       privateKey,
		HostKeyPolicy:    string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds: m.cfg.SSH.KeepAliveSeconds,
		Term:             term,
	}, remotePath, display, m.cfg.Mount.LocalMountPath)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.pendingMount = prep
	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(prep.Cmd(), func(err error) tea.Msg {
			return mountFinishedMsg{action: "mount", hostID: host.ID, local: prep.LocalPath, err: err, stderr: prep.Stderr()}
		}),
	)
}

// ── Search / spotlight ────────────────────────────────────────────────

func fuzzyScore(query, candidate string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	c := strings.ToLower(candidate)
	if q == "" {
		return 0, true
	}
	if strings.Contains(c, q) {
		return 100 + len(q)*4, true
	}
	qi := 0
	score := 0
	streak := 0
	lastMatch := -2
	for i := 0; i < len(c) && qi < len(q); i++ {
		if c[i] != q[qi] {
			continue
		}
		if i == 0 || c[i-1] == ' ' || c[i-1] == '-' || c[i-1] == '_' || c[i-1] == '.' || c[i-1] == '/' {
			score += 8
		}
		if i == lastMatch+1 {
			streak++
			score += 4 + streak
		} else {
			streak = 0
			score += 2
		}
		lastMatch = i
		qi++
	}
	if qi != len(q) {
		return 0, false
	}
	score += max(0, 20-(len(c)-len(q)))
	return score, true
}

func (m Model) buildSpotlightItems(query string) []SpotlightItem {
	query = strings.TrimSpace(query)
	if m.teamsImportMode {
		if query == "" {
			out := make([]SpotlightItem, 0, min(8, len(m.hosts)))
			for _, h := range m.hosts {
				out = append(out, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName})
				if len(out) >= 8 {
					break
				}
			}
			return out
		}

		out := make([]SpotlightItem, 0, 16)
		for _, h := range m.hosts {
			score, ok := fuzzyScore(query, hostSearchCorpus(h))
			if !ok {
				continue
			}
			out = append(out, SpotlightItem{
				Kind:      SpotlightItemHost,
				Host:      h,
				GroupName: h.GroupName,
				Score:     score,
			})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
		if len(out) > 16 {
			out = out[:16]
		}
		return out
	}

	if m.appMode == appModeTeams {
		if strings.HasPrefix(query, ">") {
			return m.teamCommandItems(query)
		}
		if query == "" {
			out := make([]SpotlightItem, 0, min(8, len(m.teamsItems)))
			for _, h := range m.teamsItems {
				out = append(out, SpotlightItem{Kind: SpotlightItemHost, TeamHost: h, GroupName: h.Group})
				if len(out) >= 8 {
					break
				}
			}
			return out
		}

		out := make([]SpotlightItem, 0, 12)
		for _, h := range m.teamsItems {
			score, ok := fuzzyScore(query, teamHostSearchCorpus(h))
			if !ok {
				continue
			}
			out = append(out, SpotlightItem{
				Kind:      SpotlightItemHost,
				TeamHost:  h,
				GroupName: h.Group,
				Score:     score,
			})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
		if len(out) > 16 {
			out = out[:16]
		}
		return out
	}

	if query == "" {
		out := make([]SpotlightItem, 0, min(8, len(m.hosts)))
		for _, h := range m.hosts {
			out = append(out, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName})
			if len(out) >= 8 {
				break
			}
		}
		return out
	}

	type scoredGroup struct {
		name  string
		score int
	}
	var groupScores []scoredGroup
	for _, it := range m.listItems {
		if it.Kind != ListItemGroup {
			continue
		}
		name := it.GroupName
		if score, ok := fuzzyScore(query, name); ok {
			groupScores = append(groupScores, scoredGroup{name: it.GroupName, score: score})
		}
	}
	sort.Slice(groupScores, func(i, j int) bool { return groupScores[i].score > groupScores[j].score })

	seenHost := map[int]bool{}
	out := make([]SpotlightItem, 0, 12)
	for i, g := range groupScores {
		if i >= 3 {
			break
		}
		out = append(out, SpotlightItem{Kind: SpotlightItemGroup, GroupName: g.name, Score: g.score})
		groupHosts := make([]SpotlightItem, 0, 4)
		for _, h := range m.hosts {
			hg := strings.TrimSpace(h.GroupName)
			if g.name == "Ungrouped" {
				if hg != "" {
					continue
				}
			} else if !strings.EqualFold(hg, g.name) {
				continue
			}
			score, ok := fuzzyScore(query, hostSearchCorpus(h))
			if !ok {
				score = 1
			}
			groupHosts = append(groupHosts, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName, Score: score, Indent: 1})
		}
		sort.Slice(groupHosts, func(i, j int) bool { return groupHosts[i].Score > groupHosts[j].Score })
		for i := 0; i < len(groupHosts) && i < 3; i++ {
			if seenHost[groupHosts[i].Host.ID] {
				continue
			}
			seenHost[groupHosts[i].Host.ID] = true
			out = append(out, groupHosts[i])
		}
	}

	var direct []SpotlightItem
	for _, h := range m.hosts {
		if seenHost[h.ID] {
			continue
		}
		score, ok := fuzzyScore(query, hostSearchCorpus(h))
		if !ok {
			continue
		}
		direct = append(direct, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName, Score: score})
	}
	sort.Slice(direct, func(i, j int) bool { return direct[i].Score > direct[j].Score })
	for i := 0; i < len(direct) && len(out) < 16; i++ {
		out = append(out, direct[i])
	}
	return out
}

// ── Token helpers ─────────────────────────────────────────────────────

func (m *Model) syncTokenLabelsWithHosts() {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return
	}
	labels := make(map[int]string, len(m.hosts))
	for _, h := range m.hosts {
		labels[h.ID] = hostDisplayName(h)
	}
	if !vault.SyncHostLabels(labels) {
		return
	}
	_ = authtoken.SaveVault(vault)
}

func (m *Model) loadTokenSummaries() {
	vault, err := authtoken.LoadVault()
	if err != nil {
		m.err = fmt.Errorf("failed to load token vault: %v", err)
		m.tokenSummaries = nil
		m.tokenIdx = 0
		return
	}
	m.tokenSummaries = vault.ListSummaries()
	if len(m.tokenSummaries) == 0 {
		m.tokenIdx = 0
		return
	}
	if m.tokenIdx < 0 {
		m.tokenIdx = 0
	}
	if m.tokenIdx >= len(m.tokenSummaries) {
		m.tokenIdx = len(m.tokenSummaries) - 1
	}
}

func (m Model) selectedTokenHostGrants() ([]authtoken.HostGrant, error) {
	grants := make([]authtoken.HostGrant, 0, len(m.tokenHostPick))
	for _, h := range m.hosts {
		if !m.tokenHostPick[h.ID] {
			continue
		}
		if !h.HasKey {
			return nil, fmt.Errorf("host '%s' has no stored auth secret", hostDisplayName(h))
		}
		secret, err := m.store.GetHostSecret(h.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret for '%s': %v", hostDisplayName(h), err)
		}
		if strings.TrimSpace(secret) == "" {
			return nil, fmt.Errorf("host '%s' has no usable auth secret", hostDisplayName(h))
		}
		grants = append(grants, authtoken.HostGrant{HostID: h.ID, DisplayLabel: hostDisplayName(h)})
	}
	if len(grants) == 0 {
		return nil, fmt.Errorf("no eligible hosts selected")
	}
	return grants, nil
}

func (m Model) tokenManagerHosts() []ui.TokenViewItem {
	out := make([]ui.TokenViewItem, 0, len(m.hosts))
	for _, h := range m.hosts {
		out = append(out, ui.TokenViewItem{
			Name:    hostDisplayName(h),
			Scope:   fmt.Sprintf("%s@%s:%d", h.Username, h.Hostname, h.Port),
			Created: "",
			LastUse: "",
		})
	}
	return out
}

func (m Model) tokenManagerTokenRows() []ui.TokenViewItem {
	out := make([]ui.TokenViewItem, 0, len(m.tokenSummaries))
	for _, t := range m.tokenSummaries {
		scope := "active"
		if t.RevokedAt != nil {
			scope = "revoked"
		} else if !t.Usable {
			scope = "inactive"
		} else if t.Legacy {
			scope = "legacy"
		}
		lastUsed := "never"
		if t.LastUsedAt != nil {
			lastUsed = t.LastUsedAt.Local().Format("2006-01-02 15:04")
		}
		name := strings.TrimSpace(t.Name)
		if name == "" {
			name = t.TokenID
		}
		out = append(out, ui.TokenViewItem{
			Name:    name,
			Scope:   scope,
			Created: t.CreatedAt.Local().Format("2006-01-02"),
			LastUse: lastUsed,
		})
	}
	return out
}

func (m Model) buildTokenHostItems() []ui.TokenHostItem {
	out := make([]ui.TokenHostItem, 0, len(m.hosts))
	for _, h := range m.hosts {
		out = append(out, ui.TokenHostItem{
			ID:       h.ID,
			Label:    hostDisplayName(h),
			Detail:   fmt.Sprintf("%s@%s:%d", h.Username, h.Hostname, h.Port),
			Selected: m.tokenHostPick[h.ID],
		})
	}
	return out
}

// ── Settings helpers ──────────────────────────────────────────────────

func (m *Model) buildSettingsItems() []ui.SettingsItem {
	boolVal := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}

	if m.appMode == appModeTeams {
		return []ui.SettingsItem{
			{Category: "ui", Label: "wrap labels", Value: boolVal(m.cfg.TeamsUI.WrapLabels), Kind: 0},
			{Category: "ui", Label: "theme", Value: m.cfg.TeamsUI.Theme, Kind: 1, Options: themeNames(), OptIdx: themeIdx(m.cfg.TeamsUI.Theme)},
			{Category: "ui", Label: "icon set", Value: m.cfg.TeamsUI.IconSet, Kind: 1, Options: iconSetNames(), OptIdx: iconSetIdx(m.cfg.TeamsUI.IconSet)},
		}
	}

	items := []ui.SettingsItem{
		// UI
		{Category: "ui", Label: "vim mode", Value: boolVal(m.cfg.UI.VimMode), Kind: 0},
		{Category: "ui", Label: "show icons", Value: boolVal(m.cfg.UI.ShowIcons), Kind: 0},
		{Category: "ui", Label: "wrap labels", Value: boolVal(m.cfg.UI.WrapLabels), Kind: 0},
		{Category: "ui", Label: "theme", Value: m.cfg.UI.Theme, Kind: 1, Options: themeNames(), OptIdx: themeIdx(m.cfg.UI.Theme)},
		{Category: "ui", Label: "icon set", Value: m.cfg.UI.IconSet, Kind: 1, Options: iconSetNames(), OptIdx: iconSetIdx(m.cfg.UI.IconSet)},
		// SSH
		{Category: "ssh", Label: "host key policy", Value: string(m.cfg.SSH.HostKeyPolicy), Kind: 1, Options: []string{"accept-new", "strict", "off"}},
		{Category: "ssh", Label: "keepalive seconds", Value: fmt.Sprintf("%d", m.cfg.SSH.KeepAliveSeconds), Kind: 2},
		{Category: "ssh", Label: "TERM mode", Value: string(m.cfg.SSH.TermMode), Kind: 1, Options: []string{"auto", "xterm-256color", "custom"}},
		{Category: "ssh", Label: "TERM custom", Value: m.cfg.SSH.TermCustom, Kind: 2, Disabled: m.cfg.SSH.TermMode != config.TermCustom},
		{Category: "ssh", Label: "password auto-login", Value: boolVal(m.cfg.SSH.PasswordAutoLogin), Kind: 0},
		{Category: "ssh", Label: "password backend (unix)", Value: string(m.cfg.SSH.PasswordBackendUnix), Kind: 1, Options: []string{"sshpass_first", "askpass_first"}, Disabled: runtime.GOOS == "windows" || !m.cfg.SSH.PasswordAutoLogin},
		// Mount
		{Category: "mount", Label: "enable mounts", Value: boolVal(m.cfg.Mount.Enabled), Kind: 0},
		{Category: "mount", Label: "default remote path", Value: m.cfg.Mount.DefaultRemotePath, Kind: 2},
		{Category: "mount", Label: "local mount path", Value: m.cfg.Mount.LocalMountPath, Kind: 2},
		{Category: "mount", Label: "quit behavior", Value: string(m.cfg.Mount.QuitBehavior), Kind: 1, Options: []string{"prompt", "always_unmount", "leave_mounted"}},
		// Sync
		{Category: "sync", Label: "enable sync", Value: boolVal(m.cfg.Sync.Enabled), Kind: 0},
		{Category: "sync", Label: "repo url", Value: m.cfg.Sync.RepoURL, Kind: 2, Disabled: !m.cfg.Sync.Enabled},
		{Category: "sync", Label: "ssh key path", Value: m.cfg.Sync.SSHKeyPath, Kind: 2, Disabled: !m.cfg.Sync.Enabled},
		{Category: "sync", Label: "branch", Value: m.cfg.Sync.Branch, Kind: 2, Disabled: !m.cfg.Sync.Enabled},
		{Category: "sync", Label: "local path", Value: m.cfg.Sync.LocalPath, Kind: 2, Disabled: !m.cfg.Sync.Enabled},
		// Updates
		{Category: "updates", Label: "channel", Value: m.updateSettingsState().ChannelLabel, Kind: 2},
		{Category: "updates", Label: "version", Value: m.updateSettingsState().VersionLabel, Kind: 2},
		{Category: "updates", Label: "check now", Value: "", Kind: 2},
		{Category: "updates", Label: "apply update", Value: "", Kind: 2},
		{Category: "updates", Label: "PATH health", Value: m.updateSettingsState().PathHealth, Kind: 2},
		{Category: "updates", Label: "fix PATH", Value: "", Kind: 2},
		{Category: "updates", Label: updateSettingsNoteLabel(), Value: "", Kind: 2},
		// Tokens
		{Category: "tokens", Label: "manage tokens", Value: "", Kind: 2},
		{Category: "tokens", Label: "sync token definitions", Value: boolVal(m.cfg.Automation.SyncTokenDefinitions), Kind: 0, Disabled: !m.cfg.Sync.Enabled},
	}
	return items
}

func updateSettingsNoteLabel() string {
	if runtime.GOOS == "windows" {
		return "if relaunch fails, open a new terminal"
	}
	return "if relaunch fails, start SSHThing again"
}

func (m *Model) filteredSettingsIdxs() []int {
	items := m.settingsItems
	if m.settingsFilter == "" {
		idxs := make([]int, len(items))
		for i := range items {
			idxs[i] = i
		}
		return idxs
	}
	q := strings.ToLower(m.settingsFilter)
	var idxs []int
	for i, s := range items {
		if strings.Contains(strings.ToLower(s.Label), q) ||
			strings.Contains(strings.ToLower(s.Category), q) ||
			strings.Contains(strings.ToLower(s.Value), q) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

type updateSettingsStateInfo struct {
	ChannelLabel string
	VersionLabel string
	PathHealth   string
	CanApply     bool
	CanFixPath   bool
	Checking     bool
	Applying     bool
}

func (m Model) updateSettingsState() updateSettingsStateInfo {
	state := updateSettingsStateInfo{
		Checking: m.updateChecking,
		Applying: m.updateApplying,
	}
	if m.updateLast != nil {
		state.ChannelLabel = update.ChannelLabel(m.updateLast.Channel, m.updateLast.ChannelDetail)
		current := strings.TrimSpace(m.updateLast.CurrentVersion)
		if current == "" {
			current = "(unknown)"
		}
		latest := strings.TrimSpace(m.updateLast.LatestVersion)
		if latest == "" {
			latest = "(unknown)"
		}
		state.VersionLabel = current + " -> " + latest
		state.PathHealth = update.PathHealthLabel(m.updateLast.PathHealth)
		state.CanApply = m.updateLast.UpdateAvailable && m.updateLast.ApplyMode != update.ApplyModeGuidance && m.updateLast.ApplyMode != update.ApplyModeNone
		state.CanFixPath = !m.updateLast.PathHealth.Healthy && strings.TrimSpace(m.updateLast.PathHealth.DesiredPath) != ""
	}
	if state.VersionLabel == "" {
		v := strings.TrimSpace(m.currentVersion)
		if v == "" {
			v = "(unknown)"
		}
		state.VersionLabel = v + " -> (not checked)"
	}
	if state.ChannelLabel == "" {
		state.ChannelLabel = "(not checked)"
	}
	if state.PathHealth == "" {
		state.PathHealth = "(not checked)"
	}
	return state
}

// ── Token CRUD helpers (used by handlers) ─────────────────────────────

func (m Model) createToken(name string) (string, error) {
	grants, grantErr := m.selectedTokenHostGrants()
	if grantErr != nil {
		return "", grantErr
	}
	pepper, _ := securestore.GetOrCreateDevicePepper(rand.Reader)
	opts := authtoken.CreateOptions{
		DevicePepper: pepper,
		BindToDevice: len(pepper) > 0,
		SyncEnabled:  m.cfg.Automation.SyncTokenDefinitions,
	}
	raw, rec, err := authtoken.CreateToken(name, grants, m.masterPassword, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %v", err)
	}
	vault, err := authtoken.LoadVault()
	if err != nil {
		return "", fmt.Errorf("failed to load token vault: %v", err)
	}
	if err := vault.AddToken(raw, rec); err != nil {
		return "", fmt.Errorf("failed to add token: %v", err)
	}
	if err := authtoken.SaveVault(vault); err != nil {
		return "", fmt.Errorf("failed to save token vault: %v", err)
	}
	return raw, nil
}

func revokeToken(tokenID string) error {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return fmt.Errorf("failed to load token vault: %v", err)
	}
	if !vault.RevokeToken(tokenID) {
		return fmt.Errorf("token not found")
	}
	return authtoken.SaveVault(vault)
}

func deleteRevokedToken(tokenID string) (bool, error) {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return false, fmt.Errorf("failed to load token vault: %v", err)
	}
	deleted, err := vault.DeleteRevokedToken(tokenID)
	if err != nil {
		return false, err
	}
	if err := authtoken.SaveVault(vault); err != nil {
		return false, fmt.Errorf("failed to save token vault: %v", err)
	}
	return deleted, nil
}

func activateToken(tokenID, masterPassword string) (string, error) {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return "", fmt.Errorf("failed to load token vault: %v", err)
	}
	pepper, _ := securestore.GetOrCreateDevicePepper(rand.Reader)
	raw, err := vault.ActivateToken(tokenID, masterPassword, pepper)
	if err != nil {
		return "", fmt.Errorf("failed to activate token: %v", err)
	}
	if err := authtoken.SaveVault(vault); err != nil {
		return "", fmt.Errorf("failed to save token vault: %v", err)
	}
	return raw, nil
}

func copyTokenToClipboard(value string) error {
	return clipboard.WriteAll(value)
}

// ── Form helpers ──────────────────────────────────────────────────────

func (m Model) modalGroupOptions(current string) []string {
	options := []string{"Ungrouped"}
	seen := map[string]bool{"ungrouped": true}

	if m.appMode == appModeTeams {
		for _, host := range m.teamsItems {
			name := strings.TrimSpace(host.Group)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true
			options = append(options, name)
		}
	} else {
		for _, g := range m.groups {
			name := strings.TrimSpace(g)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true
			options = append(options, name)
		}
	}

	current = strings.TrimSpace(current)
	if current != "" {
		key := strings.ToLower(current)
		if !seen[key] {
			options = append(options, current)
		}
	}

	return options
}

func (m Model) modalSelectedGroupName() string {
	if len(m.formGroups) == 0 {
		return ""
	}
	idx := m.formGroupIdx
	if idx < 0 || idx >= len(m.formGroups) {
		idx = 0
	}
	name := strings.TrimSpace(m.formGroups[idx])
	if strings.EqualFold(name, "Ungrouped") {
		return ""
	}
	return name
}

func (m Model) validateForm() error {
	if len(m.formFields) < 6 {
		return fmt.Errorf("No form data")
	}

	if strings.TrimSpace(m.formFields[ui.FFHostname].Value) == "" {
		return fmt.Errorf("\u26A0 Host cannot be empty")
	}

	if strings.TrimSpace(m.formFields[ui.FFUsername].Value) == "" {
		return fmt.Errorf("\u26A0 Username cannot be empty")
	}

	if m.formFields[ui.FFPort].Value == "" {
		return fmt.Errorf("\u26A0 Port cannot be empty")
	}

	port := 0
	_, err := fmt.Sscanf(m.formFields[ui.FFPort].Value, "%d", &port)
	if err != nil {
		return fmt.Errorf("\u26A0 Port must be a valid number")
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("\u26A0 Port must be between 1 and 65535")
	}

	switch m.formAuthIdx {
	case 0: // password - optional
	case 1: // paste key
		if m.appMode == appModeTeams && m.formTeamHostID != "" && m.formTeamCredentialMode == "per_member" {
			return nil
		}
		pastedKey := strings.TrimSpace(m.formFields[ui.FFAuthDet].Value)
		if pastedKey == "" {
			return fmt.Errorf("\u26A0 Please paste your SSH private key or switch auth method")
		}
		if err := ssh.ValidatePrivateKey(pastedKey); err != nil {
			return fmt.Errorf("\u26A0 Invalid private key: %v", err)
		}
	case 2: // generate
		switch m.formKeyTypes[m.formKeyIdx] {
		case "ed25519", "rsa", "ecdsa":
		default:
			return fmt.Errorf("\u26A0 Invalid key type")
		}
	default:
		return fmt.Errorf("\u26A0 Invalid auth method")
	}

	return nil
}

func normalizePrivateKey(key string) string {
	key = strings.ReplaceAll(key, "\r\n", "\n")
	key = strings.ReplaceAll(key, "\r", "\n")
	if key != "" && !strings.HasSuffix(key, "\n") {
		key += "\n"
	}
	return key
}

// ── Settings mutation ─────────────────────────────────────────────────

func (m *Model) applySettingChange(idx int, action string) {
	if m.appMode == appModeTeams {
		switch idx {
		case 0:
			m.cfg.TeamsUI.WrapLabels = !m.cfg.TeamsUI.WrapLabels
		case 1:
			names := themeNames()
			cur := themeIdx(m.cfg.TeamsUI.Theme)
			if action == "left" {
				cur = (cur - 1 + len(names)) % len(names)
			} else {
				cur = (cur + 1) % len(names)
			}
			m.cfg.TeamsUI.Theme = names[cur]
			m.syncModeAppearance()
		case 2:
			iNames := iconSetNames()
			cur := iconSetIdx(m.cfg.TeamsUI.IconSet)
			if action == "left" {
				cur = (cur - 1 + len(iNames)) % len(iNames)
			} else {
				cur = (cur + 1) % len(iNames)
			}
			m.cfg.TeamsUI.IconSet = iNames[cur]
			m.syncModeAppearance()
		}
		return
	}

	switch idx {
	case 0: // vim mode
		m.cfg.UI.VimMode = !m.cfg.UI.VimMode
	case 1: // show icons
		m.cfg.UI.ShowIcons = !m.cfg.UI.ShowIcons
	case 2: // wrap labels
		m.cfg.UI.WrapLabels = !m.cfg.UI.WrapLabels
	case 3: // theme
		names := themeNames()
		cur := themeIdx(m.cfg.UI.Theme)
		if action == "left" {
			cur = (cur - 1 + len(names)) % len(names)
		} else {
			cur = (cur + 1) % len(names)
		}
		m.cfg.UI.Theme = names[cur]
		m.theme, m.themeIdx = ui.ThemeByName(m.cfg.UI.Theme)
	case 4: // icon set
		iNames := iconSetNames()
		cur := iconSetIdx(m.cfg.UI.IconSet)
		if action == "left" {
			cur = (cur - 1 + len(iNames)) % len(iNames)
		} else {
			cur = (cur + 1) % len(iNames)
		}
		m.cfg.UI.IconSet = iNames[cur]
		m.icons, m.iconIdx = ui.IconSetByName(m.cfg.UI.IconSet)
	case 5: // host key policy
		switch m.cfg.SSH.HostKeyPolicy {
		case config.HostKeyAcceptNew:
			m.cfg.SSH.HostKeyPolicy = config.HostKeyStrict
		case config.HostKeyStrict:
			m.cfg.SSH.HostKeyPolicy = config.HostKeyOff
		default:
			m.cfg.SSH.HostKeyPolicy = config.HostKeyAcceptNew
		}
	case 6: // keepalive - editable
		if action == "left" {
			m.cfg.SSH.KeepAliveSeconds = max(10, m.cfg.SSH.KeepAliveSeconds-5)
		} else if action == "right" {
			m.cfg.SSH.KeepAliveSeconds = min(300, m.cfg.SSH.KeepAliveSeconds+5)
		}
	case 7: // TERM mode
		switch m.cfg.SSH.TermMode {
		case config.TermAuto:
			m.cfg.SSH.TermMode = config.TermXterm
		case config.TermXterm:
			m.cfg.SSH.TermMode = config.TermCustom
		default:
			m.cfg.SSH.TermMode = config.TermAuto
		}
	case 8: // TERM custom - editable
	case 9: // password auto login
		m.cfg.SSH.PasswordAutoLogin = !m.cfg.SSH.PasswordAutoLogin
		if m.cfg.SSH.PasswordAutoLogin && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
			if err := ssh.CheckSSHPass(); err != nil {
				m.err = fmt.Errorf("Tip: install sshpass for best password auto-login on %s", runtime.GOOS)
			}
		}
	case 10: // password backend
		if runtime.GOOS != "windows" && m.cfg.SSH.PasswordAutoLogin {
			switch m.cfg.SSH.PasswordBackendUnix {
			case config.PasswordBackendSSHPassFirst:
				m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendAskpassFirst
			default:
				m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendSSHPassFirst
			}
		}
	case 11: // mount enabled
		m.cfg.Mount.Enabled = !m.cfg.Mount.Enabled
	case 12: // mount remote path - editable
	case 13: // mount local path - editable
	case 14: // mount quit behavior
		switch m.cfg.Mount.QuitBehavior {
		case config.MountQuitPrompt:
			m.cfg.Mount.QuitBehavior = config.MountQuitAlwaysUnmount
		case config.MountQuitAlwaysUnmount:
			m.cfg.Mount.QuitBehavior = config.MountQuitLeaveMounted
		default:
			m.cfg.Mount.QuitBehavior = config.MountQuitPrompt
		}
	case 15: // sync enabled
		m.cfg.Sync.Enabled = !m.cfg.Sync.Enabled
		if m.store != nil {
			syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, m.masterPassword)
			if err == nil {
				m.syncManager = syncMgr
			}
		}
	case 16, 17, 18, 19: // sync repo/key/branch/local - editable
	case 27: // manage tokens (opens token page)
	case 28: // sync token definitions
		if m.cfg.Sync.Enabled {
			m.cfg.Automation.SyncTokenDefinitions = !m.cfg.Automation.SyncTokenDefinitions
		}
	}
}

func (m *Model) applySettingsEditValue(idx int, val string) bool {
	val = strings.TrimSpace(val)
	if m.appMode == appModeTeams {
		ctx := context.Background()
		switch idx {
		case 3:
			if val == "" {
				m.err = fmt.Errorf("team name cannot be empty")
				return false
			}
			if err := m.createTeam(ctx, val); err != nil {
				m.err = err
				return false
			}
			m.err = fmt.Errorf("✓ Team created")
		case 4:
			if val == "" {
				m.err = fmt.Errorf("team name cannot be empty")
				return false
			}
			if err := m.renameCurrentTeam(ctx, val); err != nil {
				m.err = err
				return false
			}
			m.err = fmt.Errorf("✓ Team renamed")
		}
		return true
	}

	switch idx {
	case 6: // keepalive
		n, err := strconv.Atoi(val)
		if err != nil {
			m.err = fmt.Errorf("keepalive must be a number")
			return false
		}
		if n < 10 {
			n = 10
		}
		if n > 600 {
			n = 600
		}
		m.cfg.SSH.KeepAliveSeconds = n
	case 8: // TERM custom
		m.cfg.SSH.TermCustom = val
	case 12: // mount remote path
		if val != "" && !strings.HasPrefix(val, "/") {
			m.err = fmt.Errorf("\u26A0 remote path must be absolute (start with /)")
			return false
		}
		m.cfg.Mount.DefaultRemotePath = val
	case 13: // local mount path
		if val != "" {
			if !strings.HasPrefix(val, "/") {
				m.err = fmt.Errorf("\u26A0 mount path must be absolute (start with /)")
				return false
			}
			// Check if parent exists; auto-create only one level deep
			parent := filepath.Dir(val)
			if _, err := os.Stat(parent); os.IsNotExist(err) {
				m.err = fmt.Errorf("\u26A0 parent directory %s does not exist", parent)
				return false
			}
			if err := os.MkdirAll(val, 0755); err != nil {
				m.err = fmt.Errorf("\u26A0 cannot create %s: %v", val, err)
				return false
			}
		}
		m.cfg.Mount.LocalMountPath = val
	case 16: // sync repo
		m.cfg.Sync.RepoURL = val
	case 17: // sync key path
		m.cfg.Sync.SSHKeyPath = val
	case 18: // sync branch
		if val == "" {
			val = "main"
		}
		m.cfg.Sync.Branch = val
	case 19: // sync local path
		m.cfg.Sync.LocalPath = val
	}
	return true
}

func (m Model) buildTeamsViewParams() ui.TeamsHomeViewParams {
	currentTeam, hasCurrentTeam := m.teamsCurrentTeam()
	selectedHostID := ""
	if host, ok := m.teamsCurrentHost(); ok {
		selectedHostID = host.ID
	}

	type groupedHosts struct {
		name  string
		hosts []teams.TeamHost
	}

	groupMap := make(map[string][]teams.TeamHost)
	groupNames := make([]string, 0)
	seenGroups := make(map[string]bool)
	for _, host := range m.teamsItems {
		name := strings.TrimSpace(host.Group)
		if name == "" {
			name = "Ungrouped"
		}
		key := strings.ToLower(name)
		if !seenGroups[key] {
			seenGroups[key] = true
			groupNames = append(groupNames, name)
		}
		groupMap[name] = append(groupMap[name], host)
	}
	sort.Slice(groupNames, func(i, j int) bool {
		left := groupNames[i]
		right := groupNames[j]
		if left == "Ungrouped" {
			return false
		}
		if right == "Ungrouped" {
			return true
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})

	items := make([]ui.TeamsHomeListItem, 0, len(m.teamsItems)+len(groupNames))
	for _, groupName := range groupNames {
		hosts := append([]teams.TeamHost(nil), groupMap[groupName]...)
		sort.Slice(hosts, func(i, j int) bool {
			leftLabel := strings.TrimSpace(hosts[i].Label)
			if leftLabel == "" {
				leftLabel = strings.TrimSpace(hosts[i].Hostname)
			}
			rightLabel := strings.TrimSpace(hosts[j].Label)
			if rightLabel == "" {
				rightLabel = strings.TrimSpace(hosts[j].Hostname)
			}
			if strings.EqualFold(leftLabel, rightLabel) {
				return strings.ToLower(hosts[i].Hostname) < strings.ToLower(hosts[j].Hostname)
			}
			return strings.ToLower(leftLabel) < strings.ToLower(rightLabel)
		})

		items = append(items, ui.TeamsHomeListItem{
			IsGroup:   true,
			GroupName: groupName,
			HostCount: len(hosts),
		})
		for _, host := range hosts {
			lastConnected := ""
			if host.LastConnectedAt != nil {
				lastConnected = ui.FormatTimeAgo(time.UnixMilli(*host.LastConnectedAt))
			}
			items = append(items, ui.TeamsHomeListItem{
				Label:              strings.TrimSpace(host.Label),
				Hostname:           host.Hostname,
				Username:           host.Username,
				Port:               host.Port,
				Group:              groupName,
				CredentialMode:     strings.TrimSpace(host.CredentialMode),
				CredentialType:     strings.TrimSpace(host.CredentialType),
				LastConnectedLabel: lastConnected,
				Tags:               append([]string(nil), host.Tags...),
				Selected:           host.ID == selectedHostID,
			})
		}
	}

	summary := ui.TeamsHomeTeamSummary{
		HostCount: len(m.teamsItems),
		TeamCount: len(m.teamsList),
	}
	if hasCurrentTeam {
		summary.Name = strings.TrimSpace(currentTeam.Name)
		summary.Slug = strings.TrimSpace(currentTeam.Slug)
		summary.Role = strings.TrimSpace(string(currentTeam.Role))
	}

	statusLines := make([]ui.TeamsHomeStatusLine, 0, 1)
	if m.err != nil {
		message := m.err.Error()
		kind := "error"
		switch {
		case strings.HasPrefix(message, "\u2713"):
			kind = "success"
		case strings.HasPrefix(message, "\u26A0"):
			kind = "warning"
		case strings.HasPrefix(message, "\u2139"):
			kind = "info"
		}
		statusLines = append(statusLines, ui.TeamsHomeStatusLine{
			Kind:    kind,
			Message: message,
		})
	}

	footer := "enter create team  / search  ? commands  , settings  shift+tab cycle  T personal mode  q quit"
	if m.profileSignedIn() && len(m.teamsList) > 0 {
		footer = "a add  ctrl+1..9 switch team  / search  r refresh  , settings  shift+tab cycle  T personal mode  q quit"
		if len(m.teamsItems) > 0 {
			footer = "\u2191\u2193 nav  \u23CE connect  a add  e edit  d del  ctrl+1..9 switch team  / search  r refresh  , settings  shift+tab cycle  T personal mode  q quit"
		}
	}

	return ui.TeamsHomeViewParams{
		Page:         m.page,
		SessionValid: m.profileSignedIn(),
		HasTeams:     len(m.teamsList) > 0,
		CurrentTeam:  summary,
		Items:        items,
		StatusLines:  statusLines,
		FooterText:   footer,
	}
}

// ── View data builders ────────────────────────────────────────────────

func (m Model) buildHomeViewParams() ui.HomeViewParams {
	m.rebuildListItems()
	var items []ui.HomeListItem
	for _, it := range m.listItems {
		switch it.Kind {
		case ListItemGroup:
			items = append(items, ui.HomeListItem{
				IsGroup:   true,
				GroupName: it.GroupName,
				Collapsed: m.collapsed[it.GroupName],
				HostCount: it.Count,
			})
		case ListItemNewGroup:
			items = append(items, ui.HomeListItem{
				IsNewGroup: true,
			})
		case ListItemHost:
			host := it.Host
			mounted := false
			mountPath := ""
			if m.mountManager != nil {
				if ok, mt := m.mountManager.IsMounted(host.ID); ok && mt != nil {
					mounted = true
					mountPath = mt.LocalPath
				}
			}

			status := 0 // offline
			if mounted {
				status = 2 // connected
			} else if host.LastConnected != nil && time.Since(*host.LastConnected) < 5*time.Minute {
				status = 1 // idle
			}

			items = append(items, ui.HomeListItem{
				Label:         host.Label,
				GroupName:     host.GroupName,
				Hostname:      host.Hostname,
				Username:      host.Username,
				Port:          host.Port,
				KeyType:       host.KeyType,
				Tags:          hostSearchTags(host),
				Status:        status,
				Mounted:       mounted,
				MountPath:     mountPath,
				LastConnected: host.LastConnected,
			})
		}
	}

	// Count connected
	connected := 0
	if m.mountManager != nil {
		connected = len(m.mountManager.ListActive())
	}

	syncStage := ""
	if m.syncManager != nil && m.syncManager.IsEnabled() {
		syncStage = m.syncManager.StageString()
	}

	return ui.HomeViewParams{
		Items:        items,
		Cursor:       m.selectedIdx,
		Err:          m.err,
		SyncActivity: &ui.SyncActivity{Active: m.syncing, Frame: m.syncAnimFrame, Progress: m.syncProgress, Stage: syncStage},
		Page:         m.page,
		HostCount:    len(m.hosts),
		Connected:    connected,
	}
}

func (m Model) buildSearchResults() []ui.SearchResultItem {
	var results []ui.SearchResultItem
	for _, it := range m.spotlightItems {
		switch it.Kind {
		case SpotlightItemHost:
			if m.appMode == appModeTeams && !m.teamsImportMode {
				lbl := it.TeamHost.Label
				if lbl == "" {
					lbl = it.TeamHost.Hostname
				}
				results = append(results, ui.SearchResultItem{
					Label:       lbl,
					Hostname:    it.TeamHost.Hostname,
					GroupName:   it.GroupName,
					CommandMode: false,
				})
				continue
			}
			status := 0
			if m.mountManager != nil {
				if ok, _ := m.mountManager.IsMounted(it.Host.ID); ok {
					status = 2
				}
			}
			lbl := it.Host.Label
			if lbl == "" {
				lbl = it.Host.Hostname
			}
			results = append(results, ui.SearchResultItem{
				Label:     lbl,
				Hostname:  it.Host.Hostname,
				GroupName: it.GroupName,
				Status:    status,
			})
		case SpotlightItemCommand:
			results = append(results, ui.SearchResultItem{
				Label:       it.Detail,
				GroupName:   "command",
				CommandMode: true,
			})
		}
	}
	return results
}

// ── Theme / icon lookup helpers ───────────────────────────────────────

func themeNames() []string {
	names := make([]string, len(ui.Themes))
	for i, t := range ui.Themes {
		names[i] = t.Name
	}
	return names
}

func themeIdx(name string) int {
	for i, t := range ui.Themes {
		if strings.EqualFold(t.Name, name) {
			return i
		}
	}
	return 0
}

func iconSetNames() []string {
	names := make([]string, len(ui.IconPresets))
	for i, s := range ui.IconPresets {
		names[i] = s.Name
	}
	return names
}

func iconSetIdx(name string) int {
	for i, s := range ui.IconPresets {
		if strings.EqualFold(s.Name, name) {
			return i
		}
	}
	return 0
}

// ── Error auto-clear ──────────────────────────────────────────────────

func (m *Model) errorAutoClearCmd(prevErr string) tea.Cmd {
	currErr := ""
	if m.err != nil {
		currErr = m.err.Error()
	}
	if currErr == "" || currErr == prevErr {
		return nil
	}

	m.errSeq++
	seq := m.errSeq
	d := autoClearDuration(currErr)
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearErrMsg{seq: seq}
	})
}

func autoClearDuration(msg string) time.Duration {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "\u2713") {
		return 5 * time.Second
	}
	return 10 * time.Second
}
