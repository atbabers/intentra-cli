package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/atbabers/intentra-cli/internal/config"
)

const (
	lockFileName    = "credentials.lock"
	lockTimeout     = 10 * time.Second
	lockStaleAge    = 30 * time.Second
	lockPollInterval = 50 * time.Millisecond
)

func GetLockFile() string {
	return filepath.Join(config.GetConfigDir(), lockFileName)
}

func AcquireCredentialLock() (func(), error) {
	lockFile := GetLockFile()

	if err := config.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	deadline := time.Now().Add(lockTimeout)

	for time.Now().Before(deadline) {
		if tryCleanStaleLock(lockFile) {
			file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err == nil {
				pid := os.Getpid()
			_, _ = file.WriteString(fmt.Sprintf("%d\n%d", pid, time.Now().UnixMilli()))
			file.Close()

				release := func() {
					os.Remove(lockFile)
				}
				return release, nil
			}
		}

		time.Sleep(lockPollInterval)
	}

	return nil, fmt.Errorf("timeout acquiring credential lock")
}

func tryCleanStaleLock(lockFile string) bool {
	info, err := os.Stat(lockFile)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > lockStaleAge {
		os.Remove(lockFile)
		return true
	}

	data, err := os.ReadFile(lockFile)
	if err != nil {
		return false
	}

	lines := string(data)
	var pid int
	_, _ = fmt.Sscanf(lines, "%d", &pid)

	if pid > 0 && !isProcessRunning(pid) {
		os.Remove(lockFile)
		return true
	}

	return false
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func WithCredentialLock(fn func() error) error {
	release, err := AcquireCredentialLock()
	if err != nil {
		return err
	}
	defer release()

	return fn()
}

func TryWithCredentialLock(fn func() error) error {
	lockFile := GetLockFile()

	if err := config.EnsureDirectories(); err != nil {
		return fn()
	}

	tryCleanStaleLock(lockFile)

	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fn()
	}

	pid := os.Getpid()
	_, _ = file.WriteString(strconv.Itoa(pid) + "\n" + strconv.FormatInt(time.Now().UnixMilli(), 10))
	file.Close()

	defer os.Remove(lockFile)

	return fn()
}
