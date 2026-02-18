# Build shrink — Go URL Shortener API

You are building a **portfolio project** for a Senior AI Engineer's public GitHub. It must be impressive, clean, and production-grade. Read these docs before writing any code:

1. **`docs/G06-go-url-shortener.md`** — Complete project spec: architecture, phases, design decisions, commit plan. This is your primary blueprint. Follow it phase by phase.
2. **`docs/github-portfolio.md`** — Portfolio goals and Definition of Done (Level 1 + Level 2). Understand the quality bar.
3. **`docs/github-portfolio-checklist.md`** — Pre-publish checklist. Every item must pass before you're done.

---

## Instructions

### Read first, build second
Read all three docs completely before writing a single line of code. Understand the hexagonal architecture, the phases, the quality expectations.

### Follow the phases in order
The project spec has 6 phases. Do them in order:
1. **Scaffold & Core Types** — project setup, domain types, base62 encoding, config
2. **Database & Service Layer** — SQLite repo with migrations, URL service, repository interface
3. **HTTP Layer & Middleware** — handlers, all middleware from scratch (logging, rate limit, recovery, CORS, request ID), routing, graceful shutdown
4. **Comprehensive Tests** — unit tests (base62, service, rate limiter, repo), HTTP integration tests with httptest, edge cases
5. **Refactor for Elegance** — extract response helpers, DRY middleware, error wrapping, godoc
6. **Documentation & Polish** — README with API reference and architecture overview, final checklist

### Use subagents
This is a substantial project. Use subagents to parallelize where it makes sense:
- One subagent for base62 encoding + tests while another does config + domain types
- One subagent for SQLite repo while another does the service layer
- One subagent for handlers while another builds the middleware chain
- A dedicated subagent for the refactoring pass
- A dedicated subagent for README + documentation

### Commit frequently
Follow the commit plan in the spec. Use **conventional commits** (`feat:`, `test:`, `refactor:`, `docs:`, `ci:`, `chore:`). Each commit should be a logical unit. Do NOT accumulate a massive uncommitted diff.

### Quality non-negotiables
- **stdlib HTTP only.** No Gin, no Chi, no Echo. Use `net/http` with Go 1.22+ routing. This shows mastery.
- **Token bucket rate limiter from scratch.** Not a library. Shows algorithm knowledge.
- **Middleware from scratch.** Every middleware piece (logging, recovery, CORS, request ID, rate limit) built by hand.
- **Tests must be real.** Table-driven where appropriate. In-memory SQLite for repo tests. httptest for integration tests. Tests must actually run and pass.
- **No fake anything.** No placeholder tests. No "TODO" comments. No stubbed-out functions. Everything works.
- **Lint clean.** Run `golangci-lint run` and fix everything. Run `gofmt` on all files.
- **Error handling.** Every error is handled. User-friendly JSON error responses. Wrapped with `fmt.Errorf("context: %w", err)`.

### Final verification
Before you consider the project done:
1. `go build ./...` — compiles clean
2. `go test ./... -race` — all tests pass, no race conditions
3. `golangci-lint run` — no issues
4. `go vet ./...` — no issues
5. Walk through `docs/github-portfolio-checklist.md` item by item
6. Test with curl: create short URL → follow redirect → check stats → verify click count incremented
7. Review git log — does the commit history tell a coherent story?

### What NOT to do
- Don't use any HTTP framework. stdlib only.
- Don't use a rate limiting library. Build token bucket from scratch.
- Don't skip the refactoring phase.
- Don't write tests after everything else. Write them alongside each component.
- Don't leave `// TODO` or `// FIXME` comments anywhere.
- Don't hardcode any personal paths, usernames, or data.
- Don't commit the binary, `.DS_Store`, or the SQLite database file.
- Don't use Docker. No Dockerfile, no docker-compose. Just `go build` and `go run`.

---

## GitHub Username

The GitHub username is **devaloi**. For Go module paths, use `github.com/devaloi/shrink`. All internal imports must use this module path. Do not guess or use any other username.

## Start

Read the three docs. Then begin Phase 1 from `docs/G06-go-url-shortener.md`.
