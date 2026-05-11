package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameynakama/reallylonglink/internal/store"
)

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
