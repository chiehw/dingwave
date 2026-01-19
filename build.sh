#!/bin/bash

set -e

# Configurable build variables
BINARY_NAME="dingwave"
BUILD_DIR="releases"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Platform targets
PLATFORMS=(
    "windows/386"
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "linux/386"
    "linux/amd64"
)

echo -e "${BLUE}=== DingTalk Exporter Build Script ===${NC}"
echo -e "${BLUE}Binary name: ${BINARY_NAME}${NC}\n"

# Prerequisites check
echo -e "${YELLOW}Checking prerequisites...${NC}"
if ! command -v pnpm &> /dev/null; then
    echo -e "${RED}Error: pnpm is not installed${NC}"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: go is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Prerequisites check passed${NC}\n"

# Clean and prepare build directory
echo -e "${YELLOW}Preparing build directory...${NC}"
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"
echo -e "${GREEN}✓ Build directory ready${NC}\n"

# Frontend build phase
echo -e "${BLUE}=== Building Frontend ===${NC}"
cd frontend/
pnpm install
pnpm build
cd ..

if [ ! -d "server/dist" ]; then
    echo -e "${RED}Error: Frontend build failed - server/dist not found${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Frontend build completed${NC}\n"

# Backend cross-compilation phase
echo -e "${BLUE}=== Building Backend for Multiple Platforms ===${NC}"
cd server/

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"

    output_name="${BINARY_NAME}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo -e "${YELLOW}Building for ${os}/${arch}...${NC}"
    GOOS=${os} GOARCH=${arch} go build \
        -ldflags "-s -w" \
        -trimpath \
        -o "../${BUILD_DIR}/${output_name}" \
        main.go

    echo -e "${GREEN}✓ Built ${output_name}${NC}"
done

cd ..

# Display build summary
echo -e "\n${BLUE}=== Build Summary ===${NC}"
echo -e "${GREEN}All builds completed successfully!${NC}\n"
echo -e "Build artifacts in ${BUILD_DIR}/:"
for file in "${BUILD_DIR}"/*; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo -e "  ${BLUE}$(basename "$file")${NC} (${size})"
    fi
done
echo -e "\n${GREEN}Build complete!${NC}"
