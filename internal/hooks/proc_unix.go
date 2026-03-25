//go:build !windows

package hooks

import "syscall"

// detachedProcAttr returns SysProcAttr that creates a new session (setsid),
// detaching the child from the parent's process group so it survives parent exit.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
