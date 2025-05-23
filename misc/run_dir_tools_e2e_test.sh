#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Set log level to ERROR to minimize output
export OM_LOG_LEVEL=ERROR

# Build the operations binary
cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations"

echo "Starting directory tools e2e tests..."

# Test directory structure
TEST_DIR="$(pwd)/misc/dir-tools-test"
OUT_YAML="$(pwd)/tmp/dir-tools-test-out.yaml"
mkdir -p "$(dirname "$OUT_YAML")"

echo -e "\n${GREEN}Test 1: Compile directory with tools directory${NC}"
${OPERATIONS_BIN} config compile -d "$TEST_DIR" -o "$OUT_YAML"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 1 passed${NC}"
else
    echo -e "${RED}✗ Test 1 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 2: Verify tools are loaded from directory${NC}"
list_output=$(${OPERATIONS_BIN} -c "$OUT_YAML" list)

if echo "$list_output" | grep -q "tool1"; then
    echo -e "${GREEN}✓ Tool1 found in list${NC}"
else
    echo -e "${RED}✗ Tool1 not found in list${NC}"
    exit 1
fi

if echo "$list_output" | grep -q "tool2"; then
    echo -e "${GREEN}✓ Tool2 found in list${NC}"
else
    echo -e "${RED}✗ Tool2 not found in list${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 3: Execute tool1${NC}"
output=$(${OPERATIONS_BIN} -c "$OUT_YAML" exec tool1 --set message="e2e test")
if echo "$output" | grep -q "Tool1 executed with message: e2e test"; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    echo "Expected output to contain 'Tool1 executed with message: e2e test'"
    echo "Actual output: $output"
    exit 1
fi

echo -e "\n${GREEN}Test 4: Execute tool2${NC}"
output=$(${OPERATIONS_BIN} -c "$OUT_YAML" exec tool2 --set count="42")
if echo "$output" | grep -q "Tool2 executed with count: 42"; then
    echo -e "${GREEN}✓ Test 4 passed${NC}"
else
    echo -e "${RED}✗ Test 4 failed${NC}"
    echo "Expected output to contain 'Tool2 executed with count: 42'"
    echo "Actual output: $output"
    exit 1
fi

echo -e "\n${GREEN}All directory tools e2e tests passed successfully!${NC}"
