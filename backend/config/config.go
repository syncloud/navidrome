package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DataDir string

	AppUrl       string
	ClientID     string
	ClientSecret string
	AuthBaseURL  string
	RedirectURI  string
}

func (c *Config) Load() error {
	kv, err := loadKV(filepath.Join(c.DataDir, "config", "oidc.env"))
	if err != nil {
		return fmt.Errorf("oidc.env: %w", err)
	}
	c.AppUrl = kv["APP_URL"]
	c.ClientID = kv["OIDC_CLIENT_ID"]
	c.ClientSecret = kv["OIDC_CLIENT_SECRET"]
	c.AuthBaseURL = kv["OIDC_AUTH_BASE_URL"]
	c.RedirectURI = kv["OIDC_REDIRECT_URI"]

	for k, v := range map[string]string{
		"OIDC_CLIENT_ID":     c.ClientID,
		"OIDC_CLIENT_SECRET": c.ClientSecret,
		"OIDC_AUTH_BASE_URL": c.AuthBaseURL,
		"OIDC_REDIRECT_URI":  c.RedirectURI,
	} {
		if v == "" {
			return fmt.Errorf("%s is empty in oidc.env", k)
		}
	}
	return nil
}

func loadKV(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"`)
	}
	return out, nil
}
