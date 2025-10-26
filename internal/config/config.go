package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/gotray/internal/logging"
)

const (
	configDirName  = "gotray"
	configFileName = "config.b64"
)

// MenuItemType represents the supported menu item types.
type MenuItemType string

const (
	MenuItemText    MenuItemType = "text"
	MenuItemDivider MenuItemType = "divider"
	MenuItemCommand MenuItemType = "command"
	MenuItemURL     MenuItemType = "url"
	MenuItemMenu    MenuItemType = "menu"
	MenuItemQuit    MenuItemType = "quit"
	MenuItemRefresh MenuItemType = "refresh"
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
		logging.Debugf("using custom configuration path %s", custom)
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

// Load retrieves the base64-encoded configuration from disk.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	logging.Debugf("loading configuration from %s", path)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	payload := strings.TrimSpace(string(raw))
	if payload == "" {
		return &Config{}, nil
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save persists the configuration using base64 encoding on disk.
func Save(cfg *Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	data := base64.StdEncoding.EncodeToString(raw)
	data += "\n"

	path, err := Path()
	if err != nil {
		return err
	}

	logging.Debugf("writing base64 configuration to %s", path)
	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, []byte(data), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return os.Rename(tempFile, path)
}
