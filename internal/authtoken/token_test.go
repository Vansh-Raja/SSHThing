package authtoken

import (
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

func TestCreateVerifyAndResolveV2(t *testing.T) {
	pepper := []byte("0123456789abcdef0123456789abcdef")
	raw, rec, err := CreateToken("deploy", []HostGrant{
		{HostID: 10, DisplayLabel: "GPU"},
		{HostID: 11, DisplayLabel: "CPU"},
	}, "master-password", CreateOptions{DevicePepper: pepper, BindToDevice: true})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	if _, ok := Verify(raw, rec); !ok {
		t.Fatalf("Verify failed")
	}

	v := &Vault{Version: vaultVersion}
	if err := v.AddToken(raw, rec); err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}

	res, err := v.Resolve(raw, "CPU", pepper)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if res.HostID != 11 {
		t.Fatalf("expected host id 11, got %d", res.HostID)
	}
	if res.DBUnlockSecret != "master-password" {
		t.Fatalf("unexpected db unlock secret")
	}
	if res.LegacyPayload != nil {
		t.Fatalf("expected v2 resolve path")
	}
}

func TestResolveRequiresDevicePepperWhenBound(t *testing.T) {
	pepper := []byte("0123456789abcdef0123456789abcdef")
	raw, rec, err := CreateToken("deploy", []HostGrant{{HostID: 1, DisplayLabel: "A"}}, "pw", CreateOptions{DevicePepper: pepper, BindToDevice: true})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	v := &Vault{Version: vaultVersion}
	if err := v.AddToken(raw, rec); err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}
	if _, err := v.Resolve(raw, "A", nil); err == nil {
		t.Fatalf("expected resolve to fail without pepper")
	}
}

func TestVaultRevokeDeleteAndActivation(t *testing.T) {
	v := &Vault{Version: vaultVersion}
	raw, rec, err := CreateToken("deploy", []HostGrant{{HostID: 1, DisplayLabel: "Old"}}, "pw", CreateOptions{})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if err := v.AddToken(raw, rec); err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}

	if !v.SyncHostLabels(map[int]string{1: "New"}) {
		t.Fatalf("expected label sync change")
	}
	res, err := v.Resolve(raw, "New", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	v.MarkUsed(res.TokenIndex)
	if v.Tokens[res.TokenIndex].UseCount != 1 {
		t.Fatalf("expected use count update")
	}

	if !v.RevokeToken(v.Tokens[0].TokenID) {
		t.Fatalf("expected revoke")
	}
	if _, err := v.Resolve(raw, "New", nil); err == nil {
		t.Fatalf("expected revoked token to fail")
	}

	deleted, err := v.DeleteRevokedToken(v.Tokens[0].TokenID)
	if err != nil {
		t.Fatalf("DeleteRevokedToken failed: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete success")
	}
	if len(v.Tokens) != 0 {
		t.Fatalf("expected hard delete for local token")
	}
}

func TestDeleteRevokedTokenRejectsActive(t *testing.T) {
	v := &Vault{Version: vaultVersion}
	raw, rec, err := CreateToken("deploy", []HostGrant{{HostID: 1, DisplayLabel: "GPU"}}, "pw", CreateOptions{})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if err := v.AddToken(raw, rec); err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}
	if _, err := v.DeleteRevokedToken(v.Tokens[0].TokenID); err == nil {
		t.Fatalf("expected delete active token error")
	}
}

func TestSyncDefinitionsMerge(t *testing.T) {
	v := &Vault{Version: vaultVersion}
	now := time.Now().UTC()
	defs := []SyncTokenDef{{
		TokenID:     "tok-1",
		Name:        "deploy",
		CreatedAt:   now,
		UpdatedAt:   now,
		SyncEnabled: true,
		Hosts:       []SyncTokenHost{{HostID: 9, DisplayLabel: "GPU"}},
	}}
	if !v.MergeSyncDefinitions(defs) {
		t.Fatalf("expected merge change")
	}
	if len(v.Tokens) != 1 || v.Tokens[0].Salt != "" {
		t.Fatalf("expected metadata-only synced token")
	}
	if v.Tokens[0].IsUsable() {
		t.Fatalf("synced metadata token should not be usable until activation")
	}
	raw, err := v.ActivateToken("tok-1", "pw", nil)
	if err != nil {
		t.Fatalf("ActivateToken failed: %v", err)
	}
	if _, err := v.Resolve(raw, "GPU", nil); err != nil {
		t.Fatalf("expected activated token to resolve: %v", err)
	}
}

func TestLoadSaveVault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SSHTHING_DATA_DIR", tmp)
	if _, err := config.DataDir(); err != nil {
		t.Fatalf("data dir failed: %v", err)
	}

	v, err := LoadVault()
	if err != nil {
		t.Fatalf("LoadVault failed: %v", err)
	}
	raw, rec, err := CreateToken("deploy", []HostGrant{{HostID: 7, DisplayLabel: "GPU"}}, "pw", CreateOptions{})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if err := v.AddToken(raw, rec); err != nil {
		t.Fatalf("AddToken failed: %v", err)
	}
	if err := SaveVault(v); err != nil {
		t.Fatalf("SaveVault failed: %v", err)
	}

	v2, err := LoadVault()
	if err != nil {
		t.Fatalf("LoadVault reload failed: %v", err)
	}
	if len(v2.Tokens) != 1 {
		t.Fatalf("expected one token, got %d", len(v2.Tokens))
	}
}
