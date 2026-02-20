// Package device provides hardware-based device identification for Intentra.
// It generates immutable device IDs using HMAC-SHA256 hashes of hardware
// identifiers, supporting macOS, Linux, and Windows platforms.
package device

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// Salt for device ID generation - ensures consistent but unique IDs
	deviceIDSalt = "intentra-device-v1"
)

var (
	cachedDeviceID string
	deviceIDOnce   sync.Once
)

// GetDeviceID returns an HMAC-immutable device identifier.
// The ID is deterministic based on hardware identifiers but cannot be reversed.
func GetDeviceID() (string, error) {
	var err error
	deviceIDOnce.Do(func() {
		cachedDeviceID, err = generateDeviceID()
	})
	if err != nil {
		return "", err
	}
	return cachedDeviceID, nil
}

// generateDeviceID creates an HMAC-based immutable device ID.
func generateDeviceID() (string, error) {
	hwID, err := getHardwareID()
	if err != nil {
		return "", fmt.Errorf("failed to get hardware ID: %w", err)
	}

	// Create HMAC-SHA256 of hardware ID with salt
	// This ensures:
	// 1. ID is deterministic (same hardware = same ID)
	// 2. ID cannot be reversed to reveal hardware UUID
	// 3. ID is cryptographically secure
	h := hmac.New(sha256.New, []byte(deviceIDSalt))
	h.Write([]byte(hwID))
	hash := h.Sum(nil)

	// Return first 32 hex chars (128 bits) for a reasonable ID length
	return hex.EncodeToString(hash)[:32], nil
}

// getHardwareID retrieves the hardware-specific identifier.
func getHardwareID() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getMacOSHardwareUUID()
	case "linux":
		return getLinuxMachineID()
	case "windows":
		return getWindowsMachineGUID()
	default:
		return getFallbackID()
	}
}

// getMacOSHardwareUUID gets the hardware UUID on macOS.
func getMacOSHardwareUUID() (string, error) {
	// Use ioreg to get Hardware UUID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ioreg failed: %w", err)
	}

	// Parse for IOPlatformUUID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			// Extract UUID from line like: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, "\"")
				return uuid, nil
			}
		}
	}

	return "", fmt.Errorf("IOPlatformUUID not found")
}

// getLinuxMachineID gets the machine ID on Linux.
func getLinuxMachineID() (string, error) {
	// Try /etc/machine-id first (systemd)
	paths := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}

	// Fallback to hostname-based ID
	return getFallbackID()
}

// getWindowsMachineGUID gets the machine GUID on Windows.
func getWindowsMachineGUID() (string, error) {
	// Query registry for MachineGuid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "reg", "query",
		"HKLM\\SOFTWARE\\Microsoft\\Cryptography",
		"/v", "MachineGuid")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("registry query failed: %w", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MachineGuid") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[len(fields)-1], nil
			}
		}
	}

	return "", fmt.Errorf("MachineGuid not found")
}

// DeviceMetadata contains device information for API requests.
type DeviceMetadata struct {
	Hostname        string `json:"hostname,omitempty"`
	Username        string `json:"username,omitempty"`
	Platform        string `json:"platform,omitempty"`
	OSVersion       string `json:"os_version,omitempty"`
}

// GetMetadata returns device metadata for API requests.
func GetMetadata() DeviceMetadata {
	hostname, _ := os.Hostname()

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	osVersion := getOSVersion()

	return DeviceMetadata{
		Hostname:  hostname,
		Username:  username,
		Platform:  runtime.GOOS,
		OSVersion: osVersion,
	}
}

// getFallbackID creates a fallback ID from hostname + username.
// This is less reliable but works as a last resort.
func getFallbackID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "unknown"
	}

	return fmt.Sprintf("%s:%s", hostname, username), nil
}


// getOSVersion returns the OS version string.
func getOSVersion() string {
	switch runtime.GOOS {
	case "darwin":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "linux":
		// Try /etc/os-release
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "VERSION_ID=") {
					return strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
				}
			}
		}
	case "windows":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "cmd", "/c", "ver").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}
