package service

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// TransferService provides SFTP upload/download operations.
type TransferService struct {
	Vault    *Vault
	CfgStore *CfgStore
	Notify   NotifyFunc

	// mu guards activeCmds.
	mu         sync.Mutex
	activeCmds map[string]*exec.Cmd
}

// TransferParams holds parameters for a transfer operation.
type TransferParams struct {
	HostID    string `json:"hostId"`
	Local     string `json:"local"`
	Remote    string `json:"remote"`
	Recursive bool   `json:"recursive"`
	Preserve  bool   `json:"preserve"`
}

// Upload copies a local path to a remote path via SFTP batch mode.
// Emits transfer.progress{status:"started"} before and transfer.progress{status:"finished"/"failed"} after.
func (ts *TransferService) Upload(params TransferParams, transferID string) error {
	return ts.doTransfer(params, transferID, ssh.TransferPut)
}

// Download copies a remote path to a local path via SFTP batch mode.
// Emits transfer.progress{status:"started"} before and transfer.progress{status:"finished"/"failed"} after.
func (ts *TransferService) Download(params TransferParams, transferID string) error {
	return ts.doTransfer(params, transferID, ssh.TransferGet)
}

// Cancel terminates an in-progress transfer identified by transferID.
// It kills the underlying sftp process and emits a failed notification.
// Returns an error if the transfer is not found or the process kill fails.
func (ts *TransferService) Cancel(transferID string) error {
	ts.mu.Lock()
	cmd, ok := ts.activeCmds[transferID]
	ts.mu.Unlock()

	if !ok {
		return fmt.Errorf("transfer %q not found or already finished", transferID)
	}

	if cmd.Process == nil {
		return fmt.Errorf("transfer %q process not started", transferID)
	}

	return cmd.Process.Kill()
}

func (ts *TransferService) doTransfer(params TransferParams, transferID string, dir ssh.TransferDirection) error {
	store := ts.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	intID, err := strconv.Atoi(params.HostID)
	if err != nil {
		return fmt.Errorf("invalid host id %q: %w", params.HostID, err)
	}
	model, err := store.GetHostByID(intID)
	if err != nil {
		return fmt.Errorf("get host: %w", err)
	}
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return fmt.Errorf("get secret: %w", err)
	}

	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	if ts.CfgStore != nil {
		cfg := ts.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
	}

	conn := ssh.Connection{
		Hostname:         model.Hostname,
		Username:         model.Username,
		Port:             model.Port,
		HostKeyPolicy:    hostKeyPolicy,
		KeepAliveSeconds: keepAlive,
	}
	if model.KeyType != "password" && secret != "" {
		conn.PrivateKey = secret
	}

	op := ssh.TransferOp{
		Direction: dir,
		Local:     params.Local,
		Remote:    params.Remote,
		Recursive: params.Recursive,
		Preserve:  params.Preserve,
	}

	// Notify start.
	if ts.Notify != nil {
		ts.Notify("transfer.progress", map[string]any{
			"transferId": transferID,
			"hostId":     params.HostID,
			"status":     "started",
			"direction":  directionName(dir),
			"local":      params.Local,
			"remote":     params.Remote,
		})
	}

	cmd, tempKey, err := ssh.ConnectTransfer(conn, []ssh.TransferOp{op}, true)
	if err != nil {
		ts.notifyFailed(transferID, params, dir, err)
		return fmt.Errorf("prepare transfer: %w", err)
	}
	defer func() {
		if tempKey != nil {
			_ = tempKey.Cleanup()
		}
	}()

	// Register the cmd so Cancel() can kill it.
	ts.mu.Lock()
	if ts.activeCmds == nil {
		ts.activeCmds = make(map[string]*exec.Cmd)
	}
	ts.activeCmds[transferID] = cmd
	ts.mu.Unlock()

	defer func() {
		ts.mu.Lock()
		delete(ts.activeCmds, transferID)
		ts.mu.Unlock()
	}()

	if err := cmd.Start(); err != nil {
		ts.notifyFailed(transferID, params, dir, err)
		return fmt.Errorf("start transfer: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		ts.notifyFailed(transferID, params, dir, err)
		return fmt.Errorf("transfer: %w", err)
	}

	// Notify finish.
	if ts.Notify != nil {
		ts.Notify("transfer.progress", map[string]any{
			"transferId": transferID,
			"hostId":     params.HostID,
			"status":     "finished",
			"direction":  directionName(dir),
			"local":      params.Local,
			"remote":     params.Remote,
		})
	}
	return nil
}

func (ts *TransferService) notifyFailed(transferID string, params TransferParams, dir ssh.TransferDirection, err error) {
	if ts.Notify != nil {
		ts.Notify("transfer.progress", map[string]any{
			"transferId": transferID,
			"hostId":     params.HostID,
			"status":     "failed",
			"direction":  directionName(dir),
			"local":      params.Local,
			"remote":     params.Remote,
			"error":      err.Error(),
		})
	}
}

func directionName(d ssh.TransferDirection) string {
	if d == ssh.TransferPut {
		return "upload"
	}
	return "download"
}
