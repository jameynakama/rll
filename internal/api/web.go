package api

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/jameynakama/reallylonglink/internal/store"
)

//go:embed templates
var templateFiles embed.FS

var (
	indexTmpl  = template.Must(template.ParseFS(templateFiles, "templates/index.html"))
	resultTmpl = template.Must(template.ParseFS(templateFiles, "templates/result.html"))
)

type indexData struct {
	Error string
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

	row, err := h.queries.CreateLink(r.Context(), store.CreateLinkParams{
		OriginalUrl:   originalUrl,
		ReallyLongUrl: generateReallyLongUrl(),
	})
	if err != nil {
		log.Printf("webCreateLink: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/links/%d", row.ID), http.StatusSeeOther)
}

func (h *Handler) webGetLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
