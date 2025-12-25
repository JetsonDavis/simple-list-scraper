#!/bin/bash

# Stop the Go API server by process name
pkill -9 -f "go run ./cmd/api" 2>/dev/null

# Wait a moment for processes to die
sleep 1

# Kill any remaining process using port 8004
lsof -ti:8004 | xargs kill -9 2>/dev/null

# Wait again
sleep 1

# Final check and kill
lsof -ti:8004 | xargs kill -9 2>/dev/null

echo "Backend stopped"
