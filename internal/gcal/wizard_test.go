package gcal

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

// sampleServiceAccount is a valid service account JSON (with a freshly generated
// throwaway RSA key) so JWTConfigFromJSON parses it in tests.
var sampleServiceAccount = mustServiceAccount("svc@proj.iam.gserviceaccount.com")

func mustServiceAccount(email string) string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	b, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": email,
		"client_id":    "123",
		"private_key":  string(keyPEM),
		"token_uri":    "https://oauth2.googleapis.com/token",
	})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCredentials(t *testing.T) {
	if err := validateCredentials([]byte(sampleCredentials)); err != nil {
		t.Errorf("desktop client should validate, got %v", err)
	}
	if err := validateCredentials([]byte(sampleServiceAccount)); err != nil {
		t.Errorf("service account should validate, got %v", err)
	}
	err := validateCredentials([]byte(sampleWebCredentials))
	if err == nil || !strings.Contains(err.Error(), "Desktop") {
		t.Errorf("web client should be rejected with a Desktop hint, got %v", err)
	}
	if validateCredentials([]byte(`{"other":{}}`)) == nil {
		t.Error("JSON without installed/web/service_account should be rejected")
	}
	if validateCredentials([]byte("not json")) == nil {
		t.Error("invalid JSON should be rejected")
	}
}

func TestCredentialKind(t *testing.T) {
	if credentialKind([]byte(sampleCredentials)) != credOAuthDesktop {
		t.Error("installed client should be credOAuthDesktop")
	}
	if credentialKind([]byte(sampleServiceAccount)) != credServiceAccount {
		t.Error("service_account should be credServiceAccount")
	}
	if credentialKind([]byte(sampleWebCredentials)) != credUnknown {
		t.Error("web client should be credUnknown")
	}
	if got := serviceAccountEmail([]byte(sampleServiceAccount)); got != "svc@proj.iam.gserviceaccount.com" {
		t.Errorf("serviceAccountEmail = %q, want the client_email", got)
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

func TestFindCredentialCandidatePrefersOAuthOverServiceAccount(t *testing.T) {
	dir := t.TempDir()
	oauth := filepath.Join(dir, "oauth-client.json")
	svc := filepath.Join(dir, "service-account.json")
	writeFile(t, oauth, sampleCredentials)
	writeFile(t, svc, sampleServiceAccount)
	// Make the service account the *newer* file, so picking by mod time alone
	// would wrongly choose it: OAuth should still win.
	newer := time.Now()
	older := newer.Add(-time.Hour)
	if err := os.Chtimes(oauth, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(svc, newer, newer); err != nil {
		t.Fatal(err)
	}

	got, ok := findCredentialCandidate(dir)
	if !ok {
		t.Fatal("should find a candidate")
	}
	if got != oauth {
		t.Errorf("candidate = %q, want the OAuth desktop client %q (preferred over a newer service account)", got, oauth)
	}
}

func TestFindCredentialCandidateFallsBackToServiceAccount(t *testing.T) {
	dir := t.TempDir()
	svc := filepath.Join(dir, "service-account.json")
	writeFile(t, svc, sampleServiceAccount)

	got, ok := findCredentialCandidate(dir)
	if !ok || got != svc {
		t.Errorf("findCredentialCandidate = %q (ok=%v), want the service account %q when no OAuth client is present", got, ok, svc)
	}
}

func TestFindCredentialCandidateSkipsHugeFiles(t *testing.T) {
	dir := t.TempDir()
	small := filepath.Join(dir, "small-oauth.json")
	huge := filepath.Join(dir, "huge-oauth.json")
	writeFile(t, small, sampleCredentials)
	// A valid-but-oversized JSON (e.g. a multi-GB data export that happens to be
	// JSON) must be skipped without being read into memory.
	writeFile(t, huge, sampleCredentials+strings.Repeat(" ", (1<<20)+1))
	// Make the huge file newer, so "newest wins" would pick it without a cap.
	now := time.Now()
	old := now.Add(-time.Hour)
	if err := os.Chtimes(small, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(huge, now, now); err != nil {
		t.Fatal(err)
	}

	got, ok := findCredentialCandidate(dir)
	if !ok || got != small {
		t.Errorf("findCredentialCandidate = %q (ok=%v), want the small file %q (oversized file skipped)", got, ok, small)
	}
}

func TestCleanPathStripsArrowKeyEscape(t *testing.T) {
	// An accidental Up-arrow at the prompt sends an ANSI escape ("\x1b[A"); it
	// must not be mistaken for a path (cleans to empty, triggering auto-detect).
	if got := cleanPath("\x1b[A"); got != "" {
		t.Errorf("cleanPath(arrow) = %q, want empty", got)
	}
	if got := cleanPath("\x1b[A/tmp/a.json"); got != "/tmp/a.json" {
		t.Errorf("cleanPath with leading escape = %q, want /tmp/a.json", got)
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

func TestEnsureCredentialsInstallsServiceAccount(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	downloads := t.TempDir()
	// The user's renamed service-account download.
	writeFile(t, filepath.Join(downloads, "Calendar API Projects.json"), sampleServiceAccount)

	out := &bytes.Buffer{}
	data, err := ensureCredentials(context.Background(), strings.NewReader("\n"), out,
		&exectest.FakeRunner{}, credsPath, downloads)
	if err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if credentialKind(data) != credServiceAccount {
		t.Error("expected a service account to be detected and installed")
	}
	if _, statErr := os.Stat(credsPath); statErr != nil {
		t.Errorf("service-account credentials should be copied into place: %v", statErr)
	}
}

func TestPrintServiceAccountNextSteps(t *testing.T) {
	out := &bytes.Buffer{}
	printServiceAccountNextSteps(out, []byte(sampleServiceAccount))
	s := out.String()
	if !strings.Contains(s, "svc@proj.iam.gserviceaccount.com") || !strings.Contains(s, "calendar_id") {
		t.Errorf("next steps should mention the service account email and calendar_id:\n%s", s)
	}
}

func TestEnsureCredentialsCancelsOnEOF(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "google-credentials.json")
	if _, err := runEnsure(t, "", credsPath, t.TempDir()); err == nil {
		t.Error("no input and no candidate should cancel with an error")
	}
}
