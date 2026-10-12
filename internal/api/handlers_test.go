package api_test

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/api"
	"github.com/jameynakama/reallylonglink/internal/blocklist"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
	"github.com/jameynakama/reallylonglink/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, drop := testdb.Create("api")
	testPool = pool
	code := m.Run()
	drop()
	os.Exit(code)
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return api.NewRouter(api.RouterConfig{Queries: store.New(testPool)})
}

// newSafeBrowsingTestServer returns a router whose Safe Browsing lookups are
// answered by sbHandler.
func newSafeBrowsingTestServer(t *testing.T, sbHandler http.HandlerFunc) http.Handler {
	t.Helper()
	sb := httptest.NewServer(sbHandler)
	t.Cleanup(sb.Close)
	return api.NewRouter(api.RouterConfig{
		Queries:      store.New(testPool),
		SafeBrowsing: safebrowsing.New(sb.URL, "test-key"),
	})
}

// flaggedResponse answers with a protobuf SearchUrlsResponse holding one
// (empty) threat, which is all IsFlagged looks for.
func flaggedResponse(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{0x0a, 0x00})
}

func countLinks(t *testing.T) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(), "SELECT count(*) FROM links").Scan(&n); err != nil {
		t.Fatalf("countLinks: %v", err)
	}
	return n
}

func blockDomain(t *testing.T, domain string) {
	t.Helper()
	if _, err := blocklist.Block(context.Background(), testPool, domain); err != nil {
		t.Fatalf("blockDomain: %v", err)
	}
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "TRUNCATE links, blocked_domains RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func createTestLink(t *testing.T, originalUrl string, reallyLongPath string, reallyLongQuery string) store.Link {
	t.Helper()
	link, err := store.New(testPool).CreateLink(context.Background(), store.CreateLinkParams{
		OriginalUrl:     originalUrl,
		ReallyLongPath:  reallyLongPath,
		ReallyLongQuery: reallyLongQuery,
		PathHash:        fmt.Sprintf("%x", md5.Sum([]byte(reallyLongPath))),
	})
	if err != nil {
		t.Fatalf("createTestLink: %v", err)
	}
	return link
}

func TestListLinks(t *testing.T) {
	truncate(t)

	for i := range 3 {
		createTestLink(t, fmt.Sprintf("https://example.com/%d", i), fmt.Sprintf("https://example.com/reallylongurl/%d", i), "")
	}

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200; got %d", w.Result().StatusCode)
	}

	var links []store.Link
	if err := json.NewDecoder(w.Body).Decode(&links); err != nil {
		t.Fatalf("error decoding response: %v", err)
	}
	if len(links) != 3 {
		t.Errorf("expected 3 links; got %d", len(links))
	}
}

func TestGetLink404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/links/666", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Result().StatusCode)
	}
}

func TestGetLink(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl", "")

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/links/%d", link.ID), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200; got %d", w.Result().StatusCode)
	}

	if err := json.NewDecoder(w.Body).Decode(&link); err != nil {
		t.Fatalf("error decoding response: %v", err)
	}
	if link.OriginalUrl != "https://example.com" {
		t.Errorf("expected original url 'https://example.com'; got '%s'", link.OriginalUrl)
	}
}

// Should create a new link with the given original and a very long url
func TestCreateLink(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)

	body := strings.NewReader(`{"original_url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected 201; got %d: %s", w.Result().StatusCode, w.Body.String())
	}

	var created store.Link
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("error decoding response: %v", err)
	}
	if created.OriginalUrl != "https://example.com" {
		t.Errorf("expected original url 'https://example.com'; got '%s'", created.OriginalUrl)
	}
	if created.ReallyLongPath == "" {
		t.Errorf("expected really long path; got empty")
	}
	if created.ReallyLongQuery == "" {
		t.Errorf("expected really long query; got empty")
	}
	if _, err := store.New(testPool).GetLink(context.Background(), created.ID); err != nil {
		t.Errorf("created link not found in db: %v", err)
	}
}

// Should reject a url that Safe Browsing flags
func TestCreateLinkFlagged(t *testing.T) {
	truncate(t)
	srv := newSafeBrowsingTestServer(t, flaggedResponse)

	body := strings.NewReader(`{"original_url":"https://evil.example/"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", w.Result().StatusCode)
	}
	if n := countLinks(t); n != 0 {
		t.Errorf("expected no links created; got %d", n)
	}
}

// Should create the link anyway when Safe Browsing can't be reached
func TestCreateLinkSafeBrowsingDown(t *testing.T) {
	truncate(t)
	srv := newSafeBrowsingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	body := strings.NewReader(`{"original_url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected 201; got %d: %s", w.Result().StatusCode, w.Body.String())
	}
}

// Should create a link that Safe Browsing doesn't flag
func TestCreateLinkNotFlagged(t *testing.T) {
	truncate(t)
	srv := newSafeBrowsingTestServer(t, func(w http.ResponseWriter, r *http.Request) {})

	body := strings.NewReader(`{"original_url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("expected 201; got %d: %s", w.Result().StatusCode, w.Body.String())
	}
}

// Should reject a url on a blocked domain without asking Safe Browsing
func TestCreateLinkBlockedDomain(t *testing.T) {
	truncate(t)
	blockDomain(t, "bad-site.example")
	srv := newSafeBrowsingTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("expected no Safe Browsing call for a blocked domain")
	})

	body := strings.NewReader(`{"original_url":"https://www.bad-site.example/campaign/"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", body)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", w.Result().StatusCode)
	}
	if n := countLinks(t); n != 0 {
		t.Errorf("expected no links created; got %d", n)
	}
}

func TestRedirectToOriginalUrl404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/rll/doesnotexist", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Result().StatusCode)
	}
}

func TestRedirectToOriginalUrlWithQueryString(t *testing.T) {
	truncate(t)
	path := "seg1/seg2/seg3"
	query := "?utm_source=foo&ref=bar&id=baz"
	createTestLink(t, "https://example.com", path, query)

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/rll/"+path, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected 301; got %d", w.Result().StatusCode)
	}
	if w.Result().Header.Get("Location") != "https://example.com" {
		t.Errorf("expected location https://example.com; got %s", w.Result().Header.Get("Location"))
	}
}

func TestRedirectToOriginalUrl(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl", "")

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/rll/%s", link.ReallyLongPath), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected 301; got %d", w.Result().StatusCode)
	}
	if w.Result().Header.Get("Location") != link.OriginalUrl {
		t.Errorf("expected location %s; got %s", link.OriginalUrl, w.Result().Header.Get("Location"))
	}
}

func TestRedirectToOriginalUrlLegacy(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl", "")

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/rll/%s", link.ReallyLongPath), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected 301; got %d", w.Result().StatusCode)
	}
	if w.Result().Header.Get("Location") != link.OriginalUrl {
		t.Errorf("expected location %s; got %s", link.OriginalUrl, w.Result().Header.Get("Location"))
	}
}

func TestWebGetLink(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl", "")
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/links/%d", link.ID), nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), link.OriginalUrl) {
		t.Errorf("expected original url in body; got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "localhost:8080") {
		t.Errorf("expected redirect url in body; got %s", w.Body.String())
	}
}

func TestWebGetLink404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/links/999", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Code)
	}
}
