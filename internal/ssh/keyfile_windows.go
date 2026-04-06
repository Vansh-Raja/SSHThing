//go:build windows

package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func ensurePrivateKeyDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create private key directory %s: %w", dir, err)
	}
	if err := applyPrivateKeyACL(dir, true); err != nil {
		return fmt.Errorf("failed to secure private key directory %s for OpenSSH: %w", dir, err)
	}
	return nil
}

func writePrivateKeyFile(path string, privateKey string) error {
	dir := filepath.Dir(path)
	if err := ensurePrivateKeyDir(dir); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create private key file %s: %w", path, err)
	}

	if _, err := file.Write(normalizePrivateKeyForFile(privateKey)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("failed to write private key %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("failed to close private key file %s: %w", path, err)
	}

	if err := applyPrivateKeyACL(path, false); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("failed to secure private key file %s for OpenSSH: %w", path, err)
	}

	return nil
}

func secureDeleteFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err == nil {
		info, statErr := file.Stat()
		if statErr == nil && info != nil {
			zeros := make([]byte, info.Size())
			_, _ = file.Write(zeros)
		}
		_ = file.Close()
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func applyPrivateKeyACL(path string, isDir bool) error {
	userSID, err := currentUserSID()
	if err != nil {
		return err
	}
	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return fmt.Errorf("resolve LocalSystem SID: %w", err)
	}

	inheritance := uint32(windows.NO_INHERITANCE)
	if isDir {
		inheritance = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}

	entries := []windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.SET_ACCESS,
			Inheritance:       inheritance,
			Trustee: windows.TRUSTEE{
				TrusteeForm: windows.TRUSTEE_IS_SID,
				TrusteeType: windows.TRUSTEE_IS_USER,
				TrusteeValue: windows.TrusteeValueFromSID(
					userSID,
				),
			},
		},
		{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.SET_ACCESS,
			Inheritance:       inheritance,
			Trustee: windows.TRUSTEE{
				TrusteeForm: windows.TRUSTEE_IS_SID,
				TrusteeType: windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
				TrusteeValue: windows.TrusteeValueFromSID(
					systemSID,
				),
			},
		},
	}

	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("build ACL: %w", err)
	}

	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	); err != nil {
		return fmt.Errorf("apply ACL: %w", err)
	}
	return nil
}

func currentUserSID() (*windows.SID, error) {
	tokenUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("resolve current user SID: %w", err)
	}
	return tokenUser.User.Sid, nil
}
