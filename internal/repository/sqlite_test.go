package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *SQLite {
	t.Helper()
	// Use shared cache mode for in-memory tests with concurrent access
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := NewSQLite(db)
	if err := repo.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("close repo: %v", err)
		}
	})

	return repo
}

func TestSQLite_Create(t *testing.T) {
	repo := setupTestDB(t)

	url, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if url.ID != 1 {
		t.Errorf("expected ID 1, got %d", url.ID)
	}
	if url.Code == "" || url.Code == "_placeholder_" {
		t.Errorf("expected valid code, got %q", url.Code)
	}
	if url.Original != "https://example.com" {
		t.Errorf("expected original https://example.com, got %q", url.Original)
	}
	if url.Clicks != 0 {
		t.Errorf("expected 0 clicks, got %d", url.Clicks)
	}
	if url.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestSQLite_GetByCode(t *testing.T) {
	repo := setupTestDB(t)

	created, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.GetByCode(created.Code)
	if err != nil {
		t.Fatalf("get by code: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, found.ID)
	}
	if found.Original != created.Original {
		t.Errorf("expected original %q, got %q", created.Original, found.Original)
	}
}

func TestSQLite_GetByCode_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.GetByCode("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLite_GetByOriginal(t *testing.T) {
	repo := setupTestDB(t)

	created, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.GetByOriginal("https://example.com")
	if err != nil {
		t.Fatalf("get by original: %v", err)
	}

	if found.Code != created.Code {
		t.Errorf("expected code %q, got %q", created.Code, found.Code)
	}
}

func TestSQLite_IncrementClicks(t *testing.T) {
	repo := setupTestDB(t)

	created, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	for i := 0; i < 5; i++ {
		if err := repo.IncrementClicks(created.Code); err != nil {
			t.Fatalf("increment clicks: %v", err)
		}
	}

	found, err := repo.GetByCode(created.Code)
	if err != nil {
		t.Fatalf("get by code: %v", err)
	}

	if found.Clicks != 5 {
		t.Errorf("expected 5 clicks, got %d", found.Clicks)
	}
}

func TestSQLite_IncrementClicks_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	err := repo.IncrementClicks("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLite_GlobalStats(t *testing.T) {
	repo := setupTestDB(t)

	for i := 0; i < 3; i++ {
		url, err := repo.Create("https://example.com/" + string(rune('a'+i)))
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		for j := 0; j <= i; j++ {
			if err := repo.IncrementClicks(url.Code); err != nil {
				t.Fatalf("increment clicks: %v", err)
			}
		}
	}

	stats, err := repo.GlobalStats()
	if err != nil {
		t.Fatalf("global stats: %v", err)
	}

	if stats.TotalURLs != 3 {
		t.Errorf("expected 3 total URLs, got %d", stats.TotalURLs)
	}
	if stats.TotalClicks != 6 {
		t.Errorf("expected 6 total clicks, got %d", stats.TotalClicks)
	}
	if stats.URLsToday != 3 {
		t.Errorf("expected 3 URLs today, got %d", stats.URLsToday)
	}
}

func TestSQLite_MultipleURLs(t *testing.T) {
	repo := setupTestDB(t)

	urls := []string{
		"https://example.com",
		"https://google.com",
		"https://github.com",
	}

	codes := make([]string, len(urls))
	for i, u := range urls {
		created, err := repo.Create(u)
		if err != nil {
			t.Fatalf("create %q: %v", u, err)
		}
		codes[i] = created.Code
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate code: %q", code)
		}
		seen[code] = true
	}

	for i, code := range codes {
		found, err := repo.GetByCode(code)
		if err != nil {
			t.Errorf("get code %q: %v", code, err)
			continue
		}
		if found.Original != urls[i] {
			t.Errorf("code %q: expected %q, got %q", code, urls[i], found.Original)
		}
	}
}

func TestSQLite_ConcurrentIncrements(t *testing.T) {
	repo := setupTestDB(t)

	created, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = repo.IncrementClicks(created.Code)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	time.Sleep(100 * time.Millisecond)

	found, err := repo.GetByCode(created.Code)
	if err != nil {
		t.Fatalf("get by code: %v", err)
	}

	if found.Clicks != 10 {
		t.Errorf("expected 10 clicks after concurrent increments, got %d", found.Clicks)
	}
}
