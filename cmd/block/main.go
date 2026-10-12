// Command block adds a domain to the blocklist and deletes every existing
// link to it or its subdomains: block bad-site.example
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/blocklist"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: block <domain>")
		os.Exit(2)
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("must set DATABASE_URL env var")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("error establishing database connection")
	}
	defer db.Close()

	deleted, err := blocklist.Block(ctx, db, os.Args[1])
	if err != nil {
		log.Fatalf("block: %v", err)
	}
	fmt.Printf("blocked %s, deleted %d existing links\n", os.Args[1], deleted)
}
