package ssh

import (
	"os/exec"
	"strings"
	"testing"
)

func TestValidatePrivateKeyParsesGeneratedOpenSSHKey(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	privateKey, _, err := GenerateKey(KeyTypeEd25519, "sshthing-test")
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if err := ValidatePrivateKey(privateKey); err != nil {
		t.Fatalf("ValidatePrivateKey rejected generated key: %v", err)
	}
}

func TestValidatePrivateKeyRejectsHeaderOnlyText(t *testing.T) {
	key := strings.Join([]string{
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"not-a-real-key",
		"-----END OPENSSH PRIVATE KEY-----",
	}, "\n")

	if err := ValidatePrivateKey(key); err == nil {
		t.Fatalf("expected header-shaped invalid key to be rejected")
	}
}

func TestValidatePrivateKeyRejectsMissingHeader(t *testing.T) {
	if err := ValidatePrivateKey("not-a-private-key"); err == nil {
		t.Fatalf("expected missing header key to be rejected")
	}
}
