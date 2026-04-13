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

// ListToolsResponse is the response to list_tools
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest is a request to call a tool
type CallToolRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ContentItem represents a content item in a response
type ContentItem struct {
	Type  string      `json:"type"`
	Text  string      `json:"text,omitempty"`
	Data  interface{} `json:"data,omitempty"`
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
						Name:        "get_pods",
						Description: "List Kubernetes pods",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type": "string",
									"description": "Kubernetes namespace",
								},
								"label": map[string]interface{}{
									"type": "string",
									"description": "Label selector",
								},
							},
						},
					},
					{
						Name:        "get_services",
						Description: "List Kubernetes services",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type": "string",
									"description": "Kubernetes namespace",
								},
							},
						},
					},
					{
						Name:        "describe_pod",
						Description: "Describe a Kubernetes pod",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type": "string",
									"description": "Pod namespace",
								},
								"name": map[string]interface{}{
									"type": "string",
									"description": "Pod name",
								},
							},
							"required": []string{"namespace", "name"},
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
	case "get_pods":
		// TODO: Implement actual kubectl call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Pods in namespace: %v", args),
		})

	case "get_services":
		// TODO: Implement actual kubectl call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Services in namespace: %v", args),
		})

	case "describe_pod":
		// TODO: Implement actual kubectl call
		resp.Content = append(resp.Content, ContentItem{
			Type: "text",
			Text: fmt.Sprintf("Pod details: %v", args),
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
