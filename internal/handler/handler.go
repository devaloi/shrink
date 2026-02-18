package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/repository"
	"github.com/devaloi/shrink/internal/service"
)

// maxRequestBodySize limits the size of incoming request bodies (1 MB).
const maxRequestBodySize = 1 << 20

// Handler handles HTTP requests for the URL shortener.
type Handler struct {
	svc       *service.URLService
	db        *sql.DB
	startTime time.Time
}

// New creates a new Handler with the given service and database connection.
func New(svc *service.URLService, db *sql.DB) *Handler {
	return &Handler{
		svc:       svc,
		db:        db,
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
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.svc.Shorten(req.URL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyURL):
			writeError(w, http.StatusBadRequest, "url is required")
		case errors.Is(err, service.ErrURLTooLong):
			writeError(w, http.StatusBadRequest, "url exceeds maximum length")
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
	code := r.PathValue("code")
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
	code := r.PathValue("code")
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
	status := "ok"
	if err := h.db.Ping(); err != nil {
		status = "degraded"
	}
	uptime := time.Since(h.startTime).Round(time.Second)
	writeJSON(w, http.StatusOK, domain.HealthResponse{
		Status: status,
		Uptime: uptime.String(),
	})
}
