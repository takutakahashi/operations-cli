# Operations CLI Tool

A CLI tool for executing operations defined in a YAML configuration file.

## Features

- Dynamic command generation based on YAML configuration
- Hierarchical command structure with subcommands
- Parameter validation and templating
- Danger level management for sensitive operations
- Configurable action types (confirm, timeout, force)
- Remote execution via SSH
- Shell script execution with template variable support

## Installation

### Quick Install (Recommended)

Install the latest version with a single command:

```bash
# Install latest version
curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | bash
```

Install a specific version:

```bash
# Install specific version
curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | bash -s -- -v 1.0.0
```

For more installation options, see [detailed installation instructions](https://takutakahashi.github.io/operation-mcp/installation).

### Building from source

Prerequisites:
- Go 1.24 or later

```bash
# Clone the repository
git clone https://github.com/takutakahashi/operation-mcp.git
cd operation-mcp

# Build the binary
make build

# Install the binary (optional)
make install
```

## Usage

### Configuration

Create a YAML configuration file with your tools and actions. See `docs/examples/config.yaml` for an example.

### Running commands

```bash
# Using the default config file (./config.yaml or ~/.operations/config.yaml)
operations kubectl_get_pod --namespace my-namespace

# Using a specific config file
operations --config /path/to/config.yaml kubectl_get_pod --namespace my-namespace

# Using a remote config file via HTTP
operations --config https://example.com/path/to/config.yaml kubectl_get_pod --namespace my-namespace

# Running a subtool with parameters
operations kubectl_describe_pod --namespace my-namespace --pod my-pod

# Running a dangerous operation (will prompt for confirmation)
operations kubectl_delete_pod --namespace my-namespace --pod my-pod

# Running a shell script tool with parameters
operations script-example --param1 "Hello World"

# Running commands on a remote host via SSH
operations --remote --host example.com --user admin kubectl_get_pod --namespace my-namespace

# Using a specific SSH key
operations --remote --host example.com --user admin --key ~/.ssh/custom_key kubectl_get_pod --namespace my-namespace

# Upgrading to the latest version
operations upgrade

# Upgrading to a specific version
operations upgrade --version v1.0.0

# List available versions without upgrading
operations upgrade --dry-run

# Upgrade without confirmation prompt
operations upgrade --force
```

### Upgrade Options

```bash
operations upgrade [flags]

Flags:
  --dry-run        Only show available versions without upgrading
-f, --force        Skip confirmation prompt
-h, --help         Help for upgrade
-o, --output path  Path where to install the binary (default is current binary location)
-v, --version ver  Version to upgrade to (default is latest version)
```

### Remote Execution Options

You can execute commands on a remote host using the following options:

```bash
--remote            Enable remote execution via SSH
--host string       SSH remote host (required in remote mode)
--user string       SSH username (default: current user)
--key string        Path to SSH private key (default: ~/.ssh/id_rsa)
--password string   SSH password (key authentication is preferred)
--port int          SSH port (default: 22)
--timeout duration  SSH connection timeout (default: 10s)
--verify-host       Verify host key (default: true)
```

You can also set SSH options in the configuration file:

```yaml
ssh:
  host: example.com
  user: username
  key: ~/.ssh/id_rsa
  port: 22
  verify_host: true
  timeout: 10
```

## Configuration Format

See `docs/spec.md` for detailed configuration format documentation.

### Configuration Imports

You can import additional configuration files using the `imports` field:

```yaml
imports:
  - path/to/another/config.yaml
  - /absolute/path/to/config.yaml
  - https://example.com/path/to/config.yaml
```

When importing configurations:
- Relative paths are resolved relative to the parent config file
- Actions are merged (combined) from all imported configs
- Tools are merged, with the parent config taking precedence for tools with the same name
- Parent SSH configuration takes precedence over imported SSH configuration

Example:

```yaml
# Main config.yaml
actions:
  - danger_level: high
    type: confirm
    # ...

imports:
  - team/database-tools.yaml
  - team/network-tools.yaml

tools:
  # These tools take precedence over tools with the same name in imported configs
```

### Tool Configuration

Tools can be configured in two ways: using commands or using shell scripts.

#### Command-based Tools

```yaml
tools:
  - name: kubectl
    command:
      - kubectl
    params:
      namespace:
        description: The namespace to run the command in
        type: string
        required: true
    subtools:
      - name: get pod
        args: ["get", "pod", "-o", "json", "-n", "{{.namespace}}"]
      # More subtools...
```

#### Script-based Tools

```yaml
tools:
  - name: script-example
    script: |
      #!/bin/bash
      echo "Running a shell script with parameters"
      echo "Parameter value: {{.param1}}"
      # Any bash commands can be used
      ls -la
      date
    params:
      param1:
        description: A parameter for the script
        type: string
        required: true
    subtools:
      - name: complex
        script: |
          #!/bin/bash
          # More complex script operation
          echo "Complex operation with {{.param1}}"
          # More bash commands...
```

Template variables defined in `params` can be used in both command arguments and shell scripts.

## Development

### Running tests

```bash
make test
```

### Running tests with coverage

```bash
make test-coverage
```

### Formatting code

```bash
make fmt
```

## CI/CD

This project uses GitHub Actions for continuous integration and continuous deployment.

### CI Workflows

- **Unit Tests**: Runs on every pull request.
  - Runs code formatting checks
  - Runs linting
  - Executes unit tests
  - Generates and uploads test coverage report

- **E2E Tests**: Runs on push to main branch and can be manually triggered.
  - Builds the application
  - Runs end-to-end tests using the test configuration
  - Uploads the built binary as an artifact

### CD Workflow

- **Release**: Triggered when a tag with format `v*` is pushed.
  - Runs unit tests
  - Uses GoReleaser to build binaries for multiple platforms:
    - Linux (x86_64, aarch64)
    - macOS (x86_64, aarch64)
  - Creates a GitHub Release with the built binaries
  - Uploads release artifacts

### Creating a Release

To create a new release:

```bash
# Tag the commit
git tag v1.0.0

# Push the tag
git push origin v1.0.0
```

This will automatically trigger the release workflow.
