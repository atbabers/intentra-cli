package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atbabers/intentra-cli/internal/config"
	"golang.org/x/crypto/hkdf"
)

const (
	encryptedCacheVersion = 1
	nonceSize             = 12
	keySize               = 32
)

func getEncryptedCacheFile() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.enc"), nil
}

func getCacheKeyFile() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".cache-key"), nil
}

func WriteEncryptedCache(creds *Credentials) error {
	if err := config.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	key, err := GetOrCreateCacheKey()
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	plaintext, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ciphertext, err := encrypt(plaintext, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	data := make([]byte, 1+len(ciphertext))
	data[0] = encryptedCacheVersion
	copy(data[1:], ciphertext)

	cacheFile, err := getEncryptedCacheFile()
	if err != nil {
		return fmt.Errorf("failed to determine cache path: %w", err)
	}
	tempFile := cacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted cache: %w", err)
	}

	if err := os.Rename(tempFile, cacheFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename encrypted cache: %w", err)
	}

	return nil
}

func ReadEncryptedCache() (*Credentials, error) {
	cacheFile, err := getEncryptedCacheFile()
	if err != nil {
		return nil, fmt.Errorf("failed to determine cache path: %w", err)
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read encrypted cache: %w", err)
	}

	if len(data) < 2 {
		return nil, fmt.Errorf("encrypted cache too short")
	}

	version := data[0]
	if version != encryptedCacheVersion {
		return nil, fmt.Errorf("unsupported encrypted cache version: %d", version)
	}

	key, err := readCacheKey()
	if err != nil {
		return nil, fmt.Errorf("failed to read cache key: %w", err)
	}

	plaintext, err := decrypt(data[1:], key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

func DeleteEncryptedCache() error {
	cacheFile, err := getEncryptedCacheFile()
	if err != nil {
		return fmt.Errorf("failed to determine cache path: %w", err)
	}
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	keyFile, err := getCacheKeyFile()
	if err != nil {
		return fmt.Errorf("failed to determine key path: %w", err)
	}
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func readCacheKey() ([]byte, error) {
	// Try keyring first (matches GetOrCreateCacheKey write path)
	kr, err := openKeyring()
	if err == nil {
		item, err := kr.Get(cacheKeyKey)
		if err == nil && len(item.Data) == keySize {
			return item.Data, nil
		}
	}

	// Fall back to file
	keyFile, err := getCacheKeyFile()
	if err != nil {
		return nil, fmt.Errorf("failed to determine key path: %w", err)
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return getDerivedKey()
		}
		return nil, err
	}

	if len(key) != keySize {
		return getDerivedKey()
	}

	return key, nil
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(data, key []byte) ([]byte, error) {
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func generateRandomKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func getDerivedKey() ([]byte, error) {
	machineID, err := getMachineID()
	if err != nil {
		machineID = "fallback-machine-id"
	}

	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	ikm := []byte(machineID + "|" + username)
	salt := []byte("intentra-cache-key-v1")
	info := []byte("credential-encryption")

	hkdfReader := hkdf.New(sha256.New, ikm, salt, info)
	key := make([]byte, keySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

func getMachineID() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxMachineID()
	case "darwin":
		return getDarwinMachineID()
	case "windows":
		return getWindowsMachineID()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func getLinuxMachineID() (string, error) {
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

	return "", fmt.Errorf("machine-id not found")
}

func getDarwinMachineID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, "\"")
				return uuid, nil
			}
		}
	}

	return "", fmt.Errorf("IOPlatformUUID not found")
}

func getWindowsMachineID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

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
