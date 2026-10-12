package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/reallylonglink/internal/api"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
)

type config struct {
	databaseURL        string
	port               string
	safeBrowsingAPIKey string
}

func loadConfig() config {
	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			log.Fatalf("must set %s env var", key)
		}
		return v
	}

	withDefault := func(key string, defaultV string) string {
		v := os.Getenv(key)
		if v == "" {
			return defaultV
		}
		return v
	}

	return config{
		databaseURL:        required("DATABASE_URL"),
		port:               withDefault("PORT", "8080"),
		safeBrowsingAPIKey: os.Getenv("SAFE_BROWSING_API_KEY"),
	}
}

func main() {
	cfg := loadConfig()

	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		log.Fatal("error establishing database connection")
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal("cannot ping database")
	}
	log.Println("database connected")

	routerCfg := api.RouterConfig{Queries: store.New(db)}
	if cfg.safeBrowsingAPIKey != "" {
		routerCfg.SafeBrowsing = safebrowsing.New(safebrowsing.DefaultBaseURL, cfg.safeBrowsingAPIKey)
	} else {
		log.Println("SAFE_BROWSING_API_KEY not set; new links won't be checked")
	}
	r := api.NewRouter(routerCfg)

	log.Printf("starting server at http://localhost:%s", cfg.port)
	if err := http.ListenAndServe(":"+cfg.port, r); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
