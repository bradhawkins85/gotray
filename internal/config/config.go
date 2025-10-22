package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const (
	configDirName  = "gotray"
	configFileName = "config.enc"
	saltSize       = 16
	nonceSize      = 12
)

// MenuItemType represents the supported menu item types.
type MenuItemType string

const (
	MenuItemText    MenuItemType = "text"
	MenuItemDivider MenuItemType = "divider"
	MenuItemCommand MenuItemType = "command"
	MenuItemURL     MenuItemType = "url"
	MenuItemMenu    MenuItemType = "menu"
)

// MenuItem represents a single menu entry in the tray.
type MenuItem struct {
	ID          string       `json:"id"`
	Order       int          `json:"order"`
	Type        MenuItemType `json:"type"`
	Label       string       `json:"label,omitempty"`
	Command     string       `json:"command,omitempty"`
	Arguments   []string     `json:"arguments,omitempty"`
	WorkingDir  string       `json:"workingDir,omitempty"`
	URL         string       `json:"url,omitempty"`
	Description string       `json:"description,omitempty"`
	ParentID    string       `json:"parentId,omitempty"`
	CreatedUTC  string       `json:"createdUtc"`
	UpdatedUTC  string       `json:"updatedUtc"`
}

// Config represents the persisted configuration file.
type Config struct {
	Items []MenuItem `json:"items"`
}

// Path returns the resolved configuration file path.
func Path() (string, error) {
	if custom := os.Getenv("GOTRAY_CONFIG_PATH"); custom != "" {
		if err := os.MkdirAll(filepath.Dir(custom), 0o700); err != nil {
			return "", fmt.Errorf("ensure custom config directory: %w", err)
		}
		return custom, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine user config dir: %w", err)
	}

	dir := filepath.Join(base, configDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("ensure config directory: %w", err)
	}

	return filepath.Join(dir, configFileName), nil
}

// Load retrieves the encrypted configuration using the provided passphrase.
func Load(passphrase string) (*Config, error) {
	if passphrase == "" {
		return nil, errors.New("missing passphrase for configuration decryption")
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	data, err := decrypt(raw, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save persists the configuration encrypted with the provided passphrase.
func Save(cfg *Config, passphrase string) error {
	if passphrase == "" {
		return errors.New("missing passphrase for configuration encryption")
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	data, err := encrypt(raw, passphrase)
	if err != nil {
		return fmt.Errorf("encrypt config: %w", err)
	}

	path, err := Path()
	if err != nil {
		return err
	}

	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return fmt.Errorf("write encrypted config: %w", err)
	}

	return os.Rename(tempFile, path)
}

func encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	sealed := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, saltSize+nonceSize+len(sealed))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

func decrypt(ciphertext []byte, passphrase string) ([]byte, error) {
	if len(ciphertext) < saltSize+nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	salt := ciphertext[:saltSize]
	nonce := ciphertext[saltSize : saltSize+nonceSize]
	payload := ciphertext[saltSize+nonceSize:]

	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return gcm.Open(nil, nonce, payload, nil)
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	const (
		keyLength = 32
		n         = 1 << 15
		r         = 8
		p         = 1
	)

	key, err := scrypt.Key([]byte(passphrase), salt, n, r, p, keyLength)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	return key, nil
}
