package gcal

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2/google"
)

// credKind classifies a Google credentials JSON file.
type credKind int

const (
	credUnknown        credKind = iota
	credOAuthDesktop            // an OAuth "Desktop app" client ("installed": {…})
	credServiceAccount          // a service account ("type": "service_account")
)

// credentialKind detects what sort of credentials data holds.
func credentialKind(data []byte) credKind {
	var probe struct {
		Type      string          `json:"type"`
		Installed json.RawMessage `json:"installed"`
	}
	if json.Unmarshal(data, &probe) != nil {
		return credUnknown
	}
	switch {
	case probe.Type == "service_account":
		return credServiceAccount
	case len(probe.Installed) > 0:
		return credOAuthDesktop
	default:
		return credUnknown
	}
}

// validateCredentials checks that data is a Google credentials file DevDeck can
// use, either an OAuth "Desktop app" client or a service account, returning a
// user-actionable error otherwise (e.g. a "Web application" client).
func validateCredentials(data []byte) error {
	switch credentialKind(data) {
	case credServiceAccount:
		if _, err := google.JWTConfigFromJSON(data, calendarScope); err != nil {
			return fmt.Errorf("invalid service account JSON: %w", err)
		}
		return nil
	case credOAuthDesktop:
		if _, err := google.ConfigFromJSON(data, calendarScope); err != nil {
			return fmt.Errorf("invalid OAuth client JSON: %w", err)
		}
		return nil
	default:
		var probe map[string]json.RawMessage
		if json.Unmarshal(data, &probe) == nil {
			if _, isWeb := probe["web"]; isWeb {
				return fmt.Errorf("this is a 'Web application' client; use a 'Desktop app' client or a service account")
			}
		}
		return fmt.Errorf("not a Google OAuth 'Desktop' client or service account JSON")
	}
}

// serviceAccountEmail returns the client_email from a service account JSON (used
// to tell the user which address to share their calendar with).
func serviceAccountEmail(data []byte) string {
	var v struct {
		ClientEmail string `json:"client_email"`
	}
	_ = json.Unmarshal(data, &v) //nolint:errcheck // best-effort; returns an empty email on parse failure
	return v.ClientEmail
}
