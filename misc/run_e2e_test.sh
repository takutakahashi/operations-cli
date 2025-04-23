#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Build the operations binary
cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations -c misc/e2e.yaml"

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
${OPERATIONS_BIN} exec echo_hello --set message="e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 2 passed${NC}"
else
    echo -e "${RED}✗ Test 2 failed${NC}"
    exit 1
fi

# Test 3: Echo goodbye command
echo -e "\n${GREEN}Test 3: Echo goodbye command${NC}"
${OPERATIONS_BIN} exec echo_goodbye --set message="e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    exit 1
fi

# Test 4: Sleep command (low danger level)
echo -e "\n${GREEN}Test 4: Sleep command (low danger level)${NC}"
${OPERATIONS_BIN} exec sleep_short
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 4 passed${NC}"
else
    echo -e "${RED}✗ Test 4 failed${NC}"
    exit 1
fi

# Test 5: Sleep command (high danger level)
echo -e "\n${GREEN}Test 5: Sleep command (high danger level)${NC}"
echo "y" | ${OPERATIONS_BIN} exec sleep_medium
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 5 passed${NC}"
else
    echo -e "${RED}✗ Test 5 failed${NC}"
    exit 1
fi

# Test 6: Sleep command with --set parameter
echo -e "\n${GREEN}Test 6: Sleep command with --set parameter${NC}"
echo "y" | ${OPERATIONS_BIN} exec sleep_long --set seconds=2
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 6 passed${NC}"
else
    echo -e "${RED}✗ Test 6 failed${NC}"
    exit 1
fi

# Test 7: Bash script with variables
echo -e "\n${GREEN}Test 7: Bash script with variables${NC}"
${OPERATIONS_BIN} exec bash_variables --set name="E2E Test" --set count=42
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 7 passed${NC}"
else
    echo -e "${RED}✗ Test 7 failed${NC}"
    exit 1
fi

# Test 8: Bash script with conditional logic
echo -e "\n${GREEN}Test 8: Bash script with conditional logic${NC}"
echo "y" | ${OPERATIONS_BIN} exec bash_conditional --set value=5
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 8 passed${NC}"
else
    echo -e "${RED}✗ Test 8 failed${NC}"
    exit 1
fi

# Test 9: Bash script with loop
echo -e "\n${GREEN}Test 9: Bash script with loop${NC}"
echo "y" | ${OPERATIONS_BIN} exec bash_loop --set iterations=3
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 9 passed${NC}"
else
    echo -e "${RED}✗ Test 9 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All e2e tests passed successfully!${NC}"