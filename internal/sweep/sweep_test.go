package sweep_test

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
	"github.com/jameynakama/reallylonglink/internal/sweep"
	"github.com/jameynakama/reallylonglink/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, drop := testdb.Create("sweep")
	testPool = pool
	code := m.Run()
	drop()
	os.Exit(code)
}

func truncate(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), "TRUNCATE links RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func createLink(t *testing.T, originalUrl string, age time.Duration) store.Link {
	t.Helper()
	path := fmt.Sprintf("path/%d", time.Now().UnixNano())
	link, err := store.New(testPool).CreateLink(context.Background(), store.CreateLinkParams{
		OriginalUrl:     originalUrl,
		ReallyLongPath:  path,
		ReallyLongQuery: "",
		PathHash:        fmt.Sprintf("%x", md5.Sum([]byte(path))),
	})
	if err != nil {
		t.Fatalf("createLink: %v", err)
	}
	if _, err := testPool.Exec(context.Background(), "UPDATE links SET create_time = NOW() - $1::interval WHERE id = $2", age.String(), link.ID); err != nil {
		t.Fatalf("createLink: backdate: %v", err)
	}
	return link
}

func linkExists(t *testing.T, id int64) bool {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(), "SELECT count(*) FROM links WHERE id = $1", id).Scan(&n); err != nil {
		t.Fatalf("linkExists: %v", err)
	}
	return n == 1
}

// newSafeBrowsing flags any url containing "evil" and errors on any containing "broken".
func newSafeBrowsing(t *testing.T) *safebrowsing.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("urls")
		switch {
		case strings.Contains(u, "broken"):
			w.WriteHeader(http.StatusInternalServerError)
		case strings.Contains(u, "evil"):
			// protobuf SearchUrlsResponse with one (empty) threat
			w.Write([]byte{0x0a, 0x00})
		}
	}))
	t.Cleanup(srv.Close)
	return safebrowsing.New(srv.URL, "test-key")
}

// Should delete recent flagged links and leave clean, old, and unchecked ones
func TestRun(t *testing.T) {
	truncate(t)
	recentEvil := createLink(t, "https://evil.example/a", 24*time.Hour)
	recentClean := createLink(t, "https://example.com", 24*time.Hour)
	recentBroken := createLink(t, "https://broken.example", 24*time.Hour)
	oldEvil := createLink(t, "https://evil.example/b", 60*24*time.Hour)

	deleted, err := sweep.Run(context.Background(), store.New(testPool), newSafeBrowsing(t), time.Now().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted; got %d", deleted)
	}
	if linkExists(t, recentEvil.ID) {
		t.Error("expected recent flagged link to be deleted")
	}
	if !linkExists(t, recentClean.ID) {
		t.Error("expected recent clean link to be kept")
	}
	if !linkExists(t, recentBroken.ID) {
		t.Error("expected link whose check errored to be kept")
	}
	if !linkExists(t, oldEvil.ID) {
		t.Error("expected link older than the window to be left alone")
	}
}

func TestRunDBError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := sweep.Run(ctx, store.New(testPool), newSafeBrowsing(t), time.Now()); err == nil {
		t.Error("expected error; got nil")
	}
}
