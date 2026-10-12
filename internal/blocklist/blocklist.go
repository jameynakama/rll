// Package blocklist keeps domains we never want to link to. It catches repeat
// offenders that Safe Browsing hasn't listed.
package blocklist

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/store"
)

// IsBlocked reports whether the url's host is a blocked domain or one of its
// subdomains. A url with no parseable host is never blocked.
func IsBlocked(ctx context.Context, q *store.Queries, rawURL string) (bool, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return false, nil
	}
	return q.IsDomainBlocked(ctx, strings.ToLower(u.Hostname()))
}

// Block adds domain to the blocklist and deletes every existing link to it or
// its subdomains, returning how many links it deleted.
func Block(ctx context.Context, db *pgxpool.Pool, domain string) (int64, error) {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" || strings.ContainsAny(domain, "/:") {
		return 0, errors.New("pass a bare domain, like bad-site.example")
	}

	var deleted int64
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.BlockDomain(ctx, domain); err != nil {
			return fmt.Errorf("block domain: %w", err)
		}
		n, err := q.DeleteLinksForDomain(ctx, domain)
		if err != nil {
			return fmt.Errorf("delete links: %w", err)
		}
		deleted = n
		return nil
	})
	return deleted, err
}
