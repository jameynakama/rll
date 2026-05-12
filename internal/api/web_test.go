package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/reallylonglink/internal/store"
)

func TestWebCreateLink(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	body := strings.NewReader("original_url=https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303; got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Result().Header.Get("Location")
	if !strings.HasPrefix(loc, "/links/") {
		t.Errorf("expected redirect to /links/{id}; got %s", loc)
	}
}

func TestWebCreateLinkNoScheme(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	body := strings.NewReader("original_url=cats.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303; got %d: %s", w.Code, w.Body.String())
	}

	_, err := store.New(testPool).GetLinkByOriginalUrl(context.Background(), "http://cats.com")
	if err != nil {
		t.Fatalf("expected link; got none")
	}
}

func TestWebCreateLinkEmptyURL(t *testing.T) {
	srv := newTestServer(t)
	body := strings.NewReader("original_url=")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "original url is required") {
		t.Errorf("expected error message in body; got %s", w.Body.String())
	}
}

func TestWebCreateLinkBadURL(t *testing.T) {
	srv := newTestServer(t)
	body := strings.NewReader("original_url=woof")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "please provide a url") {
		t.Errorf("expected error message in body; got %s", w.Body.String())
	}
}

func TestWebIndex(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Really Long Link") {
		t.Errorf("expected 'Really Long Link' in body; got %s", w.Body.String())
	}
}
