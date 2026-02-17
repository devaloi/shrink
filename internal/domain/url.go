// Package domain defines the core business types for the URL shortener.
package domain

import "time"

// URL represents a shortened URL entity.
type URL struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Original  string    `json:"original_url"`
	Clicks    int64     `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest is the payload for creating a new short URL.
type CreateRequest struct {
	URL string `json:"url"`
}

// CreateResponse is returned after successfully creating a short URL.
type CreateResponse struct {
	ShortURL string `json:"short_url"`
	Code     string `json:"code"`
}

// StatsResponse contains statistics for a shortened URL.
type StatsResponse struct {
	Code      string    `json:"code"`
	Original  string    `json:"original_url"`
	Clicks    int64     `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

// GlobalStats contains aggregate statistics for all URLs.
type GlobalStats struct {
	TotalURLs   int64 `json:"total_urls"`
	TotalClicks int64 `json:"total_clicks"`
	URLsToday   int64 `json:"urls_today"`
}

// HealthResponse contains the health check response.
type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}
