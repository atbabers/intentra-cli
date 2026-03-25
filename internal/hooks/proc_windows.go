//go:build windows

package hooks

import "syscall"

// detachedProcAttr returns SysProcAttr for Windows detached process creation.
// CREATE_NEW_PROCESS_GROUP detaches the child from the parent's console group.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000200}
}
