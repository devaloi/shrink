# G06: shrink — Go URL Shortener API

**Catalog ID:** G06 | **Size:** S | **Language:** Go
**Repo name:** `shrink`
**One-liner:** A clean, production-grade URL shortener API built with Go, SQLite, and middleware-driven architecture.

---

## Why This Stands Out

- **Hexagonal architecture** — handler → service → repository layers cleanly separated
- **Token bucket rate limiter** built from scratch (not a library) — shows algorithm knowledge
- **Graceful shutdown** with context propagation
- Custom **base62 encoding** for short codes — no UUIDs in URLs
- **Middleware chain**: logging, rate limiting, request ID, recovery, CORS
- SQLite with **migration system** — no ORM, clean SQL
- Thorough tests at every layer including **HTTP integration tests**

---

## Architecture

```
shrink/
├── cmd/
│   └── server/
│       └── main.go           # Entry point: wire deps, start server, handle signals
├── internal/
│   ├── config/
│   │   └── config.go         # Env-based config with defaults + validation
│   ├── domain/
│   │   └── url.go            # Core types: URL, CreateRequest, Stats
│   ├── encoding/
│   │   ├── base62.go         # Base62 encode/decode for short codes
│   │   └── base62_test.go
│   ├── handler/
│   │   ├── handler.go        # HTTP handlers (create, redirect, stats, health)
│   │   ├── handler_test.go   # Integration tests with httptest
│   │   └── response.go       # JSON response helpers
│   ├── middleware/
│   │   ├── chain.go          # Middleware chaining helper
│   │   ├── logging.go        # Structured request logging
│   │   ├── ratelimit.go      # Token bucket rate limiter
│   │   ├── recovery.go       # Panic recovery
│   │   ├── requestid.go      # X-Request-ID injection
│   │   ├── cors.go           # CORS headers
│   │   └── ratelimit_test.go
│   ├── repository/
│   │   ├── sqlite.go         # SQLite repository implementation
│   │   ├── sqlite_test.go
│   │   └── migrations.go     # Schema migrations
│   └── service/
│       ├── url.go            # Business logic: shorten, resolve, track clicks
│       └── url_test.go
├── migrations/
│   └── 001_create_urls.sql
├── main.go                    # Thin wrapper → cmd/server/main.go
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── .golangci.yml
├── LICENSE
└── README.md
```

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/shorten` | Create short URL. Body: `{"url": "https://..."}` → `{"short_url": "http://localhost:8080/abc123", "code": "abc123"}` |
| `GET` | `/:code` | Redirect to original URL (301). Increments click count. |
| `GET` | `/api/urls/:code` | Get URL info + stats: original URL, created, click count |
| `GET` | `/api/health` | Health check: `{"status": "ok", "uptime": "2h15m"}` |
| `GET` | `/api/stats` | Global stats: total URLs, total clicks, URLs today |

---

## Key Design Decisions

**Base62 encoding:** Convert auto-increment DB ID → base62 string (`a-z`, `A-Z`, `0-9`). ID 1 = "b", ID 62 = "ba", etc. Short, URL-safe, no collisions. Much cleaner than UUIDs.

**Token bucket rate limiter:** Per-IP rate limiting. Each IP gets a bucket of N tokens, refilling at R tokens/second. Implemented from scratch in `middleware/ratelimit.go` — NOT using a library. This demonstrates algorithm knowledge. The bucket uses `sync.Mutex` and lazy refill on each request.

**Repository interface:** `service` depends on a `Repository` interface, not `sqlite.go` directly. Tests can use an in-memory implementation. Production uses SQLite.

**Graceful shutdown:** Main goroutine listens for SIGINT/SIGTERM, calls `server.Shutdown(ctx)` with a 10-second deadline.

---

## Phases

### Phase 1: Scaffold & Core Types

**1.1 — Project setup**
- `go mod init github.com/devaloi/shrink`
- Dependencies: `github.com/mattn/go-sqlite3` (CGo SQLite driver)
- Use **only stdlib** for HTTP — no Gin, no Chi. Shows mastery of `net/http`.
- Create directory structure, Makefile, .gitignore, .golangci.yml

**1.2 — Domain types**
```go
// internal/domain/url.go
type URL struct {
    ID        int64
    Code      string
    Original  string
    Clicks    int64
    CreatedAt time.Time
}

type CreateRequest struct {
    URL string `json:"url"`
}

type CreateResponse struct {
    ShortURL string `json:"short_url"`
    Code     string `json:"code"`
}

type StatsResponse struct {
    Code      string    `json:"code"`
    Original  string    `json:"original_url"`
    Clicks    int64     `json:"clicks"`
    CreatedAt time.Time `json:"created_at"`
}
```

**1.3 — Base62 encoding**
- `Encode(id int64) string` and `Decode(code string) int64`
- Table-driven tests: 0→"a", 1→"b", 61→"9", 62→"ba", known large values
- Edge cases: negative input, empty string decode

**1.4 — Config**
- Load from environment with sane defaults:
  - `PORT` (default 8080)
  - `DATABASE_URL` (default `./shrink.db`)
  - `BASE_URL` (default `http://localhost:8080`)
  - `RATE_LIMIT` (default 10 req/s)
  - `RATE_BURST` (default 20)

### Phase 2: Database & Service Layer

**2.1 — SQLite repository**
- Migration: `CREATE TABLE urls (id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT UNIQUE NOT NULL, original TEXT NOT NULL, clicks INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`
- Index on `code` for fast lookups
- Methods: `Create(original string) (*URL, error)`, `GetByCode(code string) (*URL, error)`, `IncrementClicks(code string) error`, `GlobalStats() (*Stats, error)`
- After insert, read back the ID, encode to base62, update the `code` field

**2.2 — Service layer**
- `Shorten(url string) (*CreateResponse, error)` — validate URL (must parse, must have scheme), call repo, build response with full short URL
- `Resolve(code string) (string, error)` — look up code, increment clicks in background goroutine, return original URL
- `Stats(code string) (*StatsResponse, error)` — look up and return stats
- URL validation: reject empty, reject missing scheme, reject localhost in production

**2.3 — Repository interface**
```go
type Repository interface {
    Create(original string) (*domain.URL, error)
    GetByCode(code string) (*domain.URL, error)
    IncrementClicks(code string) error
    GlobalStats() (*domain.GlobalStats, error)
}
```

### Phase 3: HTTP Layer & Middleware

**3.1 — Handlers**
- Each handler is a method on a `Handler` struct that holds the service
- `CreateShortURL`: parse JSON body, call service, return 201 with JSON response
- `Redirect`: extract code from path, call service, return 301 redirect
- `GetStats`: extract code, call service, return 200 with stats JSON
- `HealthCheck`: return uptime, status
- Consistent JSON error responses: `{"error": "message", "code": 404}`

**3.2 — Middleware (all built from scratch)**
- `RequestID`: generate UUID, set `X-Request-ID` header, add to context
- `Logging`: log method, path, status, duration, request ID (structured format)
- `Recovery`: catch panics, log stack trace, return 500
- `RateLimit`: token bucket per IP (see design section)
- `CORS`: configurable allowed origins, methods, headers
- `Chain(middlewares...) func(http.Handler) http.Handler`: compose middlewares

**3.3 — Router**
- Use stdlib `http.ServeMux` (Go 1.22+ with method routing)
- Or use a minimal routing helper — extract path params manually
- Wire: middleware chain → router → handlers

**3.4 — Server startup**
- Wire all dependencies in `cmd/server/main.go`
- Start HTTP server in goroutine
- Listen for signals, graceful shutdown with 10s deadline
- Log startup info: port, database path, rate limit config

### Phase 4: Comprehensive Tests

**4.1 — Unit tests**
- `base62_test.go`: encode/decode round-trip, known values, edge cases
- `url_test.go` (service): mock repository, test validation, test shorten/resolve flow
- `ratelimit_test.go`: burst allows N requests, then rejects, refills over time
- `sqlite_test.go`: use `:memory:` database, test all CRUD operations

**4.2 — HTTP integration tests**
- `handler_test.go`: use `httptest.Server` with real service + in-memory SQLite
- Test full flow: create short URL → redirect → check stats
- Test error cases: invalid URL, nonexistent code, rate limit exceeded
- Test CORS headers present
- Test health endpoint

**4.3 — Edge cases**
- Duplicate URL submitted → should return same short code or new one (decide and document)
- Very long URL (test with 2000+ char URL)
- URL with special characters, unicode
- Concurrent requests to same code (race condition on click count)
- Database unavailable

### Phase 5: Refactor for Elegance

- Extract JSON response helpers into `response.go` to DRY up handlers
- Ensure middleware is composable and each piece is <50 lines
- Review error handling: wrap errors with context using `fmt.Errorf("...: %w", err)`
- Ensure all public types and functions have godoc comments
- Run `golangci-lint run` — fix everything
- Review for any code that could be simplified

### Phase 6: Documentation & Polish

**6.1 — README.md**
```
# shrink — URL Shortener API

[badges]

One-line description.

## Features
  Bullet list of key features

## Quick Start
  go run ./cmd/server

## API Reference
  Table of endpoints with curl examples

## Architecture
  Brief overview of layers + design decisions

## Configuration
  Environment variables table

## Development
  make build, make test, make lint

## License
  MIT
```

**6.2 — Final checks**
- Full posting checklist
- Fresh clone → make build → make test → make run → curl test
- No hardcoded paths, no personal data
- Conventional commits

---

## Tech Stack

| Component | Choice |
|-----------|--------|
| HTTP | Go stdlib net/http |
| Database | SQLite via go-sqlite3 |
| Config | Environment variables (stdlib os.Getenv) |
| Testing | stdlib testing + httptest |
| Linting | golangci-lint |

---

## Commit Plan

1. `feat: scaffold project structure with Makefile and config`
2. `feat: add base62 encoding with tests`
3. `feat: add SQLite repository with migrations`
4. `feat: add URL service with validation`
5. `feat: add HTTP handlers and routing`
6. `feat: add middleware chain (logging, rate limit, recovery, CORS, request ID)`
7. `test: add unit tests for service and repository`
8. `test: add HTTP integration tests`
9. `refactor: extract response helpers, improve error wrapping`
10. `docs: add README with API reference and architecture overview`
11. `chore: final lint pass and cleanup`
