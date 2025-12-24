#!/usr/bin/env bash
set -euo pipefail

# Format and lint both TypeScript and Go code
# Usage: ./scripts/format.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Code Quality Formatter ===${NC}"
echo ""

# Track overall success
OVERALL_SUCCESS=true

# Check and install missing tools
check_tool() {
  local tool=$1
  local install_cmd=$2

  if ! command -v "$tool" &> /dev/null; then
    echo -e "${YELLOW}Warning: $tool is not installed${NC}"
    echo -e "${YELLOW}Install with: $install_cmd${NC}"
    return 1
  fi
  return 0
}

echo -e "${BLUE}Checking required tools...${NC}"
GOLANGCI_AVAILABLE=false
GOIMPORTS_AVAILABLE=false

check_tool "golangci-lint" "brew install golangci-lint" && GOLANGCI_AVAILABLE=true
check_tool "goimports" "go install golang.org/x/tools/cmd/goimports@latest" && GOIMPORTS_AVAILABLE=true

echo ""

# TypeScript formatting and linting
echo -e "${BLUE}=== TypeScript Client ===${NC}"
echo "Running ESLint with auto-fix..."

cd "$ROOT_DIR/client"

if npm run lint:fix; then
  echo -e "${GREEN}✓ TypeScript linting completed successfully${NC}"
else
  echo -e "${RED}✗ TypeScript linting found issues${NC}"
  OVERALL_SUCCESS=false
fi

echo ""
echo "Running TypeScript type checking..."

if npm run type-check; then
  echo -e "${GREEN}✓ TypeScript type checking passed${NC}"
else
  echo -e "${RED}✗ TypeScript type checking failed${NC}"
  OVERALL_SUCCESS=false
fi

cd "$ROOT_DIR"
echo ""

# Go formatting and linting
echo -e "${BLUE}=== Go Server ===${NC}"

cd "$ROOT_DIR/server"

# Run gofmt
echo "Running gofmt..."
if gofmt -w .; then
  echo -e "${GREEN}✓ Go formatting completed${NC}"
else
  echo -e "${RED}✗ Go formatting failed${NC}"
  OVERALL_SUCCESS=false
fi

echo ""

# Run goimports if available
if [ "$GOIMPORTS_AVAILABLE" = true ]; then
  echo "Running goimports..."
  if goimports -w .; then
    echo -e "${GREEN}✓ Go imports organized${NC}"
  else
    echo -e "${RED}✗ Go imports organization failed${NC}"
    OVERALL_SUCCESS=false
  fi
  echo ""
else
  echo -e "${YELLOW}Skipping goimports (not installed)${NC}"
  echo ""
fi

# Run golangci-lint if available
if [ "$GOLANGCI_AVAILABLE" = true ]; then
  echo "Running golangci-lint..."
  if golangci-lint run --fix; then
    echo -e "${GREEN}✓ Go linting completed${NC}"
  else
    echo -e "${RED}✗ Go linting found issues${NC}"
    OVERALL_SUCCESS=false
  fi
else
  echo -e "${YELLOW}Skipping golangci-lint (not installed)${NC}"
fi

cd "$ROOT_DIR"
echo ""
echo -e "${BLUE}=== Summary ===${NC}"

if [ "$OVERALL_SUCCESS" = true ]; then
  echo -e "${GREEN}✓ All checks passed!${NC}"
  exit 0
else
  echo -e "${RED}✗ Some checks failed. Please review the output above.${NC}"
  exit 1
fi
