package ai

import (
	"context"
	"testing"
)

func TestRouterRoute(t *testing.T) {
	router := NewRouter(2000)

	query := &Query{
		Text:        "Show me recent deployments",
		ContextSize: 2000,
		Tools:       []string{},
	}

	response, err := router.Route(context.Background(), query)
	if err != nil {
		t.Fatalf("Failed to route query: %v", err)
	}

	if response == nil {
		t.Error("Response should not be nil")
	}

	if response.Text == "" {
		t.Error("Response text should not be empty")
	}
}

func TestRouterOptimizeContext(t *testing.T) {
	router := NewRouter(2000)

	data := map[string]interface{}{
		"large": "content",
		"items": []string{"a", "b", "c"},
	}

	optimized := router.OptimizeContext(data, 1000)
	if optimized == nil {
		t.Error("Optimized context should not be nil")
	}
}
