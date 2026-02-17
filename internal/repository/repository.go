// Package repository provides the Repository interface for URL storage.
package repository

import (
	"errors"

	"github.com/devaloi/shrink/internal/domain"
)

// ErrNotFound is returned when a URL is not found in the repository.
var ErrNotFound = errors.New("url not found")

// Repository defines the interface for URL storage operations.
type Repository interface {
	// Create inserts a new URL and returns it with the generated short code.
	Create(original string) (*domain.URL, error)

	// GetByCode retrieves a URL by its short code.
	GetByCode(code string) (*domain.URL, error)

	// GetByOriginal retrieves a URL by its original URL (for deduplication).
	GetByOriginal(original string) (*domain.URL, error)

	// IncrementClicks increases the click count for a URL.
	IncrementClicks(code string) error

	// GlobalStats returns aggregate statistics for all URLs.
	GlobalStats() (*domain.GlobalStats, error)
}
