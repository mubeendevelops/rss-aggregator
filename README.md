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
- **Framework**: `go-chi/chi/v5` for routing, `go-chi/cors` for CORS
- **Database**: PostgreSQL
- **Migrations**: `goose`-style SQL migrations under `sql/schema`
- **Queries**: Generated with `sqlc` (`internal/database`)

---

## Getting Started

### Prerequisites

- Go 1.26.9 or newer (the version pinned in `go.mod`; older Go 1.21+ toolchains download it automatically)
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

Copy `.env.example` to `.env` in the project root. The `.env` file is optional: the server also starts when `PORT` and `DB_URL` are set as real environment variables.

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
- `007_users_apikey_drop_default.sql`
- `008_indexes.sql`

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

### Status Codes

| Code | Meaning |
|---|---|
| 200 | Successful read or delete |
| 201 | Resource created (`POST`) |
| 400 | Malformed JSON body, malformed ID, empty `name`, a feed `url` that is not an absolute `http`/`https` URL, or an out-of-range `limit`/`offset` |
| 401 | Missing or malformed `Authorization` header, or unknown API key |
| 404 | Feed follow not found on delete, or a referenced record (such as `feed_id`) does not exist |
| 409 | Duplicate: the feed URL already exists, or the feed is already followed |
| 413 | Request body larger than 1 MB |
| 500 | Server fault. The body is a generic message and the details are logged server-side |

Errors have the shape `{"error": "message"}`. Database driver messages are never returned to clients.

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

  **Response**: 201 with the user object, including the generated `id` and `api_key`. The key is 32 bytes from `crypto/rand`, hex encoded (64 characters). `name` must not be empty.

- **GET** `/v1/users` (auth required)  
  Returns 200 with the authenticated user.

### Feeds

- **POST** `/v1/feeds` (auth required)  
  Create a feed owned by the authenticated user. Returns 201, or 409 if the URL is already registered. `name` must not be empty and `url` must be an absolute `http` or `https` URL, otherwise 400.

  **Body**:

  ```json
  {
    "name": "Example Blog",
    "url": "https://example.com/rss.xml"
  }
  ```

- **GET** `/v1/feeds`  
  Returns 200 with all feeds in the system.

### Feed Follows

- **POST** `/v1/feed_follows` (auth required)  
  Follow a feed. Returns 201, 409 if already followed, or 404 if the feed does not exist.

  **Body**:

  ```json
  {
    "feed_id": "uuid-of-feed"
  }
  ```

- **GET** `/v1/feed_follows` (auth required)  
  Returns 200 with the feeds followed by the authenticated user.

- **DELETE** `/v1/feed_follows/{feedFollowID}` (auth required)  
  Unfollow a feed. Returns 200, or 404 if the follow does not exist or belongs to another user.

### Posts

- **GET** `/v1/posts` (auth required)  
  Get the most recent posts for the authenticated user (across all followed feeds), newest first.

  **Query parameters**:

  | Parameter | Default | Allowed |
  |---|---|---|
  | `limit` | 10 | 1 to 100 |
  | `offset` | 0 | 0 or greater |

  For example `GET /v1/posts?limit=20&offset=40` returns the third page of 20. Values outside the allowed range return 400.

---

## Background Scraper

The scraper is started from `cmd/rss-aggregator/main.go`:

- **Function**: `startScraping(ctx context.Context, db database.Querier, cfg scraperConfig)`
- **Behavior**:
  - Periodically selects the next feeds to fetch (`GetNextFeedsToFetch`), least recently fetched first, skipping feeds fetched within the minimum refetch interval.
  - Spawns goroutines to fetch and parse RSS (`urlToFeed` in `cmd/rss-aggregator/rss.go`).
  - Inserts posts into the `posts` table via `CreatePost`, skipping duplicates on URL.

Fetching is defensive: responses must be 200 and at most 10 MB, each cycle has a one minute timeout, and the HTTP client refuses to connect to loopback, private, link-local and unspecified addresses (checked after DNS resolution and on every redirect), so a feed URL cannot be used to reach internal services. Set `FEED_ALLOW_PRIVATE_ADDRESSES=true` to fetch from a feed server on your own machine while testing.

The server has read, write and idle timeouts, and on SIGINT or SIGTERM it stops accepting requests, lets in-flight requests and the scraper finish, and exits.

You can tune the scraper with environment variables (see `.env.example`). Invalid values stop the server at startup.

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_INTERVAL` | `30s` | Time between scrape cycles, as a Go duration (`30s`, `2m`, `1h`) |
| `SCRAPE_CONCURRENCY` | `10` | Most feeds fetched per cycle, 1 to 100 |
| `SCRAPE_MIN_REFETCH_INTERVAL` | `5m` | How long a feed is left alone after a fetch, so a small set of feeds is not hit every cycle. `0` disables it |

---

## Project Structure

```
cmd/rss-aggregator/     package main: the server binary
  main.go               entrypoint, configuration, server bootstrap, scraper start
  router.go             routes and CORS
  config.go             scraper settings from the environment
  handler_*.go          HTTP handlers for users, feeds, feed follows, posts, readiness, error testing
  middleware_auth.go    authentication middleware using user API keys
  models.go, json.go    API response types and JSON helpers
  db_errors.go          maps database errors to HTTP status codes
  validate.go           request input validation
  feed_client.go        HTTP client for feeds with the private-address guard
  rss.go                RSS XML fetching and parsing
  scraper.go            periodic scraping logic and post creation
internal/auth/          API key generation and header parsing
internal/database/      sqlc-generated types and queries
sql/schema/             goose migrations
sql/queries/            source SQL used by sqlc
web/index.html          static demo page for the API
scripts/smoke.sh        end-to-end smoke test
.github/workflows/      CI
docker-compose.yml, Makefile, .env.example    local development tooling
```

---
