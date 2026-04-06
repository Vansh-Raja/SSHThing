//go:build windows

package ssh

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

const expectedPrivateKeyAccessMask windows.ACCESS_MASK = 0x001f01ff

func TestNewTempKeyFile_WritesNormalizedContentWithProtectedACL(t *testing.T) {
	tempKey, err := NewTempKeyFile("line1\r\nline2")
	if err != nil {
		t.Fatalf("NewTempKeyFile returned error: %v", err)
	}
	path := tempKey.Path()

	if got, want := filepath.Dir(path), filepath.Join(os.TempDir(), "ssh-manager"); got != want {
		t.Fatalf("expected temp key dir %q, got %q", want, got)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if got, want := string(content), "line1\nline2\n"; got != want {
		t.Fatalf("expected normalized key %q, got %q", want, got)
	}

	assertProtectedPrivateKeyACL(t, path)

	if err := tempKey.Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected key file to be removed, stat err=%v", err)
	}
}

func assertProtectedPrivateKeyACL(t *testing.T, path string) {
	t.Helper()

	userSID, err := currentUserSID()
	if err != nil {
		t.Fatalf("currentUserSID failed: %v", err)
	}
	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		t.Fatalf("CreateWellKnownSid failed: %v", err)
	}

	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatalf("GetNamedSecurityInfo failed: %v", err)
	}

	control, _, err := sd.Control()
	if err != nil {
		t.Fatalf("Control failed: %v", err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("expected protected DACL, control=%#x", control)
	}

	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL failed: %v", err)
	}
	if dacl == nil {
		t.Fatalf("expected DACL to be present")
	}
	if dacl.AceCount != 2 {
		t.Fatalf("expected exactly 2 ACEs, got %d", dacl.AceCount)
	}

	want := map[string]struct{}{
		userSID.String():   {},
		systemSID.String(): {},
	}

	for i := uint16(0); i < dacl.AceCount; i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			t.Fatalf("GetAce(%d) failed: %v", i, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatalf("expected ACCESS_ALLOWED_ACE_TYPE, got %d", ace.Header.AceType)
		}
		if ace.Mask != expectedPrivateKeyAccessMask && ace.Mask != windows.GENERIC_ALL {
			t.Fatalf("expected private-key full access mask, got %#x", ace.Mask)
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		sidStr := sid.String()
		if _, ok := want[sidStr]; !ok {
			t.Fatalf("unexpected ACE SID %s on %s", sidStr, path)
		}
		delete(want, sidStr)
	}

	if len(want) != 0 {
		t.Fatalf("missing ACEs for expected SIDs: %v", want)
	}
}
