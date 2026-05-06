package api

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jameynakama/reallylonglink/internal/store"
)

const defaultLimit = 20
const availableChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~/"
const maxURLLength = 2048

type createLinkRequest struct {
	OriginalUrl string `json:"original_url"`
}

type updateLinkRequest struct {
	OriginalUrl   string `json:"original_url"`
	ReallyLongUrl string `json:"really_long_url"`
}

func (h *Handler) listLinks(w http.ResponseWriter, r *http.Request) {
	limit := int32(defaultLimit)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = int32(v)
		}
	}

	offset := int32(0)
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = int32(v)
		}
	}

	rows, err := h.queries.ListLinks(r.Context(), store.ListLinksParams{Limit: limit, Offset: offset})
	if err != nil {
		log.Printf("listLinks: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	if rows == nil {
		rows = []store.Link{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) getLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	row, err := h.queries.GetLink(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("getLink: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) createLink(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OriginalUrl == "" {
		writeError(w, http.StatusBadRequest, "original url is required")
		return
	}

	reallyLongUrl := make([]byte, maxURLLength)
	for i := range reallyLongUrl {
		reallyLongUrl[i] = availableChars[rand.Intn(len(availableChars))]
	}

	row, err := h.queries.CreateLink(r.Context(), store.CreateLinkParams{
		OriginalUrl:   req.OriginalUrl,
		ReallyLongUrl: string(reallyLongUrl),
	})
	if err != nil {
		log.Printf("createLink: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, row)
}

func (h *Handler) updateLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	var req updateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OriginalUrl == "" {
		writeError(w, http.StatusBadRequest, "original url is required")
		return
	}

	row, err := h.queries.UpdateLink(r.Context(), store.UpdateLinkParams{
		ID:            id,
		OriginalUrl:   req.OriginalUrl,
		ReallyLongUrl: req.ReallyLongUrl,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("updateLink: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) deleteLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	if err := h.queries.DeleteLink(r.Context(), id); err != nil {
		log.Printf("deleteLink: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) redirectToOriginalUrl(w http.ResponseWriter, r *http.Request) {
	rawLink, _ := strings.CutPrefix(r.RequestURI, "/api/v1/rll/")
	reallyLongLink, err := url.PathUnescape(rawLink)
	if err != nil {
		log.Printf("redirectToOriginalUrl: unescape: %v", err)
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	row, err := h.queries.GetLinkByReallyLongUrl(r.Context(), reallyLongLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("redirectToOriginalUrl: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	http.Redirect(w, r, row.OriginalUrl, http.StatusMovedPermanently)
}
