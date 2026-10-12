package safebrowsing_test

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
)

// Real v5 responses: the API only speaks protobuf.
var (
	// http://testsafebrowsing.appspot.com/s/phishing.html, flagged SOCIAL_ENGINEERING
	flaggedBody = mustHex("0a310a2c746573747361666562726f7773696e672e61707073706f742e636f6d2f732f7068697368696e672e68746d6c120102120308ac02")
	// https://example.com/, cacheDuration only
	cleanBody = mustHex("120308ac02")
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *safebrowsing.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return safebrowsing.New(srv.URL, "test-key")
}

// Should send the url and api key to the v5 search endpoint
func TestIsFlaggedRequest(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/urls:search" {
			t.Errorf("expected path /v5/urls:search; got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("urls"); got != "https://example.com/a?b=c" {
			t.Errorf("expected urls param https://example.com/a?b=c; got %s", got)
		}
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Errorf("expected key param test-key; got %s", got)
		}
		w.Write(cleanBody)
	})

	if _, err := c.IsFlagged(context.Background(), "https://example.com/a?b=c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsFlagged(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want bool
	}{
		{"no threats", cleanBody, false},
		{"empty body", []byte{}, false},
		{"threat", flaggedBody, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Write(tt.body)
			})

			got, err := c.IsFlagged(context.Background(), "https://evil.example/")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v; got %v", tt.want, got)
			}
		})
	}
}

func TestIsFlaggedErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"non-200", http.StatusForbidden, `{"error":{"message":"bad key"}}`},
		{"malformed tag", http.StatusOK, "\xff"},
		{"truncated field", http.StatusOK, "\x12\x05ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			})

			if _, err := c.IsFlagged(context.Background(), "https://example.com"); err == nil {
				t.Error("expected error; got nil")
			}
		})
	}
}

func TestIsFlaggedRateLimited(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	if _, err := c.IsFlagged(context.Background(), "https://example.com"); !errors.Is(err, safebrowsing.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited; got %v", err)
	}
}

func TestIsFlaggedUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c := safebrowsing.New(srv.URL, "test-key")

	if _, err := c.IsFlagged(context.Background(), "https://example.com"); err == nil {
		t.Error("expected error; got nil")
	}
}

func TestIsFlaggedBadBaseURL(t *testing.T) {
	c := safebrowsing.New("://nope", "test-key")

	if _, err := c.IsFlagged(context.Background(), "https://example.com"); err == nil {
		t.Error("expected error; got nil")
	}
}
