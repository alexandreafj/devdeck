package gcal

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/alexandreafj/devdeck/internal/browser"
	"github.com/alexandreafj/devdeck/internal/exec"
)

// Console deep links used by the setup walkthrough. They are stable entry points
// into the Google Cloud console for the one-time OAuth client setup.
const (
	enableAPIURL   = "https://console.cloud.google.com/apis/library/calendar-json.googleapis.com"
	consentURL     = "https://console.cloud.google.com/apis/credentials/consent"
	credentialsURL = "https://console.cloud.google.com/apis/credentials"
)

// authTimeout bounds how long we wait for the user to finish the browser consent.
const authTimeout = 5 * time.Minute

// Authorize runs the guided `devdeck auth google` flow: it walks the user
// through creating a Google OAuth client (if they have not already), installs the
// downloaded credentials, then opens the browser for consent and caches the
// resulting token. credentialsPath and tokenPath may be empty to use the default
// locations in the config dir. It orchestrates the testable setup helpers and
// the I/O-bound OAuth exchange.
func Authorize(ctx context.Context, runner exec.CommandRunner, credentialsPath, tokenPath string) error {
	credsPath, err := resolvePath(credentialsPath, CredentialsPath)
	if err != nil {
		return err
	}
	tokenPath, err = resolvePath(tokenPath, TokenPath)
	if err != nil {
		return err
	}
	downloads, _ := downloadsDir() // best-effort; empty disables auto-detect

	data, err := ensureCredentials(ctx, os.Stdin, os.Stdout, runner, credsPath, downloads)
	if err != nil {
		return err
	}

	// A service account needs no browser sign-in — just calendar sharing.
	if credentialKind(data) == credServiceAccount {
		printServiceAccountNextSteps(os.Stdout, data)
		return nil
	}

	cfg, err := google.ConfigFromJSON(data, calendarScope)
	if err != nil {
		return fmt.Errorf("parse OAuth client: %w", err)
	}
	return runOAuth(ctx, os.Stdout, runner, cfg, tokenPath)
}

// ensureCredentials returns the raw credentials JSON to use. If a valid file is
// already present it is used as-is; otherwise the user is walked through creating
// one and the downloaded JSON is auto-detected (from downloads) or supplied by
// path, validated (OAuth "Desktop" client or service account), and installed to
// credsPath. It performs no network I/O, so it is unit-tested by driving in/out
// with scripted input.
func ensureCredentials(ctx context.Context, in io.Reader, out io.Writer, runner exec.CommandRunner, credsPath, downloads string) ([]byte, error) {
	if data, err := os.ReadFile(credsPath); err == nil && validateCredentials(data) == nil {
		fmt.Fprintf(out, "Using existing Google credentials at %s\n", credsPath)
		return data, nil
	}

	printSetupGuide(out)
	if err := browser.Open(ctx, runner, enableAPIURL); err != nil {
		fmt.Fprintln(out, "(couldn't open your browser automatically — use the links above)")
	}

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, "\nWhen the JSON has downloaded, press Enter to auto-detect it in %s,\n"+
			"or paste the file's full path here (Ctrl+C to cancel): ", displayDir(downloads))
		if !scanner.Scan() {
			return nil, fmt.Errorf("setup cancelled")
		}

		src := cleanPath(scanner.Text())
		if src == "" {
			cand, ok := findCredentialCandidate(downloads)
			if !ok {
				fmt.Fprintf(out, "Couldn't find a Google credentials JSON in %s yet.\n"+
					"Download it then press Enter again, or paste its full path.\n", displayDir(downloads))
				continue
			}
			src = cand
			fmt.Fprintf(out, "Found %s\n", cand)
		}

		if err := installCredentials(src, credsPath); err != nil {
			fmt.Fprintf(out, "That file didn't work: %v\nLet's try again.\n", err)
			continue
		}
		data, err := os.ReadFile(credsPath)
		if err != nil {
			fmt.Fprintf(out, "Saved file couldn't be read: %v\nLet's try again.\n", err)
			continue
		}
		fmt.Fprintf(out, "✓ Saved your credentials to %s\n", credsPath)
		return data, nil
	}
}

// printSetupGuide prints the one-time Google Cloud steps in plain language. It
// covers both supported credential types; DevDeck auto-detects which you provide.
func printSetupGuide(out io.Writer) {
	fmt.Fprint(out, `
Let's connect Google Calendar (one-time setup).

DevDeck uses YOUR OWN Google credentials — it ships none. Two options (pick one):
  • OAuth "Desktop app" client — browser sign-in; reads your "primary" calendar.
  • Service account — no browser; you share your calendar with it.

In the Google Cloud console:

  1. Enable the Calendar API (opening in your browser now):
       `+enableAPIURL+`
     Pick or create a project, then click "Enable".

  2. Create your credentials:
       `+credentialsURL+`
     OAuth:           "Create credentials" → "OAuth client ID"
                      → "Desktop app" → Create → "Download JSON".
                      (first set up the consent screen — User type "External",
                       add your address as a Test user: `+consentURL+`)
     Service account: "Create credentials" → "Service account" → open it →
                      "Keys" → "Add key" → "JSON".

Download/save the JSON, then come back here — I'll detect it next.
`)
}

// findCredentialCandidate returns the newest Google credentials JSON in dir
// (OAuth "Desktop" client or service account), or ok=false if none is found. It
// matches on file *contents*, so a file the user renamed still gets picked up —
// not just Google's default "client_secret_*.json" name.
func findCredentialCandidate(dir string) (string, bool) {
	if dir == "" {
		return "", false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var best string
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(full)
		if err != nil || validateCredentials(data) != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best, bestMod = full, info.ModTime()
		}
	}
	return best, best != ""
}

// installCredentials validates that src is a usable Google credentials JSON
// (OAuth "Desktop" client or service account) and copies it to dst (creating the
// parent directory), so the widget and the auth flow read it from one location.
func installCredentials(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := validateCredentials(data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

// printServiceAccountNextSteps tells the user how to finish service-account setup:
// share the calendar with the service account address and set calendar_id. No
// browser sign-in is needed for service accounts.
func printServiceAccountNextSteps(out io.Writer, data []byte) {
	email := serviceAccountEmail(data)
	if email == "" {
		email = "<the service account's client_email>"
	}
	fmt.Fprintf(out, `
✓ Service account detected — no browser sign-in needed.

Two steps so it can read your calendar:
  1. In Google Calendar → Settings → "Share with specific people", add this
     address with "See all event details":
       %s
  2. In your config.yml, set the calendar you shared:
       widgets:
         - type: google_calendar
           title: "Google Calendar"
           calendar_id: you@example.com   # the calendar shared above
           refresh: 5m

Then run devdeck.
`, email)
}

// cleanPath normalizes a path the user typed, pasted, or dragged into the
// terminal: it trims surrounding whitespace and matching quotes, unescapes
// shell-style backslash escapes (e.g. "My\ File.json" from a drag-and-drop), and
// expands a leading "~/".
func cleanPath(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if first, last := s[0], s[len(s)-1]; (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return s[1 : len(s)-1] // quoted: take verbatim, no unescaping
		}
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			b.WriteByte(s[i])
			continue
		}
		b.WriteByte(s[i])
	}
	out := b.String()
	if strings.HasPrefix(out, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			out = filepath.Join(home, out[2:])
		}
	}
	return out
}

// downloadsDir returns the user's Downloads directory (best effort).
func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads"), nil
}

// displayDir renders a directory for prompts, falling back to a generic label.
func displayDir(dir string) string {
	if dir == "" {
		return "your Downloads folder"
	}
	return dir
}

// runOAuth performs the browser consent and token exchange. It is the
// network-bound part of the flow and is kept thin (outside the unit tests).
func runOAuth(ctx context.Context, out io.Writer, runner exec.CommandRunner, cfg *oauth2.Config, tokenPath string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}
	defer func() { _ = listener.Close() }()
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/", listener.Addr().(*net.TCPAddr).Port)

	state, err := randomState()
	if err != nil {
		return err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{Handler: redirectHandler(state, codeCh, errCh)}
	go func() { _ = srv.Serve(listener) }()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintln(out, "\nOpening your browser to approve calendar access…")
	fmt.Fprintln(out, "If it doesn't open, visit:\n"+authURL)
	fmt.Fprintln(out, "\n(The \"unverified app\" screen is your own OAuth app — choose your account → Advanced → \"Go to <your app name> (unsafe)\".)")
	if err := browser.Open(ctx, runner, authURL); err != nil {
		fmt.Fprintln(out, "(couldn't open the browser automatically — use the link above)")
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(authTimeout):
		return fmt.Errorf("timed out waiting for authorization")
	}

	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}
	if err := SaveToken(tokenPath, tok); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n✓ Connected! Token saved to %s\n", tokenPath)
	fmt.Fprintln(out, "Add the widget to your config.yml and run devdeck:")
	fmt.Fprintln(out, "  widgets:\n    - type: google_calendar\n      title: \"Google Calendar\"\n      refresh: 5m")
	return nil
}

// redirectHandler serves the OAuth loopback redirect, validating state and
// forwarding the code (or an error) over the given channels.
func redirectHandler(state string, codeCh chan<- string, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth state mismatch")
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code returned")
			return
		}
		fmt.Fprintln(w, "DevDeck is connected to Google Calendar. You can close this tab.")
		codeCh <- code
	}
}

// randomState returns a random anti-CSRF state parameter.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
