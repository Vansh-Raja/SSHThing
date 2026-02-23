//go:build windows

package update

import "syscall"

func windowsHiddenSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
