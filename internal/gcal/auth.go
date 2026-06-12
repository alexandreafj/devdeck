package gcal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// calendarScope grants read-only access to the user's calendars.
	calendarScope = "https://www.googleapis.com/auth/calendar.readonly"

	credentialsFile = "google-credentials.json"
	tokenFile       = "google-token.json"
)

// CredentialsPath is where the user's OAuth client file (downloaded from Google
// Cloud) is expected, alongside the dashboard config (e.g.
// ~/Library/Application Support/devdeck/google-credentials.json on macOS).
func CredentialsPath() (string, error) { return configPath(credentialsFile) }

// TokenPath is where the OAuth token is cached after a successful authorization.
func TokenPath() (string, error) { return configPath(tokenFile) }

func configPath(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "devdeck", name), nil
}

// LoadCredentials reads an OAuth client file and returns an oauth2.Config scoped
// to read-only calendar access. The file is the "Desktop app" client JSON
// downloaded from the Google Cloud console.
func LoadCredentials(path string) (*oauth2.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials %s: %w", path, err)
	}
	cfg, err := google.ConfigFromJSON(data, calendarScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials %s: %w", path, err)
	}
	return cfg, nil
}

// LoadToken reads a cached OAuth token from path.
func LoadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token %s: %w", path, err)
	}
	return &tok, nil
}

// SaveToken writes tok to path as JSON, creating the parent directory if needed.
func SaveToken(path string, tok *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token %s: %w", path, err)
	}
	return nil
}
