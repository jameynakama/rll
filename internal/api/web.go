package api

import (
	"crypto/md5"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jameynakama/reallylonglink/internal/store"
	"github.com/jameynakama/reallylonglink/internal/urlgen"
)

//go:embed templates
var templateFiles embed.FS

var (
	indexTmpl  = template.Must(template.ParseFS(templateFiles, "templates/index.html"))
	resultTmpl = template.Must(template.ParseFS(templateFiles, "templates/result.html"))
)

type indexData struct {
	Error string
	Input string
}

type resultData struct {
	OriginalUrl string
	RedirectURL string
}

func (h *Handler) webIndex(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, indexData{}); err != nil {
		log.Printf("webIndex: %v", err)
	}
}

func (h *Handler) webCreateLink(w http.ResponseWriter, r *http.Request) {
	originalUrl := r.FormValue("original_url")

	if originalUrl == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := indexTmpl.Execute(w, indexData{Error: "original url is required"}); err != nil {
			log.Printf("webCreateLink: render: %v", err)
		}
		return
	}

	if !strings.HasPrefix(originalUrl, "http://") && !strings.HasPrefix(originalUrl, "https://") {
		originalUrl = "http://" + originalUrl
	}
	u, err := url.ParseRequestURI(originalUrl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.Contains(u.Host, ".") || u.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := indexTmpl.Execute(w, indexData{Error: "please provide a url", Input: originalUrl}); err != nil {
			log.Printf("webCreateLink: render: %v", err)
		}
		return
	}

	path, query := urlgen.Generate()
	row, err := h.queries.CreateLink(r.Context(), store.CreateLinkParams{
		OriginalUrl:     originalUrl,
		ReallyLongPath:  path,
		ReallyLongQuery: query,
		PathHash:        fmt.Sprintf("%x", md5.Sum([]byte(path))),
	})
	if err != nil {
		log.Printf("webCreateLink: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/links/%d", row.ID), http.StatusSeeOther)
}

func (h *Handler) webGetLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	row, err := h.queries.GetLink(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("webGetLink: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	redirectURL := fmt.Sprintf("%s://%s/rll/%s%s", scheme, r.Host, row.ReallyLongPath, row.ReallyLongQuery)

	if err := resultTmpl.Execute(w, resultData{
		OriginalUrl: row.OriginalUrl,
		RedirectURL: redirectURL,
	}); err != nil {
		log.Printf("webGetLink: render: %v", err)
	}
}
