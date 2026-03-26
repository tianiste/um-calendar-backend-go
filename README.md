# UM Calendar API (Go)

A Go backend that serves University of Maribor (FOV) calendars in `.ics` format.

It:
- scrapes available calendar names/links,
- stores them in Postgres,
- checks for changes every hour,
- serves names and calendar content through HTTP endpoints.

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /data/names` | Returns all available calendar names as `string[]` |
| `GET /data/cal/:name` | Returns raw calendar `.ics` content (`text/calendar`) |

### Examples

```bash
# health
curl http://localhost:8080/health

# names
curl http://localhost:8080/data/names

# single calendar (.ics content)
curl http://localhost:8080/data/cal/02---1-letnik-VS-Informacijski-sistemi-Izredni
```

## How It Works

1. On startup, the app runs DB migrations automatically.
2. It connects to Postgres and configures DB connection pooling.
3. It performs an immediate sync:
   - scrapes calendar names/links,
   - upserts calendars into DB,
   - checks each `.ics` with conditional headers (`ETag`, `Last-Modified`) and hash fallback.
4. It repeats sync every hour in the background.
5. API handlers serve names and `.ics` content from DB-backed links.

## Configuration

Create `.env` from `.env.example`:

```bash
cp .env.example .env
```

### Required

- `DATABASE_URL` — PostgreSQL connection string

### Optional

- `MIGRATIONS_PATH` (default: `db/migrations`)
- `RATE_LIMIT_RPS` (default: `5`)
- `RATE_LIMIT_BURST` (default: `20`)
- `DB_MAX_OPEN_CONNS` (default: `25`)
- `DB_MAX_IDLE_CONNS` (default: `10`)
- `DB_CONN_MAX_LIFETIME` (default: `30m`)
- `ICS_CACHE_TTL` (default: `15m`) - Redis TTL for cached `.ics` responses
- `REDIS_URL` (preferred) - full Redis connection URL, e.g. `redis://...` or `rediss://...`
- `REDIS_ADDR` - Redis host:port (used when `REDIS_URL` is not set)
- `REDIS_PASSWORD` - Redis password (optional, required for most hosted Redis)
- `REDIS_DB` (default: `0`) - Redis DB index
- `REDIS_TLS` (default: `false`) - enable TLS when using `REDIS_ADDR`

## Local Development

### Requirements

- Go `1.25+`
- PostgreSQL (local or hosted, e.g. Neon)

### Run

```bash
go mod tidy
go run ./api/cmd/main.go
```

Server starts on `http://localhost:8080`.

## Notes

- If DB setup fails at startup, the app falls back to in-memory scraping so endpoints can still work.
- `/data/cal/:name` returns `404` when the calendar is not found.
- Rate limiting is applied per client IP.
- If Redis is configured, `/data/cal/:name` uses cache-aside with TTL and sets `X-Cache: HIT|MISS`.

## Redis + Azure

For Azure Cache for Redis, use `REDIS_URL` with TLS:

```env
REDIS_URL=rediss://:<access-key>@<name>.redis.cache.windows.net:6380/0
ICS_CACHE_TTL=15m
```

Recommended Azure setup:

- Host API on Azure App Service or Container Apps.
- Store `REDIS_URL` in App Settings (or Key Vault reference), not in source code.
- Keep TLS enabled (`rediss://`) for production.
