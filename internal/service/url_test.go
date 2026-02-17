package service

import (
	"errors"
	"testing"
	"time"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/repository"
)

type mockRepo struct {
	urls      map[string]*domain.URL
	byCode    map[string]*domain.URL
	nextID    int64
	createErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		urls:   make(map[string]*domain.URL),
		byCode: make(map[string]*domain.URL),
		nextID: 1,
	}
}

func (m *mockRepo) Create(original string) (*domain.URL, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	url := &domain.URL{
		ID:        m.nextID,
		Code:      "test" + string(rune('a'+m.nextID-1)),
		Original:  original,
		Clicks:    0,
		CreatedAt: time.Now(),
	}
	m.nextID++
	m.urls[original] = url
	m.byCode[url.Code] = url
	return url, nil
}

func (m *mockRepo) GetByCode(code string) (*domain.URL, error) {
	if url, ok := m.byCode[code]; ok {
		return url, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) GetByOriginal(original string) (*domain.URL, error) {
	if url, ok := m.urls[original]; ok {
		return url, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) IncrementClicks(code string) error {
	if url, ok := m.byCode[code]; ok {
		url.Clicks++
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockRepo) GlobalStats() (*domain.GlobalStats, error) {
	var totalClicks int64
	for _, url := range m.urls {
		totalClicks += url.Clicks
	}
	return &domain.GlobalStats{
		TotalURLs:   int64(len(m.urls)),
		TotalClicks: totalClicks,
		URLsToday:   int64(len(m.urls)),
	}, nil
}

func TestURLService_Shorten(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	resp, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}

	if resp.Code == "" {
		t.Error("expected non-empty code")
	}
	if resp.ShortURL == "" {
		t.Error("expected non-empty short URL")
	}
	if resp.ShortURL != "http://localhost:8080/"+resp.Code {
		t.Errorf("expected short URL http://localhost:8080/%s, got %s", resp.Code, resp.ShortURL)
	}
}

func TestURLService_Shorten_Duplicate(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	resp1, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("first shorten: %v", err)
	}

	resp2, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("second shorten: %v", err)
	}

	if resp1.Code != resp2.Code {
		t.Errorf("duplicate URL should return same code: %s vs %s", resp1.Code, resp2.Code)
	}
}

func TestURLService_Shorten_InvalidURL(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"empty", "", ErrEmptyURL},
		{"no scheme", "example.com", ErrMissingScheme},
		{"ftp scheme", "ftp://example.com", ErrMissingScheme},
		{"no host", "http://", ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Shorten(tt.url)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestURLService_Resolve(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	resp, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}

	original, err := svc.Resolve(resp.Code)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if original != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", original)
	}
}

func TestURLService_Resolve_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	_, err := svc.Resolve("nonexistent")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestURLService_Resolve_Empty(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	_, err := svc.Resolve("")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound for empty code, got %v", err)
	}
}

func TestURLService_Stats(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	resp, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}

	stats, err := svc.Stats(resp.Code)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if stats.Code != resp.Code {
		t.Errorf("expected code %s, got %s", resp.Code, stats.Code)
	}
	if stats.Original != "https://example.com" {
		t.Errorf("expected original https://example.com, got %s", stats.Original)
	}
	if stats.Clicks != 0 {
		t.Errorf("expected 0 clicks, got %d", stats.Clicks)
	}
}

func TestURLService_Stats_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	_, err := svc.Stats("nonexistent")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestURLService_GlobalStats(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	_, _ = svc.Shorten("https://example1.com")
	_, _ = svc.Shorten("https://example2.com")

	stats, err := svc.GlobalStats()
	if err != nil {
		t.Fatalf("global stats: %v", err)
	}

	if stats.TotalURLs != 2 {
		t.Errorf("expected 2 total URLs, got %d", stats.TotalURLs)
	}
}

func TestURLService_BaseURLTrailingSlash(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080/")

	resp, err := svc.Shorten("https://example.com")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}

	expected := "http://localhost:8080/" + resp.Code
	if resp.ShortURL != expected {
		t.Errorf("expected %s, got %s", expected, resp.ShortURL)
	}
}

func TestURLService_ValidURLs(t *testing.T) {
	repo := newMockRepo()
	svc := NewURLService(repo, "http://localhost:8080")

	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://example.com/path",
		"https://example.com/path?query=value",
		"https://example.com:8080/path",
		"https://sub.domain.example.com",
		"https://example.com/path#fragment",
	}

	for _, u := range validURLs {
		t.Run(u, func(t *testing.T) {
			_, err := svc.Shorten(u)
			if err != nil {
				t.Errorf("expected valid URL %s to succeed, got error: %v", u, err)
			}
		})
	}
}
