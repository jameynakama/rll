package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jameynakama/reallylonglink/internal/blocklist"
	"github.com/jameynakama/reallylonglink/internal/safebrowsing"
	"github.com/jameynakama/reallylonglink/internal/store"
)

type RouterConfig struct {
	Queries *store.Queries
	// SafeBrowsing is nil when no API key is configured; links go unchecked.
	SafeBrowsing *safebrowsing.Client
}

type Handler struct {
	queries      *store.Queries
	safeBrowsing *safebrowsing.Client
}

func NewRouter(cfg RouterConfig) http.Handler {
	h := &Handler{
		queries:      cfg.Queries,
		safeBrowsing: cfg.SafeBrowsing,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.healthCheck)

	r.Get("/", h.webIndex)
	r.Post("/", h.webCreateLink)
	r.Get("/links/{id}", h.webGetLink)
	r.Get("/rll/*", h.redirectToOriginalUrl)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/links", h.listLinks)
		r.Get("/rll/*", h.redirectToOriginalUrl)
		r.Post("/links", h.createLink)
		r.Get("/links/{id}", h.getLink)
	})

	return r
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// isUnsafe checks our blocklist, then Safe Browsing. A blocklist error is ours
// and is returned; Safe Browsing fails open, since if it can't answer the
// nightly sweep gets another look at the link.
func (h *Handler) isUnsafe(ctx context.Context, originalUrl string) (bool, error) {
	blocked, err := blocklist.IsBlocked(ctx, h.queries, originalUrl)
	if err != nil || blocked {
		return blocked, err
	}
	if h.safeBrowsing == nil {
		return false, nil
	}
	flagged, err := h.safeBrowsing.IsFlagged(ctx, originalUrl)
	if err != nil {
		log.Printf("isUnsafe: safe browsing: %v", err)
		return false, nil
	}
	return flagged, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
