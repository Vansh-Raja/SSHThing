package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitManager handles Git operations for sync
type GitManager struct {
	repoPath   string
	repoURL    string
	branch     string
	sshKeyPath string
	repo       *git.Repository
}

// NewGitManager creates a new Git manager
func NewGitManager(repoPath, repoURL, branch, sshKeyPath string) *GitManager {
	return &GitManager{
		repoPath:   repoPath,
		repoURL:    repoURL,
		branch:     branch,
		sshKeyPath: sshKeyPath,
	}
}

// getAuth returns the authentication method for Git operations
func (gm *GitManager) getAuth() (transport.AuthMethod, error) {
	if gm.sshKeyPath == "" {
		// Try default SSH key locations
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		// Try common key locations
		keyPaths := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
		}

		for _, keyPath := range keyPaths {
			if _, err := os.Stat(keyPath); err == nil {
				gm.sshKeyPath = keyPath
				break
			}
		}
	}

	if gm.sshKeyPath == "" {
		return nil, fmt.Errorf("no SSH key found")
	}

	// Load SSH key
	auth, err := ssh.NewPublicKeysFromFile("git", gm.sshKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %w", err)
	}

	return auth, nil
}

// Init initializes the local repository by either cloning or opening existing
func (gm *GitManager) Init() error {
	// Ensure repo directory exists
	if err := os.MkdirAll(gm.repoPath, 0700); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	// Check if repo already exists
	repo, err := git.PlainOpen(gm.repoPath)
	if err == nil {
		gm.repo = repo
		return nil
	}

	if err != git.ErrRepositoryNotExists {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Repository doesn't exist - clone or init
	if gm.repoURL != "" {
		return gm.clone()
	}

	return gm.initLocal()
}

// clone clones the remote repository
func (gm *GitManager) clone() error {
	auth, err := gm.getAuth()
	if err != nil {
		return err
	}

	// First try cloning without specifying branch (use remote's default)
	repo, err := git.PlainClone(gm.repoPath, false, &git.CloneOptions{
		URL:      gm.repoURL,
		Auth:     auth,
		Progress: nil,
	})

	if err == nil {
		gm.repo = repo
		return nil
	}

	// Clone failed - check if it's because repo is empty
	errStr := err.Error()
	isEmptyRepo := errStr == "remote repository is empty" ||
		errStr == "reference not found" ||
		errStr == "couldn't find remote ref"

	if !isEmptyRepo {
		// Some other error (auth, network, etc.)
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Empty repo - initialize locally and set up remote
	os.RemoveAll(gm.repoPath)
	os.MkdirAll(gm.repoPath, 0700)
	return gm.initLocalWithRemote()
}

// initLocalWithRemote initializes a new local repo and configures the remote
func (gm *GitManager) initLocalWithRemote() error {
	repo, err := git.PlainInit(gm.repoPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	gm.repo = repo

	// Create initial commit with empty sync file
	syncFilePath := filepath.Join(gm.repoPath, SyncFileName)
	initialData := &SyncData{
		Version:   CurrentSyncVersion,
		UpdatedAt: time.Now(),
		Hosts:     []SyncHost{},
	}

	if err := writeJSONFile(syncFilePath, initialData); err != nil {
		return fmt.Errorf("failed to create initial sync file: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := w.Add(SyncFileName); err != nil {
		return fmt.Errorf("failed to stage sync file: %w", err)
	}

	_, err = w.Commit("Initial SSHThing sync", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "SSHThing",
			Email: "sync@sshthing.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Add remote
	if gm.repoURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{gm.repoURL},
		})
		if err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	return nil
}

// initLocal initializes a new local repository
func (gm *GitManager) initLocal() error {
	repo, err := git.PlainInit(gm.repoPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	gm.repo = repo

	// Create initial commit with empty sync file
	syncFilePath := filepath.Join(gm.repoPath, SyncFileName)
	initialData := &SyncData{
		Version:   CurrentSyncVersion,
		UpdatedAt: time.Now(),
		Hosts:     []SyncHost{},
	}

	if err := writeJSONFile(syncFilePath, initialData); err != nil {
		return fmt.Errorf("failed to create initial sync file: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := w.Add(SyncFileName); err != nil {
		return fmt.Errorf("failed to stage sync file: %w", err)
	}

	_, err = w.Commit("Initial SSHThing sync", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "SSHThing",
			Email: "sync@sshthing.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Add remote if URL is provided
	if gm.repoURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{gm.repoURL},
		})
		if err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	return nil
}

// Pull fetches and merges changes from the remote repository
func (gm *GitManager) Pull() error {
	if gm.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	if gm.repoURL == "" {
		return nil // No remote configured
	}

	auth, err := gm.getAuth()
	if err != nil {
		return err
	}

	w, err := gm.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Fetch first to get remote refs
	err = gm.repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		errStr := err.Error()
		// Ignore "already up-to-date" and "remote ref not found" errors
		if errStr != "already up-to-date" && errStr != "reference not found" {
			return fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Try to get the remote branch reference
	remoteRef, err := gm.repo.Reference(plumbing.NewRemoteReferenceName("origin", gm.branch), true)
	if err != nil {
		// Remote branch doesn't exist yet - this is fine, we'll push to create it
		return nil
	}

	// Reset local to remote (handles diverged histories)
	err = w.Reset(&git.ResetOptions{
		Commit: remoteRef.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to remote: %w", err)
	}

	return nil
}

// Push pushes local changes to the remote repository
func (gm *GitManager) Push() error {
	if gm.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	if gm.repoURL == "" {
		return nil // No remote configured
	}

	auth, err := gm.getAuth()
	if err != nil {
		return err
	}

	err = gm.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      true, // Force push to handle diverged histories
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// CommitChanges stages and commits the sync file
func (gm *GitManager) CommitChanges(message string) error {
	if gm.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	w, err := gm.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Stage the sync file
	if _, err := w.Add(SyncFileName); err != nil {
		return fmt.Errorf("failed to stage sync file: %w", err)
	}

	// Check if there are changes to commit
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		return nil // Nothing to commit
	}

	// Commit
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "SSHThing",
			Email: "sync@sshthing.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// GetSyncFilePath returns the path to the sync file in the repository
func (gm *GitManager) GetSyncFilePath() string {
	return filepath.Join(gm.repoPath, SyncFileName)
}

// HasRemote returns true if a remote is configured
func (gm *GitManager) HasRemote() bool {
	return gm.repoURL != ""
}

// writeJSONFile writes data to a JSON file with proper formatting
func writeJSONFile(filePath string, data interface{}) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonBytes, 0600)
}
