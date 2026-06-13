package gcal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

const sampleCredentials = `{"installed":{
	"client_id":"id.apps.googleusercontent.com",
	"client_secret":"secret",
	"redirect_uris":["http://localhost"],
	"auth_uri":"https://accounts.google.com/o/oauth2/auth",
	"token_uri":"https://oauth2.googleapis.com/token"}}`

func TestLoadCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), credentialsFile)
	if err := os.WriteFile(path, []byte(sampleCredentials), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if cfg.ClientID != "id.apps.googleusercontent.com" {
		t.Errorf("ClientID = %q, want the configured id", cfg.ClientID)
	}
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != calendarScope {
		t.Errorf("Scopes = %v, want [%s]", cfg.Scopes, calendarScope)
	}
}

func TestLoadCredentialsErrors(t *testing.T) {
	if _, err := LoadCredentials(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing credentials file should error")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCredentials(bad); err == nil {
		t.Error("malformed credentials should error")
	}
}

func TestSaveAndLoadTokenRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", tokenFile) // sub dir must be created
	want := &oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}
	if err := SaveToken(path, want); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	got, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Errorf("round-trip token = %+v, want %+v", got, want)
	}
}

func TestLoadTokenErrors(t *testing.T) {
	if _, err := LoadToken(filepath.Join(t.TempDir(), "none.json")); err == nil {
		t.Error("missing token file should error")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadToken(bad); err == nil {
		t.Error("malformed token should error")
	}
}

func TestResetRemovesCredentialsAndToken(t *testing.T) {
	dir := t.TempDir()
	creds := filepath.Join(dir, credentialsFile)
	tok := filepath.Join(dir, tokenFile)
	if err := os.WriteFile(creds, []byte(sampleCredentials), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tok, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	if err := Reset(out, creds, tok); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if _, err := os.Stat(creds); !os.IsNotExist(err) {
		t.Error("credentials file should be removed")
	}
	if _, err := os.Stat(tok); !os.IsNotExist(err) {
		t.Error("token file should be removed")
	}
	if !strings.Contains(out.String(), "Disconnected") {
		t.Errorf("Reset should confirm disconnection:\n%s", out.String())
	}
}

func TestResetWhenNothingSaved(t *testing.T) {
	dir := t.TempDir()
	out := &bytes.Buffer{}
	err := Reset(out, filepath.Join(dir, credentialsFile), filepath.Join(dir, tokenFile))
	if err != nil {
		t.Fatalf("Reset with nothing to remove should not error: %v", err)
	}
	if !strings.Contains(out.String(), "Nothing to remove") {
		t.Errorf("Reset should report nothing to remove:\n%s", out.String())
	}
}

func TestConfigPaths(t *testing.T) {
	for _, fn := range []func() (string, error){CredentialsPath, TokenPath} {
		p, err := fn()
		if err != nil {
			t.Fatalf("path resolver: %v", err)
		}
		if filepath.Base(filepath.Dir(p)) != "devdeck" {
			t.Errorf("path %q should live under a devdeck dir", p)
		}
	}
}

func TestRandomStateIsUnique(t *testing.T) {
	a, err := randomState()
	if err != nil {
		t.Fatal(err)
	}
	b, err := randomState()
	if err != nil {
		t.Fatal(err)
	}
	if a == "" || a == b {
		t.Errorf("randomState should yield distinct non-empty values, got %q and %q", a, b)
	}
}
