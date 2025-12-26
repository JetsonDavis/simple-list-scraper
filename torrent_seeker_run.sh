#!/bin/bash

# Torrent Seeker Run Script
# This script pulls the latest image from Docker Hub, runs the container, and follows logs

set -e  # Exit on error

DOCKER_USERNAME="jetsondavis"
IMAGE_NAME="torrent-seeker"
CONTAINER_NAME="torrent-seeker"
PORT="3004"
VERSION=${1:-"latest"}  # Use first argument as version, default to "latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Torrent Seeker - Pull & Run${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Stop existing container if running
if docker ps -q -f name=${CONTAINER_NAME} | grep -q .; then
    echo -e "${YELLOW}Stopping existing container '${CONTAINER_NAME}'...${NC}"
    docker stop ${CONTAINER_NAME}
    echo -e "${GREEN}✓ Container stopped${NC}"
fi

# Remove existing container if it exists
if docker ps -aq -f name=${CONTAINER_NAME} | grep -q .; then
    echo -e "${YELLOW}Removing existing container '${CONTAINER_NAME}'...${NC}"
    docker rm ${CONTAINER_NAME}
    echo -e "${GREEN}✓ Container removed${NC}"
fi

# Pull the latest image from Docker Hub
echo ""
echo -e "${YELLOW}Pulling image '${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}' from Docker Hub...${NC}"
docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Image pulled successfully${NC}"
else
    echo -e "${RED}✗ Failed to pull image${NC}"
    exit 1
fi

# Run the container
echo ""
echo -e "${YELLOW}Starting container '${CONTAINER_NAME}'...${NC}"

# Get DATABASE_URL from environment or use default
# Note: Update the username/password to match your PostgreSQL setup
DATABASE_URL=${DATABASE_URL:-"postgresql://jeff@host.docker.internal:5432/torrent_seeker?sslmode=disable"}

docker run -d \
  --name ${CONTAINER_NAME} \
  -p ${PORT}:80 \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="${DATABASE_URL}" \
  -e CHECK_INTERVAL_HOURS="${CHECK_INTERVAL_HOURS:-6}" \
  -e RUN_WORKER_ON_START="${RUN_WORKER_ON_START:-true}" \
  -e FUZZY_THRESHOLD="${FUZZY_THRESHOLD:-0.78}" \
  -e DISABLE_PLAYWRIGHT="${DISABLE_PLAYWRIGHT:-false}" \
  -e USE_ENTITY_MATCHING="${USE_ENTITY_MATCHING:-false}" \
  -e TWILIO_ACCOUNT_SID="${TWILIO_ACCOUNT_SID:-}" \
  -e TWILIO_AUTH_TOKEN="${TWILIO_AUTH_TOKEN:-}" \
  -e TWILIO_FROM_NUMBER="${TWILIO_FROM_NUMBER:-}" \
  -e ALERT_TO_NUMBER="${ALERT_TO_NUMBER:-}" \
  -e OLLAMA_MODEL="${OLLAMA_MODEL:-llama2}" \
  -e OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://localhost:11434}" \
  -v "$(pwd)/backend/data:/app/data" \
  ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Container started successfully${NC}"
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Application is running!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "Access the application at: ${GREEN}http://localhost:${PORT}${NC}"
    echo ""
    echo -e "${BLUE}Following container logs (Ctrl+C to exit)...${NC}"
    echo -e "${YELLOW}----------------------------------------${NC}"
    echo ""
    
    # Follow logs
    docker logs -f ${CONTAINER_NAME}
else
    echo -e "${RED}✗ Failed to start container${NC}"
    exit 1
fi
