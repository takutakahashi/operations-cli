#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Build the operations binary
cd "$(dirname "$0")/.."
OPERATIONS_BIN="$(pwd)/build/operations"
CONFIG_FILE="$(pwd)/misc/e2e.yaml"

# SSH connection information
SSH_HOST="localhost"
SSH_PORT="2222"
SSH_USER="testuser"
SSH_PASSWORD="testpassword"

echo "Starting SSH e2e tests..."
echo "Using operations binary: ${OPERATIONS_BIN}"
echo "Using config file: ${CONFIG_FILE}"
echo "SSH connection: ${SSH_USER}@${SSH_HOST}:${SSH_PORT}"

# Test 1: Echo hello via SSH
echo -e "\n${GREEN}Test 1: Echo hello command via SSH${NC}"
${OPERATIONS_BIN} --config "${CONFIG_FILE}" --remote --host ${SSH_HOST} --port ${SSH_PORT} --user ${SSH_USER} --password ${SSH_PASSWORD} exec echo_hello --set message="SSH e2e test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 1 passed${NC}"
else
    echo -e "${RED}✗ Test 1 failed${NC}"
    exit 1
fi

# Test 2: Sleep short via SSH (low danger level)
echo -e "\n${GREEN}Test 2: Sleep short command via SSH${NC}"
${OPERATIONS_BIN} --config "${CONFIG_FILE}" --remote --host ${SSH_HOST} --port ${SSH_PORT} --user ${SSH_USER} --password ${SSH_PASSWORD} exec sleep_short
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 2 passed${NC}"
else
    echo -e "${RED}✗ Test 2 failed${NC}"
    exit 1
fi

# Test 3: Sleep long via SSH with --set parameter
echo -e "\n${GREEN}Test 3: Sleep long command via SSH with --set parameter${NC}"
echo "y" | ${OPERATIONS_BIN} --config "${CONFIG_FILE}" --remote --host ${SSH_HOST} --port ${SSH_PORT} --user ${SSH_USER} --password ${SSH_PASSWORD} exec sleep_long --set seconds=2
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All SSH e2e tests passed successfully!${NC}"