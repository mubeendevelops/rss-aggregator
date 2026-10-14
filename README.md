## RSS Aggregator (Go + PostgreSQL)

Backend service that lets users register, follow RSS feeds, and read aggregated posts via a clean JSON API. A background scraper periodically fetches and de‑duplicates posts from all followed feeds.

### Features

- **User accounts with API keys**: Create users and authenticate via API key header.
- **RSS feed management**: Add RSS feeds, list all available feeds.
- **Feed follows**: Users can follow/unfollow feeds.
- **Post aggregation**: Background worker scrapes feeds on an interval and stores posts.
- **JSON REST API**: Versioned under `/v1` with CORS enabled for browser clients.

### Tech Stack

- **Language**: Go
- **Framework**: `go-chi/chi` for routing, `go-chi/cors` for CORS
- **Database**: PostgreSQL
- **Migrations**: `goose`-style SQL migrations under `sql/schema`
- **Queries**: Generated with `sqlc` (`internal/database`)

---

## Getting Started

### Prerequisites

- Go 1.25.4 or newer (the version pinned in `go.mod`)
- Docker with Compose (for the bundled PostgreSQL), or your own PostgreSQL
- [`goose`](https://github.com/pressly/goose) to run migrations
- [`sqlc`](https://sqlc.dev) only if you change the SQL under `sql/queries`
- `curl` and `jq` for the smoke test

### 1. Clone & Install

```bash
git clone https://github.com/mubeendevelops/rss-aggregator.git
cd rss-aggregator

go mod tidy
```

### 2. Configure Environment

Copy `.env.example` to `.env` in the project root:

```bash
cp .env.example .env
```

```bash
PORT=8080
DB_URL=postgres://postgres:postgres@localhost:5432/rss-aggregator?sslmode=disable
```

The defaults match the PostgreSQL service in `docker-compose.yml`. Adjust the connection string if you use your own PostgreSQL.

### 3. Start PostgreSQL and Run Database Migrations

```bash
make db-up      # docker compose up -d --wait (postgres:16 on localhost:5432)
make migrate    # goose up on sql/schema using DB_URL from .env
```

Migrations live under `sql/schema`:

- `001_users.sql`
- `002_users_apikey.sql`
- `003_feeds.sql`
- `004_feed_follows.sql`
- `005_feeds_lastfetchedat.sql`
- `006_posts.sql`

`make migrate` runs the equivalent of:

```bash
goose -dir ./sql/schema postgres "$DB_URL" up
```

Alternatively, you can apply the SQL files manually using any PostgreSQL client.

### 4. Run the Server

```bash
make run        # go run ./cmd/rss-aggregator
```

The server will start on `http://localhost:$PORT` (default from `.env`, e.g. `8080`).

You’ll also see logs from the background scraper, which periodically fetches new posts from configured feeds.

### 5. Try It

- `make smoke` runs `scripts/smoke.sh`, a curl script that exercises every endpoint against the running server.
- `web/index.html` is a static demo page. Open it from disk in a browser and it talks to the API at `http://localhost:8080/v1`.

### Makefile Targets

| Target | What it does |
|---|---|
| `db-up`, `db-down` | Start and stop PostgreSQL with Docker Compose |
| `migrate`, `migrate-down` | `goose up` / `goose down` on `sql/schema` |
| `sqlc` | `sqlc generate`, regenerating `internal/database` from `sql/queries` |
| `run`, `build` | Run the server, or build it to `bin/rss-aggregator` |
| `lint`, `vuln` | `golangci-lint`, `govulncheck` |
| `smoke` | End-to-end check against the running server |

The Makefile reads variables from `.env`. Use `make ENV_FILE=.env.example <target>` to run against the Compose defaults without a local `.env`.

---

## API Overview

All routes are prefixed with `/v1`. Requests and responses use JSON.

> **Auth**: Authenticated routes expect an API key, typically via a header like `Authorization: ApiKey <your-key>`. Check `cmd/rss-aggregator/middleware_auth.go` and `internal/auth` for the exact header format if you adjust it.

### Health

- **GET** `/v1/healthz`  
  Returns 200 if the service is running.

- **GET** `/v1/err`  
  Test route for error handling.

### Users

- **POST** `/v1/users`  
  **Body**:

  ```json
  {
    "name": "alice"
  }
  ```

  **Response**: User object with generated `id` and `api_key`.

- **GET** `/v1/users` (auth required)  
  Returns the authenticated user.

### Feeds

- **POST** `/v1/feeds` (auth required)  
  Create a feed owned by the authenticated user.

  **Body**:

  ```json
  {
    "name": "Example Blog",
    "url": "https://example.com/rss.xml"
  }
  ```

- **GET** `/v1/feeds`  
  List all feeds in the system.

### Feed Follows

- **POST** `/v1/feed_follows` (auth required)  
  Follow a feed.

  **Body**:

  ```json
  {
    "feed_id": "uuid-of-feed"
  }
  ```

- **GET** `/v1/feed_follows` (auth required)  
  List feeds followed by the authenticated user.

- **DELETE** `/v1/feed_follows/{feedFollowID}` (auth required)  
  Unfollow a feed.

### Posts

- **GET** `/v1/posts` (auth required)  
  Get the most recent posts for the authenticated user (across all followed feeds).  
  Currently limited to 10 posts per the `GetPostsForUser` query.

---

## Background Scraper

The scraper is started from `cmd/rss-aggregator/main.go`:

- **Function**: `startScraping(db *database.Queries, concurrency int, timeBetweenRequest time.Duration)`
- **Behavior**:
  - Periodically selects the next feeds to fetch (`GetNextFeedsToFetch`).
  - Spawns goroutines to fetch and parse RSS (`urlToFeed` in `cmd/rss-aggregator/rss.go`).
  - Inserts posts into the `posts` table via `CreatePost`, skipping duplicates on URL.

You can tune:

- **Concurrency**: number of goroutines scraping in parallel.
- **Interval**: how often feeds are fetched (currently 30 seconds in `cmd/rss-aggregator/main.go`).

---

## Project Structure

```
cmd/rss-aggregator/     package main: the server binary
  main.go               entrypoint, router setup, CORS, server bootstrap, scraper start
  handler_*.go          HTTP handlers for users, feeds, feed follows, posts, readiness, error testing
  middleware_auth.go    authentication middleware using user API keys
  models.go, json.go    API response types and JSON helpers
  rss.go                RSS XML fetching and parsing
  scraper.go            periodic scraping logic and post creation
internal/auth/          API key header parsing
internal/database/      sqlc-generated types and queries
sql/schema/             goose migrations
sql/queries/            source SQL used by sqlc
web/index.html          static demo page for the API
scripts/smoke.sh        end-to-end smoke test
docker-compose.yml, Makefile, .env.example    local development tooling
```

---
