#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Build the operations binary
cd "$(dirname "$0")/.."

# Check if we're running in GitHub Actions
if [ -n "$GITHUB_ACTIONS" ]; then
  # In GitHub Actions, use the GitHub Pages URL
  REMOTE_CONFIG_URL="https://takutakahashi.github.io/operation-mcp/remote_config.yaml"
else
  # Create a simple HTTP server to serve the remote config file
  echo "Starting Python HTTP server..."
  mkdir -p /tmp/remote-config-test
  cp docs/remote_config.yaml /tmp/remote-config-test/
  cd /tmp/remote-config-test
  python3 -m http.server 8080 &
  SERVER_PID=$!

  # Ensure the server is killed when the script exits
  trap "kill $SERVER_PID" EXIT

  # Wait for the server to start
  sleep 2

  # Go back to the project directory
  cd - > /dev/null
  
  # Set the local URL
  REMOTE_CONFIG_URL="http://localhost:8080/remote_config.yaml"
fi

# Use the existing binary if available, or skip the tests
if [ ! -f "build/operations" ]; then
    echo -e "${RED}Operations binary not found. Please build it first with 'make build'.${NC}"
    echo -e "${RED}Skipping tests...${NC}"
    exit 0
fi

OPERATIONS_BIN="$(pwd)/build/operations"

# Wait for the server to start
sleep 2

# Remote config URL is set above

echo "Starting remote config e2e tests..."
echo "Using operations binary: ${OPERATIONS_BIN}"
echo "Using remote config URL: ${REMOTE_CONFIG_URL}"

# Test 1: List tools from remote config
echo -e "\n${GREEN}Test 1: List tools from remote config${NC}"
${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} list
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 1 passed${NC}"
else
    echo -e "${RED}✗ Test 1 failed${NC}"
    exit 1
fi

# Test 2: Remote echo hello command
echo -e "\n${GREEN}Test 2: Remote echo hello command${NC}"
${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} exec remote-echo_hello --set message="remote e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 2 passed${NC}"
else
    echo -e "${RED}✗ Test 2 failed${NC}"
    exit 1
fi

# Test 3: Remote echo goodbye command
echo -e "\n${GREEN}Test 3: Remote echo goodbye command${NC}"
${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} exec remote-echo_goodbye --set message="remote e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    exit 1
fi

# Test 4: Remote sleep command (low danger level)
echo -e "\n${GREEN}Test 4: Remote sleep command (low danger level)${NC}"
${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} exec remote-sleep_short
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 4 passed${NC}"
else
    echo -e "${RED}✗ Test 4 failed${NC}"
    exit 1
fi

# Test 5: Remote sleep command (medium danger level)
echo -e "\n${GREEN}Test 5: Remote sleep command (medium danger level)${NC}"
echo "y" | ${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} exec remote-sleep_medium
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 5 passed${NC}"
else
    echo -e "${RED}✗ Test 5 failed${NC}"
    exit 1
fi

# Test 6: Remote sleep command with --set parameter
echo -e "\n${GREEN}Test 6: Remote sleep command with --set parameter${NC}"
echo "y" | ${OPERATIONS_BIN} -c ${REMOTE_CONFIG_URL} exec remote-sleep_long --set seconds=2
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 6 passed${NC}"
else
    echo -e "${RED}✗ Test 6 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All remote config e2e tests passed successfully!${NC}"