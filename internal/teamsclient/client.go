package teamsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type StartCLIAuthRequest struct {
	DeviceName string `json:"deviceName"`
}

type StartCLIAuthResponse struct {
	AuthURL              string `json:"authUrl"`
	DeviceCode           string `json:"deviceCode"`
	PollID               string `json:"pollId"`
	PollIntervalSeconds  int    `json:"pollIntervalSeconds"`
	LocalhostCallbackURL string `json:"localhostCallbackUrl,omitempty"`
}

type PollCLIAuthRequest struct {
	PollID string `json:"pollId"`
}

type PollCLIAuthResponse struct {
	Status  string            `json:"status"`
	Session *teamsSessionWire `json:"session,omitempty"`
	Auth    *teams.AuthState  `json:"auth,omitempty"`
}

type RefreshCLIAuthRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshCLIAuthResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type InviteMemberRequest struct {
	Email string     `json:"email"`
	Role  teams.Role `json:"role"`
}

type UpdateMemberRequest struct {
	Role teams.Role `json:"role"`
}

type teamsSessionWire struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	ActiveTeamID string    `json:"activeTeamId,omitempty"`
	UserID       string    `json:"userId,omitempty"`
	UserEmail    string    `json:"userEmail,omitempty"`
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Enabled() bool { return strings.TrimSpace(c.baseURL) != "" }

func (c *Client) StartCLIAuth(ctx context.Context, req StartCLIAuthRequest) (StartCLIAuthResponse, error) {
	var out StartCLIAuthResponse
	err := c.doJSON(ctx, http.MethodPost, "/cli-auth/start", "", req, &out)
	return out, err
}

func (c *Client) PollCLIAuth(ctx context.Context, req PollCLIAuthRequest) (PollCLIAuthResponse, error) {
	var out PollCLIAuthResponse
	err := c.doJSON(ctx, http.MethodPost, "/cli-auth/poll", "", req, &out)
	return out, err
}

func (c *Client) RefreshCLIAuth(ctx context.Context, req RefreshCLIAuthRequest) (RefreshCLIAuthResponse, error) {
	var out RefreshCLIAuthResponse
	err := c.doJSON(ctx, http.MethodPost, "/cli-auth/refresh", "", req, &out)
	return out, err
}

func (c *Client) LogoutCLIAuth(ctx context.Context, accessToken, refreshToken string) error {
	return c.doJSON(ctx, http.MethodPost, "/cli-auth/logout", accessToken, map[string]string{"refreshToken": refreshToken}, nil)
}

func (c *Client) Me(ctx context.Context, accessToken string) (teams.MeResponse, error) {
	var out teams.MeResponse
	err := c.doJSON(ctx, http.MethodGet, "/teams/me", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListHosts(ctx context.Context, accessToken string) ([]teams.Host, error) {
	var out []teams.Host
	err := c.doJSON(ctx, http.MethodGet, "/teams/current/hosts", accessToken, nil, &out)
	return out, err
}

func (c *Client) ListMembers(ctx context.Context, accessToken string) ([]teams.Member, error) {
	var out []teams.Member
	err := c.doJSON(ctx, http.MethodGet, "/teams/current/members", accessToken, nil, &out)
	return out, err
}

func (c *Client) CreateInvite(ctx context.Context, accessToken string, req InviteMemberRequest) (teams.Invite, error) {
	var out teams.Invite
	err := c.doJSON(ctx, http.MethodPost, "/teams/current/invites", accessToken, req, &out)
	return out, err
}

func (c *Client) UpdateMember(ctx context.Context, accessToken, memberID string, req UpdateMemberRequest) (teams.Member, error) {
	var out teams.Member
	err := c.doJSON(ctx, http.MethodPatch, "/teams/current/members/"+memberID, accessToken, req, &out)
	return out, err
}

func (c *Client) DeleteMember(ctx context.Context, accessToken, memberID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/teams/current/members/"+memberID, accessToken, nil, nil)
}

func (c *Client) Connect(ctx context.Context, accessToken, hostID string) (teams.ConnectPayload, error) {
	var out teams.ConnectPayload
	err := c.doJSON(ctx, http.MethodPost, "/teams/current/hosts/"+hostID+"/connect", accessToken, map[string]string{}, &out)
	return out, err
}

func (c *Client) doJSON(ctx context.Context, method, path, accessToken string, in any, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("teams client is not configured")
	}

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("teams api %s %s failed: %s", method, path, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
