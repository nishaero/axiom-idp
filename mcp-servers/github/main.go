package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ContentItem represents a content item in a response
type ContentItem struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// ListToolsResponse is the response to list_tools
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolResponse is the response from calling a tool
type CallToolResponse struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

func main() {
	// Read from stdin
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req map[string]interface{}
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading request: %v", err)
			continue
		}

		method, ok := req["method"].(string)
		if !ok {
			log.Printf("Invalid request: missing method")
			continue
		}

		switch method {
		case "list_tools":
			response := ListToolsResponse{
				Tools: []Tool{
					{
						Name:        "get_repos",
						Description: "List GitHub repositories",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"owner": map[string]interface{}{
									"type": "string",
									"description": "Repository owner",
								},
								"type": map[string]interface{}{
									"type": "string",
									"enum": []string{"all", "owner", "public", "private"},
									"description": "Repository type to filter",
								},
							},
							"required": []string{"owner"},
						},
					},
					{
						Name:        "get_issues",
						Description: "List GitHub issues",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"repo": map[string]interface{}{
									"type": "string",
									"description": "Repository in format owner/repo",
								},
								"state": map[string]interface{}{
									"type": "string",
									"enum": []string{"open", "closed", "all"},
									"description": "Issue state",
								},
							},
							"required": []string{"repo"},
						},
					},
					{
						Name:        "get_pull_requests",
						Description: "List GitHub pull requests",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"repo": map[string]interface{}{
									"type": "string",
									"description": "Repository in format owner/repo",
								},
								"state": map[string]interface{}{
									"type": "string",
									"enum": []string{"open", "closed", "all"},
									"description": "PR state",
								},
							},
							"required": []string{"repo"},
						},
					},
				},
			}

			encoder.Encode(response)

		case "call_tool":
			toolReq := req["params"].(map[string]interface{})
			name := toolReq["name"].(string)

			response := handleTool(name, toolReq["arguments"].(map[string]interface{}))
			encoder.Encode(response)

		default:
			log.Printf("Unknown method: %s", method)
		}
	}
}

func handleTool(name string, args map[string]interface{}) *CallToolResponse {
	resp := &CallToolResponse{
		Content: []ContentItem{},
	}

	switch name {
	case "get_repos":
		// TODO: Implement actual GitHub API call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Repositories for: %v", args),
		})

	case "get_issues":
		// TODO: Implement actual GitHub API call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Issues: %v", args),
		})

	case "get_pull_requests":
		// TODO: Implement actual GitHub API call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Pull requests: %v", args),
		})

	default:
		resp.IsError = true
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Unknown tool: %s", name),
		})
	}

	return resp
}
