package sweep_test

import (
	"context"
	"crypto/md5"
	"errors"
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

const day = 24 * time.Hour

// fakeSafeBrowsing flags any url containing "evil", errors on "broken", and
// rate-limits on "quota". It records every url it's asked about.
type fakeSafeBrowsing struct {
	checked []string
}

func (f *fakeSafeBrowsing) client(t *testing.T) *safebrowsing.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("urls")
		f.checked = append(f.checked, u)
		switch {
		case strings.Contains(u, "quota"):
			w.WriteHeader(http.StatusTooManyRequests)
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

// Should check only links at the check ages and delete every link to a flagged url
func TestRun(t *testing.T) {
	truncate(t)
	evilAt1 := createLink(t, "https://evil.example/1", 1*day)
	evilAt3 := createLink(t, "https://evil.example/3", 3*day)
	evilAt7 := createLink(t, "https://evil.example/7", 7*day)
	evilAt7Old := createLink(t, "https://evil.example/7", 40*day)
	evilAt2 := createLink(t, "https://evil.example/2", 2*day)
	cleanAt1 := createLink(t, "https://example.com", 1*day)
	brokenAt1 := createLink(t, "https://broken.example", 1*day)

	sb := &fakeSafeBrowsing{}
	deleted, err := sweep.Run(context.Background(), store.New(testPool), sb.client(t), time.Now(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deleted != 4 {
		t.Errorf("expected 4 deleted; got %d", deleted)
	}
	for _, l := range []store.Link{evilAt1, evilAt3, evilAt7, evilAt7Old} {
		if linkExists(t, l.ID) {
			t.Errorf("expected link %d (%s) to be deleted", l.ID, l.OriginalUrl)
		}
	}
	for _, l := range []store.Link{evilAt2, cleanAt1, brokenAt1} {
		if !linkExists(t, l.ID) {
			t.Errorf("expected link %d (%s) to be kept", l.ID, l.OriginalUrl)
		}
	}
	if len(sb.checked) != 5 {
		t.Errorf("expected 5 urls checked (each once, none outside the check ages); got %v", sb.checked)
	}
}

// Should stop at the first rate limit instead of burning through the rest
func TestRunStopsWhenRateLimited(t *testing.T) {
	truncate(t)
	createLink(t, "https://a-quota.example", 1*day)
	createLink(t, "https://b-evil.example", 1*day)

	sb := &fakeSafeBrowsing{}
	_, err := sweep.Run(context.Background(), store.New(testPool), sb.client(t), time.Now(), 0)

	if !errors.Is(err, safebrowsing.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited; got %v", err)
	}
	if len(sb.checked) != 1 {
		t.Errorf("expected to stop after 1 check; got %v", sb.checked)
	}
}

// Should wait the pace between checks
func TestRunPaces(t *testing.T) {
	truncate(t)
	createLink(t, "https://a.example", 1*day)
	createLink(t, "https://b.example", 1*day)
	createLink(t, "https://c.example", 1*day)

	sb := &fakeSafeBrowsing{}
	start := time.Now()
	if _, err := sweep.Run(context.Background(), store.New(testPool), sb.client(t), time.Now(), 50*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 2 paces (100ms) for 3 checks; took %v", elapsed)
	}
}

func TestRunDBError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sb := &fakeSafeBrowsing{}
	if _, err := sweep.Run(ctx, store.New(testPool), sb.client(t), time.Now(), 0); err == nil {
		t.Error("expected error; got nil")
	}
}
