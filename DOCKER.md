# Docker Setup Guide

This guide explains how to run the Torrent Seeker application in a single Docker container, connecting to a PostgreSQL database running on your host machine.

## Prerequisites

1. **Docker** installed on your system
2. **PostgreSQL** running on your host machine (localhost:5432)
3. A database named `simple_list_scraper` created in PostgreSQL

## Quick Start

### 1. Ensure PostgreSQL is Running

Make sure PostgreSQL is running on your host machine:

```bash
# macOS with Homebrew
brew services start postgresql@15

# Or check if it's running
psql -U postgres -h localhost -p 5432 -c "SELECT version();"
```

### 2. Create the Database (if not already created)

```bash
psql -U postgres -h localhost -p 5432
CREATE DATABASE simple_list_scraper;
\q
```

### 3. Build the Docker Image

```bash
docker build -t torrent-seeker -f deploy/Dockerfile .
```

### 4. Run the Container

```bash
docker run -d \
  --name torrent-seeker \
  -p 80:80 \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="postgresql://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable" \
  -e CHECK_INTERVAL_HOURS=6 \
  -e RUN_WORKER_ON_START=true \
  -e FUZZY_THRESHOLD=0.78 \
  -v $(pwd)/backend/data:/app/data \
  torrent-seeker
```

The application will be available at: **http://localhost**

## Container Management

### View Logs
```bash
docker logs -f torrent-seeker
```

### Stop Container
```bash
docker stop torrent-seeker
```

### Start Container
```bash
docker start torrent-seeker
```

### Remove Container
```bash
docker rm -f torrent-seeker
```

### Rebuild and Restart
```bash
docker rm -f torrent-seeker
docker build -t torrent-seeker -f deploy/Dockerfile .
docker run -d --name torrent-seeker -p 80:80 \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="postgresql://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable" \
  torrent-seeker
```

## Environment Variables

Configure the application by setting environment variables in `docker-compose.yml` or passing them with `-e` flags:

### Required

- **DATABASE_URL**: PostgreSQL connection string
  - Format: `postgresql://user:password@host:port/database?sslmode=disable`
  - For host database: `postgresql://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable`

### Optional

- **CHECK_INTERVAL_HOURS**: How often to run the worker (default: 6)
- **RUN_WORKER_ON_START**: Run worker immediately on startup (default: true)
- **FUZZY_THRESHOLD**: Fuzzy matching threshold 0-1 (default: 0.78)
- **DISABLE_PLAYWRIGHT**: Set to `true` to disable browser automation (default: false)
- **USE_ENTITY_MATCHING**: Enable LLM-based entity extraction (default: false)

### Twilio SMS Notifications (Optional)

- **TWILIO_ACCOUNT_SID**: Your Twilio account SID
- **TWILIO_AUTH_TOKEN**: Your Twilio auth token
- **TWILIO_FROM_NUMBER**: Your Twilio phone number (e.g., +1234567890)
- **ALERT_TO_NUMBER**: Phone number to receive alerts (e.g., +1234567890)

### Ollama Integration (Optional)

- **OLLAMA_MODEL**: Model to use (default: llama2)
- **OLLAMA_BASE_URL**: Ollama API URL (default: http://localhost:11434)
  - For host Ollama: `http://host.docker.internal:11434`

## Architecture

The Docker container runs:

1. **Nginx** (port 80) - Serves the React frontend and proxies API requests
2. **Go Backend** (port 8004) - REST API and WebSocket server
3. **Playwright/Chromium** - Browser automation for web scraping

Both services are managed by **supervisord** and start automatically.

## Database Connection

The container connects to PostgreSQL running on your **host machine** using `host.docker.internal`, which Docker automatically resolves to your host's IP address.

### Connection String Format

```
postgresql://username:password@host.docker.internal:5432/database_name?sslmode=disable
```

### Troubleshooting Database Connection

If the container cannot connect to your host database:

1. **Check PostgreSQL is listening on all interfaces**:
   ```bash
   # Edit postgresql.conf
   listen_addresses = '*'  # or 'localhost,127.0.0.1'
   ```

2. **Check pg_hba.conf allows connections**:
   ```bash
   # Add this line to allow Docker connections
   host    all    all    172.17.0.0/16    md5
   ```

3. **Restart PostgreSQL**:
   ```bash
   brew services restart postgresql@15
   ```

4. **Test connection from container**:
   ```bash
   docker exec -it torrent-seeker /bin/bash
   apt-get update && apt-get install -y postgresql-client
   psql -h host.docker.internal -U postgres -d simple_list_scraper
   ```

## Data Persistence

The container mounts `./backend/data` to `/app/data` for:
- Screenshots captured during scraping
- HTML files saved for debugging
- Logs and temporary files

This data persists on your host machine even if the container is removed.

## Updating the Application

When you make code changes:

```bash
# Stop and remove the old container
docker-compose down

# Rebuild and start with new code
docker-compose up -d --build

# Or with manual Docker commands
docker rm -f torrent-seeker
docker build -t torrent-seeker -f deploy/Dockerfile .
docker run -d --name torrent-seeker -p 80:80 \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="postgresql://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable" \
  torrent-seeker
```

## Viewing Logs

```bash
# All logs
docker logs -f torrent-seeker

# Filter for API logs
docker logs -f torrent-seeker 2>&1 | grep api

# Filter for Nginx logs
docker logs -f torrent-seeker 2>&1 | grep nginx
```

## Port Configuration

- **Port 80**: Main application (frontend + API)
- **Backend internally runs on**: 8004 (proxied by Nginx)

To use a different external port, modify the `-p` flag in the `docker run` command:

```bash
docker run -d --name torrent-seeker -p 8080:80 \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="postgresql://postgres:postgres@host.docker.internal:5432/simple_list_scraper?sslmode=disable" \
  torrent-seeker

# Access at http://localhost:8080
```

## Security Notes

1. The default configuration uses `sslmode=disable` for local development
2. For production, use `sslmode=require` and proper SSL certificates
3. Change default PostgreSQL passwords
4. Use environment files or secrets management for sensitive data

## Production Deployment

For production with AWS RDS or other hosted PostgreSQL:

1. Update `DATABASE_URL` in the `docker run` command:
   ```bash
   docker run -d --name torrent-seeker -p 80:80 \
     -e DATABASE_URL="postgresql://username:password@your-rds-endpoint.region.rds.amazonaws.com:5432/dbname?sslmode=require" \
     -e CHECK_INTERVAL_HOURS=6 \
     -e RUN_WORKER_ON_START=true \
     torrent-seeker
   ```

2. Remove `--add-host host.docker.internal:host-gateway` (not needed for remote databases)

3. Configure proper SSL/TLS certificates for Nginx

4. Use Docker secrets or environment variable injection for credentials
