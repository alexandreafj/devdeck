package gcal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/alexandreafj/devdeck/internal/browser"
	"github.com/alexandreafj/devdeck/internal/exec"
)

// Authorize runs the interactive OAuth flow used by `devdeck auth google`: it
// opens the browser to Google's consent screen, captures the authorization code
// on a loopback HTTP server, exchanges it for a token, and caches the token for
// the widget to use. It is inherently I/O-bound (network + browser) and so is
// kept thin and outside the unit-tested logic.
func Authorize(ctx context.Context, runner exec.CommandRunner) error {
	credsPath, err := CredentialsPath()
	if err != nil {
		return err
	}
	cfg, err := LoadCredentials(credsPath)
	if err != nil {
		return fmt.Errorf("%w\nCreate an OAuth 'Desktop' client in Google Cloud (Calendar API enabled) and save it to %s", err, credsPath)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}
	defer listener.Close()
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
	fmt.Println("Opening your browser to authorize DevDeck…")
	fmt.Println("If it doesn't open, visit:\n" + authURL)
	if err := browser.Open(ctx, runner, authURL); err != nil {
		fmt.Println("(could not open the browser automatically — use the link above)")
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(3 * time.Minute):
		return fmt.Errorf("timed out waiting for authorization")
	}

	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}

	tokenPath, err := TokenPath()
	if err != nil {
		return err
	}
	if err := SaveToken(tokenPath, tok); err != nil {
		return err
	}
	fmt.Println("Connected. Token saved to", tokenPath)
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
