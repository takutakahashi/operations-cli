set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations -c misc/param_ref_e2e.yaml"

echo "Starting param_ref e2e tests..."
echo "Using operations binary: ${OPERATIONS_BIN}"

echo -e "\n${GREEN}Test 1: List tools command${NC}"
${OPERATIONS_BIN} list
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 1 passed${NC}"
else
    echo -e "${RED}✗ Test 1 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 2: Command hello with message parameter${NC}"
${OPERATIONS_BIN} exec command_hello --set message="param ref test"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 2 passed${NC}"
else
    echo -e "${RED}✗ Test 2 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 3: Command hello with message and prefix parameters${NC}"
${OPERATIONS_BIN} exec command_hello --set message="param ref test" --set prefix="[TEST] "
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 3 passed${NC}"
else
    echo -e "${RED}✗ Test 3 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 5: Sleep short command (low danger level)${NC}"
${OPERATIONS_BIN} exec sleep_short --set seconds=1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 5 passed${NC}"
else
    echo -e "${RED}✗ Test 5 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 6: Sleep medium command (medium danger level)${NC}"
echo "y" | ${OPERATIONS_BIN} exec sleep_medium --set seconds=3
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 6 passed${NC}"
else
    echo -e "${RED}✗ Test 6 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Test 7: Sleep long command with seconds parameter${NC}"
echo "y" | ${OPERATIONS_BIN} exec sleep_long --set seconds=2
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Test 7 passed${NC}"
else
    echo -e "${RED}✗ Test 7 failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All param_ref e2e tests passed successfully!${NC}"
