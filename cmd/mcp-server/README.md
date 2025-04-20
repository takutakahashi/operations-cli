# MCP Server for operation-mcp

This is an implementation of the Model Context Protocol (MCP) server for operation-mcp, which exposes all operation-mcp tools as MCP Tools for LLM applications.

## Features

- Exposes all operation-mcp tools as MCP Tools
- Uses operation-mcp parameters as MCP Tool inputs
- Uses parameter descriptions as MCP parameter descriptions
- Handles dangerous operations with confirmation mechanism

## Building

```bash
# Build the MCP server
make build-mcp-server

# Install the MCP server
make install-mcp-server
```

## Usage

```bash
# Run the MCP server with default config
./build/mcp-server

# Run the MCP server with specific config
./build/mcp-server --config /path/to/config.yaml
```

## Integration with LLM Applications

The MCP server can be integrated with any LLM application that supports the Model Context Protocol. The server exposes all operation-mcp tools as MCP Tools, which can be called by the LLM application.

### Example

```go
// LLM application code
client := mcp.NewClient("http://localhost:8080")
result, err := client.CallTool(ctx, "kubectl_get_pod", map[string]interface{}{
    "namespace": "default",
})
if err != nil {
    log.Fatalf("Error calling tool: %v", err)
}
fmt.Println(result.Text)
```

## Security Considerations

- The MCP server requires confirmation for dangerous operations
- Access to the MCP server should be restricted to trusted applications
- All tool executions are logged for audit purposes
