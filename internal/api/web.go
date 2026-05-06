package api

import (
	"embed"
	"html/template"
	"net/http"
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
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) webCreateLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) webGetLink(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
