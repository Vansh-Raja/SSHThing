package ssh

import "strings"

func normalizePrivateKeyForFile(privateKey string) []byte {
	privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
	privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
	if privateKey != "" && !strings.HasSuffix(privateKey, "\n") {
		privateKey += "\n"
	}
	return []byte(privateKey)
}

// WritePrivateKeyFile writes a private key to disk using platform-appropriate permissions.
func WritePrivateKeyFile(path string, privateKey string) error {
	return writePrivateKeyFile(path, privateKey)
}

// SecureDeleteFile best-effort overwrites and removes a private key file.
func SecureDeleteFile(path string) error {
	return secureDeleteFile(path)
}
