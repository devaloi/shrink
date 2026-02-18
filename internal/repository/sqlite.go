package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/encoding"
)

// SQLite implements the Repository interface using SQLite.
type SQLite struct {
	db *sql.DB
}

// NewSQLite creates a new SQLite repository with the given database connection.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

// Migrate runs the database migrations.
func (r *SQLite) Migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			original TEXT NOT NULL,
			clicks INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_urls_code ON urls(code);
		CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
	`
	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Create inserts a new URL and returns it with the generated short code.
func (r *SQLite) Create(original string) (*domain.URL, error) {
	result, err := r.db.Exec(
		"INSERT INTO urls (code, original) VALUES (?, ?)",
		"_placeholder_", original,
	)
	if err != nil {
		return nil, fmt.Errorf("create url: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	code := encoding.Encode(id)

	_, err = r.db.Exec("UPDATE urls SET code = ? WHERE id = ?", code, id)
	if err != nil {
		return nil, fmt.Errorf("update code: %w", err)
	}

	return r.GetByID(id)
}

// getURL executes a query that returns a single URL row.
func (r *SQLite) getURL(query string, arg any) (*domain.URL, error) {
	url := &domain.URL{}
	err := r.db.QueryRow(query, arg).Scan(
		&url.ID, &url.Code, &url.Original, &url.Clicks, &url.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return url, nil
}

// GetByID retrieves a URL by its database ID.
func (r *SQLite) GetByID(id int64) (*domain.URL, error) {
	return r.getURL("SELECT id, code, original, clicks, created_at FROM urls WHERE id = ?", id)
}

// GetByCode retrieves a URL by its short code.
func (r *SQLite) GetByCode(code string) (*domain.URL, error) {
	return r.getURL("SELECT id, code, original, clicks, created_at FROM urls WHERE code = ?", code)
}

// GetByOriginal retrieves a URL by its original URL if it exists.
func (r *SQLite) GetByOriginal(original string) (*domain.URL, error) {
	return r.getURL("SELECT id, code, original, clicks, created_at FROM urls WHERE original = ?", original)
}

// IncrementClicks increases the click count for a URL by 1.
func (r *SQLite) IncrementClicks(code string) error {
	result, err := r.db.Exec("UPDATE urls SET clicks = clicks + 1 WHERE code = ?", code)
	if err != nil {
		return fmt.Errorf("increment clicks: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GlobalStats returns aggregate statistics for all URLs.
func (r *SQLite) GlobalStats() (*domain.GlobalStats, error) {
	stats := &domain.GlobalStats{}

	err := r.db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(clicks), 0) FROM urls",
	).Scan(&stats.TotalURLs, &stats.TotalClicks)
	if err != nil {
		return nil, fmt.Errorf("get global stats: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	err = r.db.QueryRow(
		"SELECT COUNT(*) FROM urls WHERE DATE(created_at) = ?",
		today,
	).Scan(&stats.URLsToday)
	if err != nil {
		return nil, fmt.Errorf("get urls today: %w", err)
	}

	return stats, nil
}

// Close closes the database connection.
func (r *SQLite) Close() error {
	return r.db.Close()
}
