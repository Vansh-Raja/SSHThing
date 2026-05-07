package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	gogitcfg "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// TestGitConnectivity performs a lightweight connectivity check against the
// given git repository URL using the provided SSH key path (or default keys
// if sshKeyPath is empty). It does not modify any local state.
func TestGitConnectivity(ctx context.Context, repoURL, sshKeyPath string) error {
	if strings.TrimSpace(repoURL) == "" {
		return fmt.Errorf("repoUrl is required")
	}

	auth, err := resolveSSHAuth(sshKeyPath)
	if err != nil {
		return fmt.Errorf("ssh auth: %w", err)
	}

	rem := gogit.NewRemote(nil, &gogitcfg.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	listOpts := &gogit.ListOptions{}
	if auth != nil {
		listOpts.Auth = auth
	}

	refs, err := rem.ListContext(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("cannot reach remote: %w", err)
	}
	_ = refs
	return nil
}

// resolveSSHAuth returns an SSH transport.AuthMethod for the given key path,
// or attempts default key locations if sshKeyPath is empty.
func resolveSSHAuth(sshKeyPath string) (transport.AuthMethod, error) {
	if strings.TrimSpace(sshKeyPath) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
			p := filepath.Join(home, ".ssh", name)
			if _, err := os.Stat(p); err == nil {
				sshKeyPath = p
				break
			}
		}
	}
	if strings.TrimSpace(sshKeyPath) == "" {
		// No key found; rely on ssh-agent or allow anonymous.
		return nil, nil
	}
	auth, err := gitssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("load ssh key %q: %w", sshKeyPath, err)
	}
	return auth, nil
}
