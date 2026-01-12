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

- Go (1.21+ recommended)
- PostgreSQL running locally or in Docker
- `goose` (optional, if you want to run migrations yourself)

### 1. Clone & Install

```bash
git clone https://github.com/mubeencodes/rss-aggregator.git
cd rss-aggregator

go mod tidy
```

### 2. Configure Environment

Create a `.env` file in the project root:

```bash
PORT=8080
DB_URL=postgres://user:password@localhost:5432/rss_aggregator?sslmode=disable
```

Adjust the connection string to match your local PostgreSQL setup.

### 3. Run Database Migrations

Migrations live under `sql/schema`:

- `001_users.sql`
- `002_users_apikey.sql`
- `003_feeds.sql`
- `004_feed_follows.sql`
- `005_feeds_lastfetchedat.sql`
- `006_posts.sql`

If you use `goose`, an example command would look like:

```bash
goose -dir ./sql/schema postgres "$DB_URL" up
```

Alternatively, you can apply the SQL files manually using any PostgreSQL client.

### 4. Run the Server

```bash
go run ./...
```

The server will start on `http://localhost:$PORT` (default from `.env`, e.g. `8080`).

You’ll also see logs from the background scraper, which periodically fetches new posts from configured feeds.

---

## API Overview

All routes are prefixed with `/v1`. Requests and responses use JSON.

> **Auth**: Authenticated routes expect an API key, typically via a header like `Authorization: ApiKey <your-key>`. Check `middleware_auth.go` and `internal/auth` for the exact header format if you adjust it.

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

The scraper is started from `main.go`:

- **Function**: `startScraping(db *database.Queries, concurrency int, timeBetweenRequest time.Duration)`
- **Behavior**:
  - Periodically selects the next feeds to fetch (`GetNextFeedsToFetch`).
  - Spawns goroutines to fetch and parse RSS (`urlToFeed` in `rss.go`).
  - Inserts posts into the `posts` table via `CreatePost`, skipping duplicates on URL.

You can tune:

- **Concurrency**: number of goroutines scraping in parallel.
- **Interval**: how often feeds are fetched (currently `time.Minute` in `main.go`).

---

## Project Structure

- `main.go` – App entrypoint, router setup, CORS, server bootstrap, and scraper start.
- `handler_*.go` – HTTP handlers for users, feeds, feed follows, posts, readiness, and error testing.
- `middleware_auth.go` – Authentication middleware using user API keys.
- `internal/database` – `sqlc`-generated types and queries.
- `sql/schema` – Database schema migrations.
- `sql/queries` – Source SQL used by `sqlc`.
- `rss.go` – RSS XML parsing into Go structs.
- `scraper.go` – Periodic scraping logic and post creation.

---
