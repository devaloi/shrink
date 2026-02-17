package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/repository"
	"github.com/devaloi/shrink/internal/service"
)

// Handler handles HTTP requests for the URL shortener.
type Handler struct {
	svc       *service.URLService
	startTime time.Time
}

// New creates a new Handler with the given service.
func New(svc *service.URLService) *Handler {
	return &Handler{
		svc:       svc,
		startTime: time.Now(),
	}
}

// CreateShortURL handles POST /api/shorten
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req domain.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.svc.Shorten(req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyURL):
			writeError(w, http.StatusBadRequest, "url is required")
		case errors.Is(err, service.ErrMissingScheme):
			writeError(w, http.StatusBadRequest, "url must have http or https scheme")
		case errors.Is(err, service.ErrInvalidURL):
			writeError(w, http.StatusBadRequest, "invalid url")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create short url")
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Redirect handles GET /{code}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	originalURL, err := h.svc.Resolve(code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short url not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve url")
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

// GetStats handles GET /api/urls/{code}
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/api/urls/")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	stats, err := h.svc.Stats(code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short url not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GlobalStats handles GET /api/stats
func (h *Handler) GlobalStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GlobalStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get global stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// HealthCheck handles GET /api/health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime).Round(time.Second)
	writeJSON(w, http.StatusOK, domain.HealthResponse{
		Status: "ok",
		Uptime: uptime.String(),
	})
}
