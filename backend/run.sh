#!/bin/bash

# Load environment variables from .env file
set -a
source .env
set +a

# Ensure Playwright is enabled
export DISABLE_PLAYWRIGHT=false

# Run the Go API server
go run ./cmd/api
