// Package cloud centralises the Convex / web cloud base URL resolution so the
// TUI (cmd/sshthing) and the daemon (cmd/sshthing-daemon) agree on which
// SSHThing cloud they target.
//
// Resolution order:
//  1. SSHTHING_CLOUD_BASE_URL env var (trimmed, no trailing slash)
//  2. DefaultBaseURL — set at build time via ldflags, e.g.:
//       -X github.com/Vansh-Raja/SSHThing/internal/cloud.DefaultBaseURL=https://sshthing.vanshraja.me
//  3. http://localhost:3000 (dev fallback)
package cloud

import (
	"os"
	"strings"
)

// DefaultBaseURL is overridden at build time via -ldflags. Empty means dev.
var DefaultBaseURL = ""

// BaseURL returns the cloud service base URL using the resolution order
// described in the package doc.
func BaseURL() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("SSHTHING_CLOUD_BASE_URL")), "/"); v != "" {
		return v
	}
	if v := strings.TrimRight(strings.TrimSpace(DefaultBaseURL), "/"); v != "" {
		return v
	}
	return "http://localhost:3000"
}
