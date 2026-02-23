//go:build !windows

package update

func findWindowsInstallExe() (string, string, error) {
	return "", "", nil
}

func detectWindowsPathHealth(_ string) (PathHealth, error) {
	return PathHealth{Healthy: true}, nil
}
