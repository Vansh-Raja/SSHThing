package ssh

import (
	"fmt"
	"os"
	"strings"
)

const (
	askpassModeEnv     = "SSHTHING_ASKPASS_MODE"
	askpassEndpointEnv = "SSHTHING_ASKPASS_ENDPOINT"
	askpassNonceEnv    = "SSHTHING_ASKPASS_NONCE"
)

type askpassServer interface {
	Endpoint() string
	Nonce() string
	Close() error
}

// IsAskpassInvocation returns true when this process was launched as askpass helper.
func IsAskpassInvocation() bool {
	return strings.TrimSpace(os.Getenv(askpassModeEnv)) == "1"
}

// RunAskpassHelper prints a one-time password to stdout for OpenSSH askpass usage.
func RunAskpassHelper() error {
	if !IsAskpassInvocation() {
		return fmt.Errorf("not an askpass invocation")
	}

	prompt := ""
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}
	if prompt != "" && !looksLikePasswordPrompt(prompt) {
		return fmt.Errorf("unsupported askpass prompt")
	}

	endpoint := strings.TrimSpace(os.Getenv(askpassEndpointEnv))
	nonce := strings.TrimSpace(os.Getenv(askpassNonceEnv))
	if endpoint == "" || nonce == "" {
		return fmt.Errorf("missing askpass endpoint configuration")
	}

	secret, err := requestAskpassSecret(endpoint, nonce)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.WriteString(secret + "\n"); err != nil {
		return fmt.Errorf("failed to print askpass secret: %w", err)
	}
	return nil
}

func looksLikePasswordPrompt(prompt string) bool {
	p := strings.ToLower(strings.TrimSpace(prompt))
	if p == "" {
		return true
	}
	if strings.Contains(p, "yes/no") ||
		strings.Contains(p, "fingerprint") ||
		strings.Contains(p, "host key") ||
		strings.Contains(p, "verification code") ||
		strings.Contains(p, "one-time") {
		return false
	}
	if strings.Contains(p, "assword") {
		return true
	}
	if strings.Contains(p, "passphrase") {
		return true
	}
	return false
}
