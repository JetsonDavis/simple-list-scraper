# Simple List + Scheduled Checker (Go + React + PostgreSQL)

Single-container app that serves:
- **React + TypeScript** frontend (static, via Nginx on port 80)
- **Go API** backend (proxied by Nginx at `/api/*`)
- **Go-native scheduler** that runs a **Playwright-backed worker** every N hours.

PostgreSQL database:
- **Local development**: localhost:5432
- **Production**: AWS RDS (future deployment)

This starter includes one site adapter: **Example.com**, which does a simple page visit and returns a single result (the page title + URL).
Use it to verify end-to-end wiring (Playwright, fuzzy match, dedupe insert, optional Twilio SMS).

## Local dev (no Docker)

### PostgreSQL Setup

Install PostgreSQL if you haven't already:
```bash
# macOS
brew install postgresql@15
brew services start postgresql@15

# Or use Docker
docker run -d --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=simple_list_scraper \
  -p 5432:5432 \
  postgres:15
```

Create the database:
```bash
psql -U postgres -h localhost -p 5432
CREATE DATABASE simple_list_scraper;
\q
```

### Backend

Prereqs: Go 1.22+, PostgreSQL running on localhost:5432, and Playwright browsers installed locally if you want to run the worker with Playwright.

The backend uses a `.env` file for configuration. A sample `.env` file is provided at `backend/.env` with default values.

```bash
cd backend
# Install dependencies
go mod tidy

# Edit .env file if needed (update DATABASE_URL with your credentials), then:
source .env
go run ./cmd/api
```

Alternatively, you can export variables manually:
```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/simple_list_scraper?sslmode=disable
export DISABLE_PLAYWRIGHT=true
go run ./cmd/api
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```
Vite dev server runs on port 3004 and proxies `/api` to `http://127.0.0.1:8004` (see `vite.config.ts`).

## Run with Docker (single container, port 80)

From repo root:
```bash
docker build -t simple-list-scraper -f deploy/Dockerfile .
docker run -d --name simple-list-scraper \
  -p 80:80 \
  -e DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable \
  simple-list-scraper
```

Note: For production with AWS RDS, update `DATABASE_URL` to your RDS endpoint.

Open: http://localhost/

## Environment variables

Core:
- `DATABASE_URL` (required): PostgreSQL connection string
  - Local: `postgres://postgres:postgres@localhost:5432/simple_list_scraper?sslmode=disable`
  - AWS RDS: `postgres://username:password@your-rds-endpoint.region.rds.amazonaws.com:5432/dbname?sslmode=require`
- `CHECK_INTERVAL_HOURS` (optional): override 6-hour schedule (e.g. `1` for hourly while testing)
- `RUN_WORKER_ON_START` (optional): `true|false` (default true)
- `FUZZY_THRESHOLD` (optional): default `0.78` (0..1)
- `DISABLE_PLAYWRIGHT` (optional): `true` to skip Playwright and do no searches (for quick API-only dev)

Quality exclusion:
- If a candidate result title contains **TS**, **CAM**, or **Telesync** (case-sensitive, TS/CAM treated as tokens), it will be **logged** and **ignored**.

Twilio (optional):
- `TWILIO_ACCOUNT_SID`
- `TWILIO_AUTH_TOKEN`
- `TWILIO_FROM_NUMBER`
- `ALERT_TO_NUMBER`

If Twilio env vars are set, the worker sends an SMS **only when a new match row is inserted** (deduped in DB).
