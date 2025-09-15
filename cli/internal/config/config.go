package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Config struct {
	Version        int           `json:"version"`
	Servers        []Server      `json:"servers"`
	DefaultServer  string        `json:"defaultServer"`
	Preferences    Preferences   `json:"preferences"`
}

type Server struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	VerifyTLS bool   `json:"verifyTLS"`
}

type Preferences struct {
	AutosaveEnabled    bool   `json:"autosaveEnabled"`
	AutosaveDebounceMs int    `json:"autosaveDebounceMs"`
	Theme              string `json:"theme"` // "dark" or "light"
}

func Default() Config {
	return Config{
		Version: 1,
		Servers: []Server{
			{Name: "local", URL: "http://127.0.0.1:8091", VerifyTLS: true},
		},
		DefaultServer: "local",
		Preferences: Preferences{
			AutosaveEnabled:    false,
			AutosaveDebounceMs: 1200,
			Theme:              "dark",
		},
	}
}

func pathForConfig(custom string) (string, error) {
	if custom != "" {
		return custom, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "SecretNotes")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

func LoadOrInit(customPath string) (Config, string, error) {
	cfgPath, err := pathForConfig(customPath)
	if err != nil {
		return Config{}, "", err
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := Default()
			if err := Save(cfgPath, &cfg); err != nil {
				return Config{}, "", err
			}
			return cfg, cfgPath, nil
		}
		return Config{}, "", err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, "", err
	}
	if err := cfg.Validate(); err != nil {
		// Reset to default on invalid
		cfg = Default()
		if err2 := Save(cfgPath, &cfg); err2 != nil {
			return Config{}, "", fmt.Errorf("config invalid: %v; also failed to write default: %v", err, err2)
		}
	}
	return cfg, cfgPath, nil
}

func Save(path string, cfg *Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, fs.FileMode(0o600)); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c *Config) Validate() error {
	if c.DefaultServer == "" || len(c.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}
	found := false
	for _, s := range c.Servers {
		if s.Name == c.DefaultServer {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("defaultServer not found in servers")
	}
	if c.Preferences.AutosaveDebounceMs <= 0 {
		c.Preferences.AutosaveDebounceMs = 1200
	}
	if c.Preferences.Theme == "" {
		c.Preferences.Theme = "dark"
	}
	return nil
}

func (c *Config) CurrentServer() *Server {
	for i, s := range c.Servers {
		if s.Name == c.DefaultServer {
			return &c.Servers[i]
		}
	}
	return nil
}

func (c *Config) OverrideURL(url string) {
	if srv := c.CurrentServer(); srv != nil {
		srv.URL = url
	}
}

func (c *Config) OverrideVerifyTLS(v bool) {
	if srv := c.CurrentServer(); srv != nil {
		srv.VerifyTLS = v
	}
}