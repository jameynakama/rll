// Package sweep re-checks recent links against Safe Browsing. New phishing
// domains often aren't listed yet when a link is created, so the create-time
// check alone lets them through.
package sweep

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
)

// CheckAges are the link ages, in days, at which a link gets re-checked.
// Re-checking a whole window nightly doesn't fit Safe Browsing's 1,000/day
// quota; a few checkpoints cover the days when new phishing gets listed.
var CheckAges = []int{1, 3, 7}

// Run checks each distinct url from links created (UTC) CheckAges days before
// now, waiting pace between checks, and deletes every link to a flagged url.
// It returns how many links it deleted. A url whose check errors is kept and
// logged; a rate limit stops the run, since every later check would fail too
// and new links need the remaining quota.
func Run(ctx context.Context, q *store.Queries, sb *safebrowsing.Client, now time.Time, pace time.Duration) (int64, error) {
	today := now.UTC().Truncate(24 * time.Hour)
	days := make([]pgtype.Date, len(CheckAges))
	for i, age := range CheckAges {
		days[i] = pgtype.Date{Time: today.AddDate(0, 0, -age), Valid: true}
	}

	urls, err := q.ListURLsCreatedOn(ctx, days)
	if err != nil {
		return 0, fmt.Errorf("list urls: %w", err)
	}
	log.Printf("sweep: checking %d urls", len(urls))

	var deleted int64
	for i, u := range urls {
		if i > 0 {
			time.Sleep(pace)
		}
		flagged, err := sb.IsFlagged(ctx, u)
		if errors.Is(err, safebrowsing.ErrRateLimited) {
			return deleted, fmt.Errorf("stopped after %d of %d urls: %w", i, len(urls), err)
		}
		if err != nil {
			log.Printf("sweep: check %s: %v", u, err)
			continue
		}
		if !flagged {
			continue
		}
		n, err := q.DeleteLinksByOriginalUrl(ctx, u)
		if err != nil {
			return deleted, fmt.Errorf("delete links to %s: %w", u, err)
		}
		log.Printf("sweep: deleted %d links -> %s", n, u)
		deleted += n
	}
	return deleted, nil
}
