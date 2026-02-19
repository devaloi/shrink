# shrink

[![CI](https://github.com/devaloi/shrink/actions/workflows/ci.yml/badge.svg)](https://github.com/devaloi/shrink/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A clean, production-grade URL shortener API built with Go, SQLite, and middleware-driven architecture.

## Features

- **Hexagonal architecture** — handler → service → repository layers cleanly separated
- **Token bucket rate limiter** built from scratch (not a library)
- **Graceful shutdown** with context propagation
- **Custom base62 encoding** for short codes — no UUIDs in URLs
- **Middleware chain**: logging, rate limiting, request ID, recovery, CORS
- **SQLite with migrations** — no ORM, clean SQL
- **Comprehensive tests** at every layer including HTTP integration tests
- **stdlib HTTP only** — Go 1.22+ method routing, no third-party frameworks

## Quick Start

```bash
# Clone the repository
git clone https://github.com/devaloi/shrink.git
cd shrink

# Run the server
go run ./cmd/server

# Or build and run
make build
./bin/shrink
```

The server starts on port 8080 by default.

## API Reference

### Create Short URL
```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'
```

Response:
```json
{
  "short_url": "http://localhost:8080/b",
  "code": "b"
}
```

### Follow Redirect
```bash
curl -L http://localhost:8080/b
```

Redirects to the original URL with a 301 status.

### Get URL Stats
```bash
curl http://localhost:8080/api/urls/b
```

Response:
```json
{
  "code": "b",
  "original_url": "https://example.com",
  "clicks": 5,
  "created_at": "2026-02-17T12:00:00Z"
}
```

### Global Stats
```bash
curl http://localhost:8080/api/stats
```

Response:
```json
{
  "total_urls": 42,
  "total_clicks": 1337,
  "urls_today": 10
}
```

### Health Check
```bash
curl http://localhost:8080/api/health
```

Response:
```json
{
  "status": "ok",
  "uptime": "2h15m0s"
}
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/shorten` | Create short URL |
| `GET` | `/{code}` | Redirect to original URL |
| `GET` | `/api/urls/{code}` | Get URL stats |
| `GET` | `/api/stats` | Global statistics |
| `GET` | `/api/health` | Health check |

## Architecture

```
shrink/
├── cmd/server/         # Application entry point
├── internal/
│   ├── config/         # Environment-based configuration
│   ├── domain/         # Core business types
│   ├── encoding/       # Base62 encoding for short codes
│   ├── handler/        # HTTP handlers
│   ├── middleware/     # Custom middleware (logging, rate limit, etc.)
│   ├── repository/     # SQLite data persistence
│   └── service/        # Business logic
└── migrations/         # Database schema
```

### Design Decisions

**Base62 Encoding:** Converts auto-increment database IDs to URL-safe strings using `a-zA-Z0-9`. This produces short, collision-free codes without the complexity of UUIDs.

**Token Bucket Rate Limiter:** Per-IP rate limiting implemented from scratch. Each IP gets a bucket of N tokens that refills at R tokens/second. Demonstrates algorithm knowledge rather than library usage.

**Repository Interface:** The service layer depends on a Repository interface, not the SQLite implementation directly. This enables easy testing with mock implementations.

**Graceful Shutdown:** The server listens for SIGINT/SIGTERM and gracefully drains connections with a 10-second deadline.

## Configuration

Configuration is loaded from environment variables with sensible defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | `./shrink.db` | SQLite database path |
| `BASE_URL` | `http://localhost:8080` | Base URL for short links |
| `RATE_LIMIT` | `10` | Requests per second |
| `RATE_BURST` | `20` | Maximum burst size |

Example:
```bash
PORT=3000 BASE_URL=https://short.io go run ./cmd/server
```

## Development

### Prerequisites

- Go 1.22 or later
- SQLite (via CGo go-sqlite3 driver)
- golangci-lint (for linting)

### Commands

```bash
# Build the binary
make build

# Run the server
make run

# Run all tests
make test

# Run tests with coverage
make cover

# Run linter
make lint

# Format code
make fmt

# Clean build artifacts
make clean

# Run all checks (format, vet, lint, test)
make check
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test -v ./internal/handler/...
```

## Tech Stack

| Component | Choice |
|-----------|--------|
| HTTP | Go stdlib `net/http` |
| Database | SQLite via go-sqlite3 |
| Config | Environment variables (stdlib) |
| Testing | stdlib `testing` + `httptest` |
| Linting | golangci-lint |

## License

MIT License - see [LICENSE](LICENSE) for details.
