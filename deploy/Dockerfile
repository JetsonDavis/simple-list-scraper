# ---------- frontend build ----------
FROM node:20-bookworm AS frontend-build
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---------- backend build ----------
FROM golang:1.22-bookworm AS backend-build
WORKDIR /backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Build Go binary (CGO not required for PostgreSQL)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

# ---------- runtime ----------
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

# Nginx + supervisor + Playwright deps
RUN apt-get update && apt-get install -y \
  ca-certificates \
  nginx \
  supervisor \
  tzdata \
  curl \
  gnupg \
  libnss3 \
  libnspr4 \
  libatk1.0-0 \
  libatk-bridge2.0-0 \
  libcups2 \
  libdrm2 \
  libxkbcommon0 \
  libxcomposite1 \
  libxdamage1 \
  libxfixes3 \
  libxrandr2 \
  libgbm1 \
  libasound2 \
  libpango-1.0-0 \
  libcairo2 \
  libatspi2.0-0 \
  libx11-6 \
  libx11-xcb1 \
  libxcb1 \
  libxext6 \
  libxshmfence1 \
  libxrender1 \
  libxss1 \
  libglib2.0-0 \
  && rm -rf /var/lib/apt/lists/*

# Install Node (for Playwright browser installation) - minimal
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get update && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install Go in runtime container (needed for playwright-go installation)
COPY --from=backend-build /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

# Install playwright-go CLI tool and browsers with all dependencies
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@v0.4501.1 && \
    /root/go/bin/playwright install --with-deps chromium

WORKDIR /app
COPY --from=backend-build /out/api /app/api
COPY backend/migrations /app/migrations

# Run playwright.Install() during build by starting the app briefly with DISABLE_PLAYWRIGHT=false
# This ensures the Go library installs its driver in the correct location
# Set RUN_WORKER_ON_START=false to prevent worker from running during build
RUN timeout 10 sh -c 'RUN_WORKER_ON_START=false DISABLE_PLAYWRIGHT=false DATABASE_URL=invalid ./api || true'

# Now set environment variable to skip installation at runtime since it's already done
ENV PLAYWRIGHT_SKIP_INSTALL=1

RUN rm -rf /usr/share/nginx/html/*
COPY --from=frontend-build /frontend/dist/ /usr/share/nginx/html/

COPY deploy/nginx.conf /etc/nginx/sites-available/default
COPY deploy/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

RUN mkdir -p /data

EXPOSE 80
CMD ["/usr/bin/supervisord"]
