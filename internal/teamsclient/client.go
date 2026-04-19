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

func (c *Client) CurrentWorkspace(ctx context.Context, accessToken string) (teams.WorkspaceSummary, error) {
	var out teams.WorkspaceSummary
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/workspaces/current", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListVaults(ctx context.Context, accessToken string) ([]teams.Vault, error) {
	var out []teams.Vault
	err := c.doJSON(ctx, http.MethodGet, "/api/teams/vaults", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListResources(ctx context.Context, accessToken, vaultID string) ([]teams.Resource, error) {
	var out []teams.Resource
	path := "/api/teams/resources?vaultId=" + url.QueryEscape(vaultID)
	err := c.doJSON(ctx, http.MethodGet, path, accessToken, nil, &out)
	return out, err
}

func (c *Client) ListMembers(ctx context.Context, accessToken, vaultID string) ([]teams.Member, error) {
	var out []teams.Member
	path := "/api/teams/members?vaultId=" + url.QueryEscape(vaultID)
	err := c.doJSON(ctx, http.MethodGet, path, accessToken, nil, &out)
	return out, err
}

func (c *Client) InviteMember(ctx context.Context, accessToken string, req teams.InviteRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/api/teams/invites", accessToken, req, nil)
}

func (c *Client) UpdateMemberRole(ctx context.Context, accessToken, memberID string, req teams.UpdateMemberRoleRequest) error {
	return c.doJSON(ctx, http.MethodPatch, "/api/teams/vault-members/"+url.PathEscape(memberID), accessToken, req, nil)
}

func (c *Client) RemoveMember(ctx context.Context, accessToken, memberID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/teams/members/"+url.PathEscape(memberID), accessToken, nil, nil)
}

func (c *Client) ConnectResource(ctx context.Context, accessToken, resourceID string) (teams.ConnectResponse, error) {
	var out teams.ConnectResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/teams/resources/"+url.PathEscape(resourceID)+"/connect", accessToken, map[string]any{}, &out)
	return out, err
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
