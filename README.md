# Simple List + Scheduled Checker (Go + React + SQLite)

Single-container app that serves:
- **React + TypeScript** frontend (static, via Nginx on port 80)
- **Go API** backend (proxied by Nginx at `/api/*`)
- **Go-native scheduler** that runs a **Playwright-backed worker** every N hours.

SQLite persists **outside the container** via a host bind mount (`/data/app.db`).

This starter includes one site adapter: **Example.com**, which does a simple page visit and returns a single result (the page title + URL).
Use it to verify end-to-end wiring (Playwright, fuzzy match, dedupe insert, optional Twilio SMS).

## Local dev (no Docker)

### Backend
Prereqs: Go 1.22+, and Playwright browsers installed locally if you want to run the worker with Playwright.

The backend uses a `.env` file for configuration. A sample `.env` file is provided at `backend/.env` with default values.

```bash
cd backend
mkdir -p ../data
# Edit .env file if needed, then:
source .env
go run ./cmd/api
```

Alternatively, you can export variables manually:
```bash
export DB_PATH=../data/app.db
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
mkdir -p data
docker run -d --name simple-list-scraper   -p 80:80   -v "$(pwd)/data:/data"   -e DB_PATH=/data/app.db   simple-list-scraper
```

Open: http://localhost/

## Environment variables

Core:
- `DB_PATH` (required): path to SQLite file (e.g. `/data/app.db`)
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
