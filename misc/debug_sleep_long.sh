#!/bin/bash

# Enable debug logging
export OM_DEBUG=true

# Build the operations binary
cd "$(dirname "$0")/.."
go build -o build/operations .
OPERATIONS_BIN="$(pwd)/build/operations -c misc/e2e.yaml"

echo "Debug: Testing sleep_long command..."
echo "Debug: Binary path: $OPERATIONS_BIN"
echo "Debug: Tool configuration:"
echo "----------------------------------------"
echo "Full sleep tool configuration:"
sed -n '/name: sleep/,/name: bash/p' misc/e2e.yaml
echo "----------------------------------------"

echo "Debug: Listing available tools"
$OPERATIONS_BIN list

echo "Debug: Running command with --help first"
$OPERATIONS_BIN exec sleep --help
echo "Debug: Sleep long help"
$OPERATIONS_BIN exec sleep_long --help

echo "Debug: Running actual command"
echo "Debug: First attempt with normal execution"
echo -e "y\ny" | $OPERATIONS_BIN exec sleep_long --set seconds=2
exit_code=$?
echo "Debug: Command exited with code $exit_code"

if [ $exit_code -ne 0 ]; then
    echo "Debug: Command failed. Trying with verbose mode..."
    echo -e "y\ny" | $OPERATIONS_BIN -v exec sleep_long --set seconds=2
    echo "Debug: Verbose command exited with code $?"
    
    echo "Debug: Trying with explicit parameter format..."
    echo -e "y\ny" | $OPERATIONS_BIN exec sleep_long --set "seconds=2"
    echo "Debug: Explicit parameter command exited with code $?"
    
    echo "Debug: Checking parameter parsing..."
    $OPERATIONS_BIN exec sleep_long --help
fi 