package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/repository"
	"github.com/devaloi/shrink/internal/service"
)

func setupTestHandler(t *testing.T) (*Handler, func()) {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := repository.NewSQLite(db)
	if err := repo.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := New(svc)

	cleanup := func() {
		_ = db.Close()
	}

	return h, cleanup
}

func TestHandler_CreateShortURL(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateShortURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp domain.CreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code == "" {
		t.Error("expected non-empty code")
	}
	if !strings.HasPrefix(resp.ShortURL, "http://localhost:8080/") {
		t.Errorf("expected short URL to start with base URL, got %s", resp.ShortURL)
	}
}

func TestHandler_CreateShortURL_InvalidJSON(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateShortURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_CreateShortURL_EmptyURL(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateShortURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_CreateShortURL_InvalidScheme(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":"ftp://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateShortURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_CreateShortURL_MethodNotAllowed(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/shorten", nil)
	w := httptest.NewRecorder()

	h.CreateShortURL(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandler_Redirect(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":"https://example.com"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.CreateShortURL(createW, createReq)

	var createResp domain.CreateResponse
	if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	redirectReq := httptest.NewRequest(http.MethodGet, "/"+createResp.Code, nil)
	redirectW := httptest.NewRecorder()
	h.Redirect(redirectW, redirectReq)

	if redirectW.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 301, got %d", redirectW.Code)
	}

	location := redirectW.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected Location https://example.com, got %s", location)
	}
}

func TestHandler_Redirect_NotFound(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Redirect(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandler_GetStats(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":"https://example.com"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.CreateShortURL(createW, createReq)

	var createResp domain.CreateResponse
	if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+createResp.Code, nil)
	statsW := httptest.NewRecorder()
	h.GetStats(statsW, statsReq)

	if statsW.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", statsW.Code)
	}

	var stats domain.StatsResponse
	if err := json.NewDecoder(statsW.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}

	if stats.Code != createResp.Code {
		t.Errorf("expected code %s, got %s", createResp.Code, stats.Code)
	}
	if stats.Original != "https://example.com" {
		t.Errorf("expected original https://example.com, got %s", stats.Original)
	}
}

func TestHandler_GetStats_NotFound(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/urls/nonexistent", nil)
	w := httptest.NewRecorder()

	h.GetStats(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandler_GlobalStats(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		body := bytes.NewBufferString(`{"url":"https://example.com/` + string(rune('a'+i)) + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.CreateShortURL(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	h.GlobalStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var stats domain.GlobalStats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("decode global stats response: %v", err)
	}

	if stats.TotalURLs != 3 {
		t.Errorf("expected 3 total URLs, got %d", stats.TotalURLs)
	}
}

func TestHandler_HealthCheck(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var health domain.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("expected status ok, got %s", health.Status)
	}
}

func TestHandler_FullFlow(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	createBody := `{"url":"https://github.com"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.CreateShortURL(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected status 201, got %d", createW.Code)
	}

	var createResp domain.CreateResponse
	if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	redirectReq := httptest.NewRequest(http.MethodGet, "/"+createResp.Code, nil)
	redirectW := httptest.NewRecorder()
	h.Redirect(redirectW, redirectReq)

	if redirectW.Code != http.StatusMovedPermanently {
		t.Fatalf("redirect: expected status 301, got %d", redirectW.Code)
	}

	time.Sleep(100 * time.Millisecond)

	statsReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+createResp.Code, nil)
	statsW := httptest.NewRecorder()
	h.GetStats(statsW, statsReq)

	if statsW.Code != http.StatusOK {
		t.Fatalf("stats: expected status 200, got %d", statsW.Code)
	}

	var stats domain.StatsResponse
	if err := json.NewDecoder(statsW.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}

	if stats.Clicks != 1 {
		t.Errorf("expected 1 click after redirect, got %d", stats.Clicks)
	}
}

func TestHandler_DuplicateURL(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"url":"https://example.com"}`

	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.CreateShortURL(w1, req1)

	var resp1 domain.CreateResponse
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("decode response 1: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.CreateShortURL(w2, req2)

	var resp2 domain.CreateResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode response 2: %v", err)
	}

	if resp1.Code != resp2.Code {
		t.Errorf("duplicate URL should return same code: %s vs %s", resp1.Code, resp2.Code)
	}
}

func TestHandler_LongURL(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	longPath := strings.Repeat("a", 2000)
	body := `{"url":"https://example.com/` + longPath + `"}`

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateShortURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201 for long URL, got %d", w.Code)
	}
}

func TestHandler_SpecialCharactersURL(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	testURLs := []string{
		"https://example.com/path?q=hello%20world",
		"https://example.com/path#section",
		"https://example.com/path?a=1&b=2",
	}

	for _, u := range testURLs {
		t.Run(u, func(t *testing.T) {
			body, _ := json.Marshal(domain.CreateRequest{URL: u})
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.CreateShortURL(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201, got %d", w.Code)
			}
		})
	}
}
