.PHONY: build test clean

# Variables
BINARY_NAME=operations
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: build

# Build the application
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install the application
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Run the application with example config
run-example: build
	$(BUILD_DIR)/$(BINARY_NAME) --config docs/examples/config.yaml

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golint ./...

# Vet code
vet:
	go vet ./...