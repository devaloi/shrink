// Package service implements the business logic for URL shortening.
package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/devaloi/shrink/internal/domain"
	"github.com/devaloi/shrink/internal/repository"
)

// Common errors returned by the service.
var (
	ErrInvalidURL    = errors.New("invalid URL")
	ErrEmptyURL      = errors.New("URL cannot be empty")
	ErrMissingScheme = errors.New("URL must have http or https scheme")
	ErrNotFound      = repository.ErrNotFound
)

// URLService handles URL shortening business logic.
type URLService struct {
	repo    repository.Repository
	baseURL string
}

// NewURLService creates a new URL service with the given repository and base URL.
func NewURLService(repo repository.Repository, baseURL string) *URLService {
	return &URLService{
		repo:    repo,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// Shorten creates a new short URL for the given original URL.
// If the URL already exists, it returns the existing short URL.
func (s *URLService) Shorten(originalURL string) (*domain.CreateResponse, error) {
	if err := s.validateURL(originalURL); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByOriginal(originalURL)
	if err == nil {
		return &domain.CreateResponse{
			ShortURL: fmt.Sprintf("%s/%s", s.baseURL, existing.Code),
			Code:     existing.Code,
		}, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check existing url: %w", err)
	}

	created, err := s.repo.Create(originalURL)
	if err != nil {
		return nil, fmt.Errorf("create short url: %w", err)
	}

	return &domain.CreateResponse{
		ShortURL: fmt.Sprintf("%s/%s", s.baseURL, created.Code),
		Code:     created.Code,
	}, nil
}

// Resolve looks up the original URL for a short code and increments the click count.
func (s *URLService) Resolve(code string) (string, error) {
	if code == "" {
		return "", ErrNotFound
	}

	urlRecord, err := s.repo.GetByCode(code)
	if err != nil {
		return "", err
	}

	go func() {
		_ = s.repo.IncrementClicks(code)
	}()

	return urlRecord.Original, nil
}

// Stats returns statistics for a shortened URL.
func (s *URLService) Stats(code string) (*domain.StatsResponse, error) {
	if code == "" {
		return nil, ErrNotFound
	}

	urlRecord, err := s.repo.GetByCode(code)
	if err != nil {
		return nil, err
	}

	return &domain.StatsResponse{
		Code:      urlRecord.Code,
		Original:  urlRecord.Original,
		Clicks:    urlRecord.Clicks,
		CreatedAt: urlRecord.CreatedAt,
	}, nil
}

// GlobalStats returns aggregate statistics for all URLs.
func (s *URLService) GlobalStats() (*domain.GlobalStats, error) {
	return s.repo.GlobalStats()
}

func (s *URLService) validateURL(rawURL string) error {
	if rawURL == "" {
		return ErrEmptyURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrMissingScheme
	}

	if parsed.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	return nil
}
