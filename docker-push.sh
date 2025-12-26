#!/bin/bash

# Docker Build and Push Script for Torrent Seeker
# This script builds the Docker image and pushes it to Docker Hub

set -e  # Exit on error

DOCKER_USERNAME="jetsondavis"
IMAGE_NAME="torrent-seeker"
VERSION=${1:-"latest"}  # Use first argument as version, default to "latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Docker Build & Push to Docker Hub${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if logged into Docker Hub
if ! docker info 2>&1 | grep -q "Username: ${DOCKER_USERNAME}"; then
    echo -e "${YELLOW}Not logged into Docker Hub as ${DOCKER_USERNAME}${NC}"
    echo -e "${YELLOW}Attempting to log in...${NC}"
    docker login
    echo ""
fi

# Build the Docker image
echo -e "${YELLOW}Building Docker image '${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}'...${NC}"
docker build -t ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} -f deploy/Dockerfile .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Docker image built successfully${NC}"
else
    echo -e "${RED}✗ Docker build failed${NC}"
    exit 1
fi

# Also tag as latest if a specific version was provided
if [ "$VERSION" != "latest" ]; then
    echo ""
    echo -e "${YELLOW}Tagging as 'latest' as well...${NC}"
    docker tag ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} ${DOCKER_USERNAME}/${IMAGE_NAME}:latest
    echo -e "${GREEN}✓ Tagged as latest${NC}"
fi

# Push to Docker Hub
echo ""
echo -e "${YELLOW}Pushing to Docker Hub...${NC}"
docker push ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully pushed ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}${NC}"
else
    echo -e "${RED}✗ Failed to push to Docker Hub${NC}"
    exit 1
fi

# Push latest tag if it was created
if [ "$VERSION" != "latest" ]; then
    echo ""
    echo -e "${YELLOW}Pushing 'latest' tag...${NC}"
    docker push ${DOCKER_USERNAME}/${IMAGE_NAME}:latest
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Successfully pushed ${DOCKER_USERNAME}/${IMAGE_NAME}:latest${NC}"
    else
        echo -e "${RED}✗ Failed to push latest tag${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Push Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Image available at:${NC}"
echo -e "  ${YELLOW}docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}${NC}"
if [ "$VERSION" != "latest" ]; then
    echo -e "  ${YELLOW}docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:latest${NC}"
fi
echo ""
echo -e "${BLUE}View on Docker Hub:${NC}"
echo -e "  ${YELLOW}https://hub.docker.com/r/${DOCKER_USERNAME}/${IMAGE_NAME}${NC}"
echo ""
