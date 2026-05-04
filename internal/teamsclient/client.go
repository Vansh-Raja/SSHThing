package teamsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/personalsync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type apiError struct {
	Error string `json:"error"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Enabled() bool {
	return c.baseURL != ""
}

func (c *Client) StartCLIAuth(ctx context.Context, deviceName string) (teams.CliAuthStartResponse, error) {
	var out teams.CliAuthStartResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/cli-auth/start", "", map[string]string{
		"deviceName": strings.TrimSpace(deviceName),
	}, &out)
	return out, err
}

func (c *Client) PollCLIAuth(ctx context.Context, sessionID, pollSecret string) (teams.CliAuthPollResponse, error) {
	var out teams.CliAuthPollResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/cli-auth/poll", "", map[string]string{
		"sessionId":  sessionID,
		"pollSecret": pollSecret,
	}, &out)
	return out, err
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (teams.RefreshResponse, error) {
	var out teams.RefreshResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/cli-auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	}, &out)
	return out, err
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/teams/cli-auth/logout", "", map[string]string{
		"refreshToken": refreshToken,
	}, nil)
}

func (c *Client) Me(ctx context.Context, accessToken string) (teams.MeResponse, error) {
	var out teams.MeResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/me", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListTeams(ctx context.Context, accessToken string) ([]teams.TeamSummary, error) {
	var out []teams.TeamSummary
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/list", accessToken, nil, &out)
	return out, err
}

func (c *Client) CreateTeam(ctx context.Context, accessToken, name string) (teams.TeamSummary, error) {
	var out teams.TeamSummary
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/create", accessToken, map[string]string{
		"name": strings.TrimSpace(name),
	}, &out)
	return out, err
}

func (c *Client) RenameTeam(ctx context.Context, accessToken, teamID, name string) (teams.TeamSummary, error) {
	var out teams.TeamSummary
	err := c.doJSON(ctx, http.MethodPatch, "/api/teams/"+url.PathEscape(teamID), accessToken, map[string]string{
		"name": strings.TrimSpace(name),
	}, &out)
	return out, err
}

func (c *Client) DeleteTeam(ctx context.Context, accessToken, teamID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/teams/"+url.PathEscape(teamID), accessToken, nil, nil)
}

func (c *Client) ReorderTeams(ctx context.Context, accessToken string, teamIDs []string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/teams/reorder", accessToken, map[string]any{
		"teamIds": teamIDs,
	}, nil)
}

func (c *Client) ListTeamHosts(ctx context.Context, accessToken, teamID string) ([]teams.TeamHost, error) {
	var out []teams.TeamHost
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/"+url.PathEscape(teamID)+"/hosts", accessToken, nil, &out)
	return out, err
}

func (c *Client) CreateTeamHost(ctx context.Context, accessToken, teamID string, req teams.CreateTeamHostRequest) (teams.TeamHost, error) {
	var out teams.TeamHost
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/"+url.PathEscape(teamID)+"/hosts", accessToken, req, &out)
	return out, err
}

func (c *Client) GetTeamHost(ctx context.Context, accessToken, hostID string) (teams.TeamHostDetail, error) {
	var out teams.TeamHostDetail
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/hosts/"+url.PathEscape(hostID), accessToken, nil, &out)
	return out, err
}

func (c *Client) UpdateTeamHost(ctx context.Context, accessToken, hostID string, req teams.UpdateTeamHostRequest) error {
	body := map[string]any{
		"label":            req.Label,
		"hostname":         req.Hostname,
		"username":         req.Username,
		"port":             req.Port,
		"group":            req.Group,
		"tags":             req.Tags,
		"notes":            req.Notes,
		"credentialMode":   req.CredentialMode,
		"credentialType":   req.CredentialType,
		"secretVisibility": req.SecretVisibility,
	}
	if req.ClearSharedCredential {
		body["sharedCredential"] = nil
	} else if req.SharedCredential != "" {
		body["sharedCredential"] = req.SharedCredential
	}
	return c.doJSON(ctx, http.MethodPatch, "/api/teams/hosts/"+url.PathEscape(hostID), accessToken, body, nil)
}

func (c *Client) DeleteTeamHost(ctx context.Context, accessToken, hostID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/teams/hosts/"+url.PathEscape(hostID), accessToken, nil, nil)
}

func (c *Client) ListHostCredentialRoster(ctx context.Context, accessToken, hostID string) ([]teams.TeamHostCredentialRosterEntry, error) {
	var out []teams.TeamHostCredentialRosterEntry
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/hosts/"+url.PathEscape(hostID)+"/credentials", accessToken, nil, &out)
	return out, err
}

func (c *Client) RevealSharedCredential(ctx context.Context, accessToken, hostID string) (teams.RevealedTeamHostCredential, error) {
	var out teams.RevealedTeamHostCredential
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/hosts/"+url.PathEscape(hostID)+"/credentials/shared/reveal", accessToken, map[string]any{}, &out)
	return out, err
}

func (c *Client) RevealMemberCredential(ctx context.Context, accessToken, hostID, memberID string) (teams.RevealedTeamHostCredential, error) {
	var out teams.RevealedTeamHostCredential
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/hosts/"+url.PathEscape(hostID)+"/credentials/"+url.PathEscape(memberID)+"/reveal", accessToken, map[string]any{}, &out)
	return out, err
}

func (c *Client) DeleteMemberCredentialAsAdmin(ctx context.Context, accessToken, hostID, memberID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/teams/hosts/"+url.PathEscape(hostID)+"/credentials/"+url.PathEscape(memberID), accessToken, nil, nil)
}

func (c *Client) ListTeamAuditEvents(ctx context.Context, accessToken, teamID string) ([]teams.TeamAuditEvent, error) {
	var out []teams.TeamAuditEvent
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/"+url.PathEscape(teamID)+"/audit", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListTeamTokens(ctx context.Context, accessToken, teamID string) ([]teams.TeamAutomationToken, error) {
	var out []teams.TeamAutomationToken
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/"+url.PathEscape(teamID)+"/tokens", accessToken, nil, &out)
	return out, err
}

func (c *Client) CreateTeamToken(ctx context.Context, accessToken, teamID string, req teams.CreateTeamAutomationTokenRequest) (teams.CreateTeamAutomationTokenResponse, error) {
	var out teams.CreateTeamAutomationTokenResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/"+url.PathEscape(teamID)+"/tokens", accessToken, req, &out)
	return out, err
}

func (c *Client) RevokeTeamToken(ctx context.Context, accessToken, teamID, tokenDocID string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/teams/"+url.PathEscape(teamID)+"/tokens/"+url.PathEscape(tokenDocID), accessToken, map[string]any{}, nil)
}

func (c *Client) DeleteRevokedTeamToken(ctx context.Context, accessToken, teamID, tokenDocID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/teams/"+url.PathEscape(teamID)+"/tokens/"+url.PathEscape(tokenDocID), accessToken, nil, nil)
}

func (c *Client) GetTeamHostConnectConfig(ctx context.Context, accessToken, hostID string) (teams.TeamHostConnectConfig, error) {
	var out teams.TeamHostConnectConfig
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/hosts/"+url.PathEscape(hostID)+"/connect-config", accessToken, map[string]any{}, &out)
	return out, err
}

func (c *Client) ResolveTeamToken(ctx context.Context, req teams.TeamTokenResolveRequest) (teams.TeamTokenResolveResponse, error) {
	var out teams.TeamTokenResolveResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/tokens/resolve", "", req, &out)
	return out, err
}

func (c *Client) FinishTeamTokenExecution(ctx context.Context, executionID string, req teams.TeamTokenExecutionFinishRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/api/teams/tokens/executions/"+url.PathEscape(executionID)+"/finish", "", req, nil)
}

func (c *Client) GetPersonalVault(ctx context.Context, accessToken string) (personalsync.VaultSummary, error) {
	var out personalsync.VaultSummary
	err := c.doJSON(ctx, http.MethodGet, "/api/personal/vault", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListPersonalVaultItems(ctx context.Context, accessToken, since string) (personalsync.ListItemsResponse, error) {
	path := "/api/personal/vault/items"
	if strings.TrimSpace(since) != "" {
		path += "?since=" + url.QueryEscape(since)
	}
	var out personalsync.ListItemsResponse
	err := c.doJSON(ctx, http.MethodGet, path, accessToken, nil, &out)
	return out, err
}

func (c *Client) UpsertPersonalVaultItems(ctx context.Context, accessToken string, req personalsync.UpsertRequest) (personalsync.UpsertResponse, error) {
	var out personalsync.UpsertResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/personal/vault/items", accessToken, req, &out)
	return out, err
}

func (c *Client) RecordPersonalSyncEvent(ctx context.Context, accessToken string, req personalsync.SyncEventRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/api/personal/vault/events", accessToken, req, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path, accessToken string, body any, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("teams api base url is not configured")
	}

	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var apiErr apiError
		if len(data) > 0 && json.Unmarshal(data, &apiErr) == nil && strings.TrimSpace(apiErr.Error) != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		if len(data) > 0 {
			return fmt.Errorf("teams api %s %s failed: %s", method, path, strings.TrimSpace(string(data)))
		}
		return fmt.Errorf("teams api %s %s failed with status %d", method, path, res.StatusCode)
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
