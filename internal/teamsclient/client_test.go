package teamsclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartCLIAuthParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/teams/cli-auth/start" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authUrl":"https://app.example.com/cli-auth/complete","deviceCode":"ABCD-1234","sessionId":"sess_1","pollSecret":"poll_1","pollIntervalSeconds":2,"expiresAt":123456}`))
	}))
	defer server.Close()

	client := New(server.URL)
	got, err := client.StartCLIAuth(context.Background(), "SSHThing TUI")
	if err != nil {
		t.Fatalf("start cli auth: %v", err)
	}
	if got.SessionID != "sess_1" || got.DeviceCode != "ABCD-1234" {
		t.Fatalf("unexpected start response: %+v", got)
	}
}

func TestClientSurfacesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.Me(context.Background(), "token")
	if err == nil || err.Error() != "forbidden" {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
