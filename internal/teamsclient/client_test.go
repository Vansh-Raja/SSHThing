package teamsclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestClientSurfacesHTMLAPIResponseAsDeployHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>missing route</body></html>`))
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetPersonalVault(context.Background(), "token")
	if err == nil {
		t.Fatalf("expected html response error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "returned HTTP 404 with an HTML page") {
		t.Fatalf("expected HTTP html hint, got %q", msg)
	}
	if strings.Contains(msg, "<!DOCTYPE html>") {
		t.Fatalf("expected HTML body to be suppressed, got %q", msg)
	}
}

func TestClientSurfacesHTMLSuccessAsInvalidAPIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>fallback</body></html>`))
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetPersonalVault(context.Background(), "token")
	if err == nil {
		t.Fatalf("expected html response error")
	}
	if !strings.Contains(err.Error(), "HTML page") {
		t.Fatalf("expected HTML response hint, got %q", err.Error())
	}
}
