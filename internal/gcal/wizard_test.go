package gcal

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
)

const sampleWebCredentials = `{"web":{
	"client_id":"id.apps.googleusercontent.com",
	"client_secret":"secret",
	"redirect_uris":["http://localhost"],
	"auth_uri":"https://accounts.google.com/o/oauth2/auth",
	"token_uri":"https://oauth2.googleapis.com/token"}}`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDesktopCredentials(t *testing.T) {
	if err := validateDesktopCredentials([]byte(sampleCredentials)); err != nil {
		t.Errorf("desktop client should validate, got %v", err)
	}
	err := validateDesktopCredentials([]byte(sampleWebCredentials))
	if err == nil || !strings.Contains(err.Error(), "Desktop") {
		t.Errorf("web client should be rejected with a Desktop hint, got %v", err)
	}
	if validateDesktopCredentials([]byte(`{"other":{}}`)) == nil {
		t.Error("JSON without installed/web should be rejected")
	}
	if validateDesktopCredentials([]byte("not json")) == nil {
		t.Error("invalid JSON should be rejected")
	}
}

func TestFindCredentialCandidatePicksNewestDesktop(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "notes.json"), `{"hello":1}`)                    // not a client
	writeFile(t, filepath.Join(dir, "client_secret_web.json"), sampleWebCredentials) // web → ignored
	older := filepath.Join(dir, "client_secret_old.apps.googleusercontent.com.json")
	newer := filepath.Join(dir, "client_secret_new.apps.googleusercontent.com.json")
	writeFile(t, older, sampleCredentials)
	writeFile(t, newer, sampleCredentials)
	// Make `older` genuinely older.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, old, old); err != nil {
		t.Fatal(err)
	}

	got, ok := findCredentialCandidate(dir)
	if !ok {
		t.Fatal("should find a desktop client candidate")
	}
	if got != newer {
		t.Errorf("candidate = %q, want the newest desktop client %q", got, newer)
	}
}

func TestFindCredentialCandidateDetectsRenamedFile(t *testing.T) {
	dir := t.TempDir()
	// A desktop client the user renamed away from Google's default name.
	renamed := filepath.Join(dir, "OAuth Client ID JSON.json")
	writeFile(t, renamed, sampleCredentials)

	got, ok := findCredentialCandidate(dir)
	if !ok || got != renamed {
		t.Errorf("findCredentialCandidate = %q (ok=%v), want the renamed desktop client %q", got, ok, renamed)
	}
}

func TestCleanPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := map[string]string{
		`  /tmp/a.json  `:                 "/tmp/a.json",
		`"/tmp/a b.json"`:                 "/tmp/a b.json",
		`'/tmp/a b.json'`:                 "/tmp/a b.json",
		`/Users/x/OAuth\ Client\ ID.json`: "/Users/x/OAuth Client ID.json",
		`~/Downloads/c.json`:              filepath.Join(home, "Downloads/c.json"),
	}
	for in, want := range cases {
		if got := cleanPath(in); got != want {
			t.Errorf("cleanPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnsureCredentialsAcceptsEscapedDragAndDropPath(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "OAuth Client ID JSON.json")
	writeFile(t, src, sampleCredentials)

	// Simulate a terminal drag-and-drop: spaces escaped with backslashes.
	escaped := strings.ReplaceAll(src, " ", `\ `)
	if _, err := runEnsure(t, escaped+"\n", credsPath, t.TempDir()); err != nil {
		t.Fatalf("ensureCredentials with escaped path: %v", err)
	}
	if _, statErr := os.Stat(credsPath); statErr != nil {
		t.Errorf("escaped drag-and-drop path should install credentials: %v", statErr)
	}
}

func TestFindCredentialCandidateNoneOrMissingDir(t *testing.T) {
	if _, ok := findCredentialCandidate(t.TempDir()); ok {
		t.Error("empty dir should yield no candidate")
	}
	if _, ok := findCredentialCandidate(""); ok {
		t.Error("empty path should yield no candidate")
	}
	if _, ok := findCredentialCandidate(filepath.Join(t.TempDir(), "nope")); ok {
		t.Error("missing dir should yield no candidate")
	}
}

func TestInstallCredentials(t *testing.T) {
	src := filepath.Join(t.TempDir(), "client_secret.json")
	writeFile(t, src, sampleCredentials)
	dst := filepath.Join(t.TempDir(), "nested", "google-credentials.json")

	if err := installCredentials(src, dst); err != nil {
		t.Fatalf("installCredentials: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || !strings.Contains(string(got), "client_id") {
		t.Errorf("installed file missing or wrong: %v", err)
	}

	// A web client is rejected and not written.
	webSrc := filepath.Join(t.TempDir(), "web.json")
	writeFile(t, webSrc, sampleWebCredentials)
	webDst := filepath.Join(t.TempDir(), "out.json")
	if err := installCredentials(webSrc, webDst); err == nil {
		t.Error("installing a web client should error")
	}
	if _, err := os.Stat(webDst); !os.IsNotExist(err) {
		t.Error("a rejected client must not be written to dst")
	}
}

func runEnsure(t *testing.T, input, credsPath, downloads string) (*bytes.Buffer, error) {
	t.Helper()
	out := &bytes.Buffer{}
	_, err := ensureCredentials(context.Background(), strings.NewReader(input), out,
		&exectest.FakeRunner{}, credsPath, downloads)
	return out, err
}

func TestEnsureCredentialsUsesExisting(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	writeFile(t, credsPath, sampleCredentials)

	out, err := runEnsure(t, "", credsPath, t.TempDir())
	if err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if !strings.Contains(out.String(), "Using existing") {
		t.Errorf("should report using existing credentials:\n%s", out.String())
	}
}

func TestEnsureCredentialsAutoDetectsFromDownloads(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "cfg", "google-credentials.json")
	downloads := t.TempDir()
	writeFile(t, filepath.Join(downloads, "client_secret_x.apps.googleusercontent.com.json"), sampleCredentials)

	out, err := runEnsure(t, "\n", credsPath, downloads) // user just presses Enter
	if err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if _, statErr := os.Stat(credsPath); statErr != nil {
		t.Errorf("credentials should be installed at %s: %v", credsPath, statErr)
	}
	if !strings.Contains(out.String(), "Found") || !strings.Contains(out.String(), "Saved") {
		t.Errorf("walkthrough should report finding + saving:\n%s", out.String())
	}
}

func TestEnsureCredentialsAcceptsPastedPath(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	src := filepath.Join(t.TempDir(), "downloaded.json")
	writeFile(t, src, sampleCredentials)

	// Downloads empty → user pastes the path (quoted, as a drag-drop might give).
	out, err := runEnsure(t, "\""+src+"\"\n", credsPath, t.TempDir())
	if err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if _, statErr := os.Stat(credsPath); statErr != nil {
		t.Errorf("credentials should be installed from the pasted path: %v", statErr)
	}
	_ = out
}

func TestEnsureCredentialsRetriesAfterBadFile(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	bad := filepath.Join(t.TempDir(), "web.json")
	good := filepath.Join(t.TempDir(), "desktop.json")
	writeFile(t, bad, sampleWebCredentials)
	writeFile(t, good, sampleCredentials)

	out, err := runEnsure(t, bad+"\n"+good+"\n", credsPath, t.TempDir())
	if err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if !strings.Contains(out.String(), "didn't work") {
		t.Errorf("a bad file should prompt a retry:\n%s", out.String())
	}
	if _, statErr := os.Stat(credsPath); statErr != nil {
		t.Errorf("the good file should eventually install: %v", statErr)
	}
}

func TestEnsureCredentialsCancelsOnEOF(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	if _, err := runEnsure(t, "", credsPath, t.TempDir()); err == nil {
		t.Error("no input and no candidate should cancel with an error")
	}
}
