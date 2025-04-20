package main

import (
	"flag"
	"log"

	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	toolMgr := tool.NewManager(cfg)

	server := NewCustomMCPServer("operation-mcp", "1.0.0", toolMgr)

	server.RegisterTools()

	if err := server.ServeStdio(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
