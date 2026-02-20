package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/99designs/keyring"
	"github.com/atbabers/intentra-cli/internal/config"
)

const (
	serviceName    = "intentra"
	credentialsKey = "credentials"
	cacheKeyKey    = "cache-encryption-key"
)

var (
	ring        keyring.Keyring
	ringOpenErr error
	ringOnce    sync.Once
)

func openKeyring() (keyring.Keyring, error) {
	ringOnce.Do(func() {
		backends := getBackendsForPlatform()

		cfg := keyring.Config{
			ServiceName:                    serviceName,
			KeychainTrustApplication:       true,
			KeychainSynchronizable:         false,
			KeychainAccessibleWhenUnlocked: true,
			FileDir:                        config.GetConfigDir(),
			FilePasswordFunc:               filePasswordPrompt,
			AllowedBackends:                backends,
		}

		ring, ringOpenErr = keyring.Open(cfg)
	})
	return ring, ringOpenErr
}

func getBackendsForPlatform() []keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.FileBackend,
		}
	case "windows":
		return []keyring.BackendType{
			keyring.WinCredBackend,
			keyring.FileBackend,
		}
	default:
		return []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.KeyCtlBackend,
			keyring.FileBackend,
		}
	}
}

func filePasswordPrompt(prompt string) (string, error) {
	key, err := getDerivedKey()
	if err != nil {
		return "", fmt.Errorf("failed to derive key for file backend: %w", err)
	}
	return string(key[:16]), nil
}

func StoreCredentialsInKeyring(creds *Credentials) error {
	return WithCredentialLock(func() error {
		return storeCredentialsInKeyringUnlocked(creds)
	})
}

func storeCredentialsInKeyringUnlocked(creds *Credentials) error {
	kr, err := openKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	err = kr.Set(keyring.Item{
		Key:  credentialsKey,
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to store credentials in keyring: %w", err)
	}

	if err := WriteEncryptedCache(creds); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write encrypted cache: %v\n", err)
	}

	return nil
}

func LoadCredentialsFromKeyring() (*Credentials, error) {
	if token := os.Getenv("INTENTRA_TOKEN"); token != "" {
		return &Credentials{
			AccessToken: token,
			TokenType:   "Bearer",
		}, nil
	}

	kr, err := openKeyring()
	if err != nil {
		return loadFromEncryptedCache()
	}

	item, err := kr.Get(credentialsKey)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return loadFromEncryptedCache()
		}
		return loadFromEncryptedCache()
	}

	var creds Credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

func DeleteCredentialsFromKeyring() error {
	return WithCredentialLock(func() error {
		return deleteCredentialsFromKeyringUnlocked()
	})
}

func deleteCredentialsFromKeyringUnlocked() error {
	kr, err := openKeyring()
	if err == nil {
		_ = kr.Remove(credentialsKey)
	}

	if err := DeleteEncryptedCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete encrypted cache: %v\n", err)
	}

	return DeleteCredentials()
}

func loadFromEncryptedCache() (*Credentials, error) {
	creds, err := ReadEncryptedCache()
	if err != nil {
		return loadFromCleartextAndMigrate()
	}
	return creds, nil
}

func loadFromCleartextAndMigrate() (*Credentials, error) {
	creds, err := LoadCredentials()
	if err != nil || creds == nil {
		return nil, err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Warning: migration to secure storage panicked: %v\n", r)
			}
		}()
		if err := MigrateToSecureStorage(creds); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: migration to secure storage failed: %v\n", err)
		}
	}()

	return creds, nil
}

func MigrateToSecureStorage(creds *Credentials) error {
	if creds == nil {
		return nil
	}

	if err := StoreCredentialsInKeyring(creds); err != nil {
		if err := WriteEncryptedCache(creds); err != nil {
			return fmt.Errorf("failed to write encrypted cache during migration: %w", err)
		}
	}

	encCreds, err := ReadEncryptedCache()
	if err != nil || encCreds == nil || encCreds.AccessToken != creds.AccessToken {
		return fmt.Errorf("encrypted cache verification failed")
	}

	credFile := config.GetCredentialsFile()
	if _, err := os.Stat(credFile); err == nil {
		if err := os.Remove(credFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove cleartext credentials: %v\n", err)
		}
	}

	return nil
}

func GetOrCreateCacheKey() ([]byte, error) {
	kr, err := openKeyring()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: keyring unavailable, using derived key: %v\n", err)
		return getDerivedKey()
	}

	item, err := kr.Get(cacheKeyKey)
	if err == nil && len(item.Data) == 32 {
		return item.Data, nil
	}

	key, err := generateRandomKey()
	if err != nil {
		return getDerivedKey()
	}

	_ = kr.Set(keyring.Item{
		Key:  cacheKeyKey,
		Data: key,
	})

	return key, nil
}
