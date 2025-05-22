package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed cursor-tools.json
var cursorTools string

type ToolSpecs []ToolSpec

type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Arguments   []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Required    bool   `json:"required"`
		Description string `json:"description"`
	}
}

func main() {
	var specs ToolSpecs

	if err := json.Unmarshal([]byte(cursorTools), &specs); err != nil {
		panic(err)
	}

	log("mcp server starting with %d tools\n", len(specs))

	tools := []mcp.Tool{}
	for _, spec := range specs {
		opts := []mcp.ToolOption{
			mcp.WithDescription(spec.Description),
		}
		for _, arg := range spec.Arguments {
			argOpts := []mcp.PropertyOption{
				mcp.Description(arg.Description),
			}
			if arg.Required {
				argOpts = append(argOpts, mcp.Required())
			}
			switch arg.Type {
			case "string":
				opts = append(opts, mcp.WithString(arg.Name,
					argOpts...,
				))
			case "string[]":
				opts = append(opts, mcp.WithArray(arg.Name,
					append([]mcp.PropertyOption{mcp.Items(map[string]any{"type": "string"})}, argOpts...)...,
				))
			case "boolean":
				opts = append(opts, mcp.WithBoolean(arg.Name,
					argOpts...,
				))
			case "integer":
				opts = append(opts, mcp.WithNumber(arg.Name,
					mcp.Required(),
					mcp.Description(arg.Description),
				))
			}
		}

		tools = append(tools, mcp.NewTool(spec.Name, opts...))
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Dagger",
		"1.0.0",
		// server.WithToolCapabilities(false),
	)

	for _, tool := range tools {
		s.AddTool(tool, catchAll)
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

var l sync.Mutex

func log(s string, args ...any) {
	l.Lock()
	defer l.Unlock()

	f, err := os.OpenFile("/tmp/mcp.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	entry := fmt.Sprintf("[%s] ", time.Now().Format("2006-01-02 15:04:05")) + fmt.Sprintf(s, args...)

	_, err = f.WriteString(entry)
	if err != nil {
		panic(err)
	}
}

func catchAll(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log("%+v\n", request)

	return mcp.NewToolResultText(""), nil
}
