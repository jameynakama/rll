// Package sweep re-checks recent links against Safe Browsing. New phishing
// domains often aren't listed yet when a link is created, so the create-time
// check alone lets them through.
package sweep

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
)

// Run deletes links created since `since` that Safe Browsing now flags and
// returns how many it deleted. A link whose check errors is kept and logged.
func Run(ctx context.Context, q *store.Queries, sb *safebrowsing.Client, since time.Time) (int, error) {
	links, err := q.ListLinksSince(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("list links: %w", err)
	}

	deleted := 0
	for _, link := range links {
		flagged, err := sb.IsFlagged(ctx, link.OriginalUrl)
		if err != nil {
			log.Printf("sweep: check link %d: %v", link.ID, err)
			continue
		}
		if !flagged {
			continue
		}
		if err := q.DeleteLink(ctx, link.ID); err != nil {
			return deleted, fmt.Errorf("delete link %d: %w", link.ID, err)
		}
		log.Printf("sweep: deleted link %d -> %s", link.ID, link.OriginalUrl)
		deleted++
	}
	return deleted, nil
}
