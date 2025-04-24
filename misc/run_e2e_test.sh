#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Enable debug logging
export OM_DEBUG=true

# Debug function
debug() {
    echo -e "${YELLOW}[DEBUG] $1${NC}"
}

# Build the operations binary
cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations -c misc/e2e.yaml"

echo "Starting e2e tests..."
debug "Using operations binary: ${OPERATIONS_BIN}"
debug "Config file contents:"
cat misc/e2e.yaml | while read line; do
    debug "  $line"
done

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
debug "Running sleep_long test with seconds=2"
debug "Command: ${OPERATIONS_BIN} exec sleep_long --set seconds=2"
debug "Expected to see two prompts for danger level confirmation"

# Run the command with debug output
output=$(echo -e "y\ny" | ${OPERATIONS_BIN} exec sleep_long --set seconds=2 2>&1)
exit_code=$?

debug "Command exit code: $exit_code"
debug "Command output:"
echo "$output" | while read line; do
    debug "  $line"
done

if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}✓ Test 6 passed${NC}"
else
    echo -e "${RED}✗ Test 6 failed${NC}"
    debug "Test failed with exit code $exit_code"
    debug "Full command output:"
    echo "$output"
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

# Test 10: Lifecycle hooks without parameters
echo -e "\n${GREEN}Test 10: Lifecycle hooks without parameters${NC}"
output=$(${OPERATIONS_BIN} exec lifecycle_test)
expected_order="Root before_exec

Subtool before_exec

Main script execution

Subtool after_exec

Root after_exec"

if [ "$output" = "$expected_order" ]; then
    echo -e "${GREEN}✓ Test 10 passed${NC}"
else
    echo -e "${RED}✗ Test 10 failed${NC}"
    echo "Expected output:"
    echo "$expected_order"
    echo "Actual output:"
    echo "$output"
    exit 1
fi

# Test 11: Lifecycle hooks with parameters
echo -e "\n${GREEN}Test 11: Lifecycle hooks with parameters${NC}"
output=$(${OPERATIONS_BIN} exec lifecycle_with-params --set param="test value")
expected_order="Root before_exec

Before exec with param: test value

Main script with param: test value

After exec with param: test value

Root after_exec"

if [ "$output" = "$expected_order" ]; then
    echo -e "${GREEN}✓ Test 11 passed${NC}"
else
    echo -e "${RED}✗ Test 11 failed${NC}"
    echo "Expected output:"
    echo "$expected_order"
    echo "Actual output:"
    echo "$output"
    exit 1
fi

echo -e "\n${GREEN}All e2e tests passed successfully!${NC}"