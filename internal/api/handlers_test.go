package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/api"
	"github.com/jameynakama/reallylonglink/internal/store"
)

var testPool *pgxpool.Pool

func getRequiredEnvVar(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s must be set", key)
	}
	return v
}

func getDBConn(ctx context.Context, dbURL string) *pgxpool.Pool {
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("error establishing test database connection: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("cannot ping test database %s: %v", dbURL, err)
	}
	return db
}

func TestMain(m *testing.M) {
	testDBURL := getRequiredEnvVar("TEST_DATABASE_URL")
	testDBName := getDBName(testDBURL)

	ctx := context.Background()

	pgDB := getDBConn(ctx, swapDBName(testDBURL, "postgres"))
	pgDB.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if _, err := pgDB.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", testDBName)); err != nil {
		log.Fatal("could not create test database")
	}

	migrateURL := strings.Replace(testDBURL, "postgres://", "pgx5://", 1)
	mig, err := migrate.New("file://../../migrations", migrateURL)
	if err != nil {
		log.Fatalf("could not create migrate instance: %v", err)
	}
	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("could not migrate test db: %v", err)
	}

	testPool = getDBConn(ctx, testDBURL)

	code := m.Run()

	testPool.Close()
	// Postgres refuses to drop a DB with active connections -- force evictions first
	pgDB.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, testDBName)
	pgDB.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	pgDB.Close()

	os.Exit(code)
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return api.NewRouter(api.RouterConfig{Queries: store.New(testPool)})
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "TRUNCATE links RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func createTestLink(t *testing.T, originalUrl string, reallyLongUrl string) store.Link {
	t.Helper()
	link, err := store.New(testPool).CreateLink(context.Background(), store.CreateLinkParams{
		OriginalUrl:   originalUrl,
		ReallyLongUrl: reallyLongUrl,
	})
	if err != nil {
		t.Fatalf("createTestLink: %v", err)
	}
	return link
}

func getDBName(dbURL string) string {
	u, _ := url.Parse(dbURL)
	return u.Path[1:]
}

func swapDBName(oldDB, newDB string) string {
	u, _ := url.Parse(oldDB)
	u.Path = "/" + newDB
	return u.String()
}

func TestSwapDBName(t *testing.T) {
	expected := "pg://hello:moto@some.place/woof?one=1&two=2"
	if r := swapDBName("pg://hello:moto@some.place/meow?one=1&two=2", "woof"); r != expected {
		t.Errorf("wanted %s but got %s", expected, r)
	}
}

func TestGetDBName(t *testing.T) {
	if r := getDBName("pg://hello:moto@some.place/woof?one=1&two=2"); r != "woof" {
		t.Errorf("wanted woof but got %s", r)
	}
}

func TestListLinks(t *testing.T) {
	truncate(t)

	for i := range 3 {
		createTestLink(t, fmt.Sprintf("https://example.com/%d", i), fmt.Sprintf("https://example.com/reallylongurl/%d", i))
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
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")

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
	if created.ReallyLongUrl == "" {
		t.Errorf("expected really long url; got empty")
	}
	if _, err := store.New(testPool).GetLink(context.Background(), created.ID); err != nil {
		t.Errorf("created link not found in db: %v", err)
	}
}

func TestUpdateLink404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)

	body := strings.NewReader(`{"original_url":"https://example.com","really_long_url":"https://example.com/reallylongurl"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/links/666", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Result().StatusCode)
	}
}

func TestUpdateLink(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")

	srv := newTestServer(t)
	body := strings.NewReader(`{"original_url":"https://example.com","really_long_url":"https://example.com/reallylongurl"}`)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/links/%d", link.ID), body)
	req.Header.Set("Content-Type", "application/json")
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

func TestDeleteLink(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/links/%d", link.ID), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204; got %d", w.Result().StatusCode)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/links/%d", link.ID), nil)
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete; got %d", w.Result().StatusCode)
	}
}

func TestWebCreateLink(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	body := strings.NewReader("original_url=https%3A%2F%2Fexample.com")
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

func TestRedirectToOriginalUrl404(t *testing.T) {
	truncate(t)
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rll/doesnotexist", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Result().StatusCode)
	}
}

func TestRedirectToOriginalUrlWithQueryString(t *testing.T) {
	truncate(t)
	reallyLongUrl := "seg1/seg2/seg3?utm_source=foo&ref=bar&id=baz"
	link := createTestLink(t, "https://example.com", reallyLongUrl)

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rll/"+link.ReallyLongUrl, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected 301; got %d", w.Result().StatusCode)
	}
}

func TestRedirectToOriginalUrl(t *testing.T) {
	truncate(t)
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/rll/%s", link.ReallyLongUrl), nil)
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
	link := createTestLink(t, "https://example.com", "https://example.com/reallylongurl")
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
