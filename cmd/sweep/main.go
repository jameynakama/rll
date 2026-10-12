// Command sweep deletes recent links that Safe Browsing has since flagged.
// Run it daily; see deploy/rll-sweep.timer.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
	"github.com/jameynakama/reallylonglink/internal/sweep"
)

// Phishing domains tend to get listed within days; older links have had
// their chance.
const window = 30 * 24 * time.Hour

func required(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("must set %s env var", key)
	}
	return v
}

func main() {
	databaseURL := required("DATABASE_URL")
	apiKey := required("SAFE_BROWSING_API_KEY")

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("error establishing database connection")
	}
	defer db.Close()

	sb := safebrowsing.New(safebrowsing.DefaultBaseURL, apiKey)
	deleted, err := sweep.Run(ctx, store.New(db), sb, time.Now().Add(-window))
	if err != nil {
		log.Fatalf("sweep: %v", err)
	}
	log.Printf("sweep: done, deleted %d", deleted)
}
