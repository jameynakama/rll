package blocklist_test

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/blocklist"
	"github.com/jameynakama/reallylonglink/internal/store"
	"github.com/jameynakama/reallylonglink/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, drop := testdb.Create("blocklist")
	testPool = pool
	code := m.Run()
	drop()
	os.Exit(code)
}

func truncate(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), "TRUNCATE links, blocked_domains RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func createLink(t *testing.T, originalUrl string) store.Link {
	t.Helper()
	link, err := store.New(testPool).CreateLink(context.Background(), store.CreateLinkParams{
		OriginalUrl:     originalUrl,
		ReallyLongPath:  originalUrl,
		ReallyLongQuery: "",
		PathHash:        fmt.Sprintf("%x", md5.Sum([]byte(originalUrl))),
	})
	if err != nil {
		t.Fatalf("createLink: %v", err)
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

// Should block the domain and delete existing links to it and its subdomains only
func TestBlock(t *testing.T) {
	truncate(t)
	doomed := []store.Link{
		createLink(t, "https://bad-site.example/a"),
		createLink(t, "https://www.bad-site.example/campaign/abc123/?utm_campaign=x"),
		createLink(t, "http://WWW.BAD-SITE.EXAMPLE:8080/b"),
		createLink(t, "https://user@bad-site.example/c"),
	}
	spared := []store.Link{
		createLink(t, "https://notbad-site.example/a"),
		createLink(t, "https://bad-site.example.evil.example/a"),
		createLink(t, "https://example.com/?next=https://bad-site.example"),
	}

	deleted, err := blocklist.Block(context.Background(), testPool, " Bad-site.example. ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deleted != int64(len(doomed)) {
		t.Errorf("expected %d deleted; got %d", len(doomed), deleted)
	}
	for _, l := range doomed {
		if linkExists(t, l.ID) {
			t.Errorf("expected %s to be deleted", l.OriginalUrl)
		}
	}
	for _, l := range spared {
		if !linkExists(t, l.ID) {
			t.Errorf("expected %s to be kept", l.OriginalUrl)
		}
	}

	blocked, err := store.New(testPool).IsDomainBlocked(context.Background(), "bad-site.example")
	if err != nil {
		t.Fatalf("IsDomainBlocked: %v", err)
	}
	if !blocked {
		t.Error("expected bad-site.example to be stored normalized and blocked")
	}
}

// Should be a no-op when the domain is already blocked
func TestBlockTwice(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	if _, err := blocklist.Block(ctx, testPool, "bad-site.example"); err != nil {
		t.Fatalf("first block: %v", err)
	}

	deleted, err := blocklist.Block(ctx, testPool, "bad-site.example")
	if err != nil {
		t.Fatalf("second block: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted; got %d", deleted)
	}
}

func TestBlockRejectsNonDomains(t *testing.T) {
	for _, in := range []string{"", "   ", "https://bad-site.example", "bad-site.example/x"} {
		t.Run(in, func(t *testing.T) {
			if _, err := blocklist.Block(context.Background(), testPool, in); err == nil {
				t.Errorf("expected error for %q; got nil", in)
			}
		})
	}
}

func TestIsBlocked(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	if _, err := blocklist.Block(ctx, testPool, "bad-site.example"); err != nil {
		t.Fatalf("block: %v", err)
	}

	tests := []struct {
		url  string
		want bool
	}{
		{"https://bad-site.example/x", true},
		{"https://www.Bad-site.example/x", true},
		{"https://notbad-site.example/x", false},
		{"https://example.com/?u=bad-site.example", false},
		{"not a url", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := blocklist.IsBlocked(ctx, store.New(testPool), tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v; got %v", tt.want, got)
			}
		})
	}
}
