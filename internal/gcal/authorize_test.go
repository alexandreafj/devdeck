package gcal

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectHandlerValidCode(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	h := redirectHandler("st8", codeCh, errCh)

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?state=st8&code=xyz", nil))

	select {
	case code := <-codeCh:
		if code != "xyz" {
			t.Errorf("captured code = %q, want xyz", code)
		}
	default:
		t.Fatal("expected the authorization code on the channel")
	}
}

func TestRedirectHandlerStateMismatch(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	h := redirectHandler("st8", codeCh, errCh)

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?state=wrong&code=xyz", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 on state mismatch", rec.Code)
	}
	select {
	case <-errCh:
	default:
		t.Fatal("expected an error for the state mismatch")
	}
}

func TestRedirectHandlerMissingCode(t *testing.T) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	h := redirectHandler("st8", codeCh, errCh)

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/?state=st8", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when code is missing", rec.Code)
	}
	select {
	case <-errCh:
	default:
		t.Fatal("expected an error when the code is missing")
	}
}
