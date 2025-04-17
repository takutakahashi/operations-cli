#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Build the operations binary
cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations"

echo "Starting e2e tests..."
echo "Using operations binary: ${OPERATIONS_BIN}"

# Test 1: List tools
echo -e "\n${GREEN}Test 1: List tools command${NC}"
${OPERATIONS_BIN} list
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 1 passed${NC}"
else
    echo -e "${RED}✗ Test 1 failed${NC}"
    exit 1
fi

# Test 2: Echo hello command
echo -e "\n${GREEN}Test 2: Echo hello command${NC}"
${OPERATIONS_BIN} exec echo_hello --message "e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 2 passed${NC}"
else
    echo -e "${RED}✗ Test 2 failed${NC}"
    exit 1
fi

# Test 3: Echo goodbye command
echo -e "\n${GREEN}Test 3: Echo goodbye command${NC}"
${OPERATIONS_BIN} exec echo_goodbye --message "e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    exit 1
fi

# Test 4: Sleep command (low danger level)
echo -e "\n${GREEN}Test 4: Sleep command (low danger level)${NC}"
${OPERATIONS_BIN} exec sleep --seconds 1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 4 passed${NC}"
else
    echo -e "${RED}✗ Test 4 failed${NC}"
    exit 1
fi

# Test 5: Sleep command (high danger level)
echo -e "\n${GREEN}Test 5: Sleep command (high danger level)${NC}"
echo "y" | ${OPERATIONS_BIN} exec sleep --seconds 5
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 5 passed${NC}"
else
    echo -e "${RED}✗ Test 5 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All e2e tests passed successfully!${NC}"