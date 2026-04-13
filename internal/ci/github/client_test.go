package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Test GitHub Client

func createTestClient(t *testing.T) (*GitHubClient, *httptest.Server) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/repos/test-owner/test-repo":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 1,
				"name": "test-repo",
				"full_name": "test-owner/test-repo",
				"html_url": "https://github.com/test-owner/test-repo",
				"language": "go",
				"private": false,
				"owner": map[string]string{
					"login": "test-owner",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	config := ClientConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}

	client := NewGitHubClient(logger, config)
	return client, server
}

func TestGitHubClient_GetRepository(t *testing.T) {
	client, server := createTestClient(t)
	defer server.Close()

	ctx := context.Background()
	repo, err := client.GetRepository(ctx, "test-owner", "test-repo")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo == nil {
		t.Error("Expected non-nil repository")
	}

	if repo.Name != "test-repo" {
		t.Errorf("Expected name 'test-repo', got '%s'", repo.Name)
	}
}

func TestGitHubClient_Validate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := ClientConfig{
		Timeout: 100 * time.Millisecond,
	}

	client := NewGitHubClient(logger, config)
	err := client.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid timeout")
	}
}

func TestGitHubClient_GetHealthStatus(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := ClientConfig{
		Timeout: 30 * time.Second,
	}

	client := NewGitHubClient(logger, config)
	status := client.GetHealthStatus(context.Background())

	if status["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status["status"])
	}
}

// Test Webhook Handler

func createTestWebhookHandler(t *testing.T) (*WebhookHandler, *httptest.Server) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	registry := NewEventHandlerRegistry()
	client, _ := createTestClient(t)

	handler := NewWebhookHandler(logger, client, "secret", registry)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.Handle(w, r)
	}))

	return handler, server
}

func TestWebhookHandler_Handle(t *testing.T) {
	handler, server := createTestWebhookHandler(t)
	defer server.Close()

	// Create a mock push event payload
	payload := map[string]string{
		"action": "push",
	}
	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestEventHandlerRegistry_RegisterHandler(t *testing.T) {
	registry := NewEventHandlerRegistry()

	handlerCalled := false
	handler := func(event *PREvent) {
		handlerCalled = true
	}

	registry.RegisterHandler("push", handler)

	// Verify handler was registered
	handlers := registry.GetHandlers("push")
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}

	// Execute handler
	handlers[0](&PREvent{})
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

// Test Workflow Processor

func TestWorkflowProcessor_Configure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := WorkflowProcessorConfig{
		MaxConcurrentRuns: 10,
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
		Timeout:           30 * time.Second,
		EnableMetrics:     true,
		AutoRetry:         true,
		MaxHistory:        100,
	}

	if config.MaxConcurrentRuns == 0 {
		t.Error("MaxConcurrentRuns should be set")
	}

	if config.RetryDelay < 1 * time.Second {
		t.Error("RetryDelay should be at least 1 second")
	}
}

func TestPendingWorkflow(t *testing.T) {
	workflow := &PendingWorkflow{
		WorkflowRun: &WorkflowRun{
			ID:       123,
			Name:     "test-workflow",
			Status:   "in_progress",
			Conclusion: "",
		},
		StartedAt:  time.Now(),
		RetryCount: 0,
	}

	if workflow.ID != 123 {
		t.Errorf("Expected ID 123, got %d", workflow.ID)
	}

	if workflow.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", workflow.Status)
	}
}

// Test workflow run types
func TestWorkflowRun_Types(t *testing.T) {
	run := &WorkflowRun{
		ID:           1,
		Name:         "CI/CD Pipeline",
		HeadBranch:   "main",
		HeadSha:      "abc123",
		RunNumber:    1,
		Event:        "push",
		Status:       "completed",
		Conclusion:   "success",
		WorkflowID:   12345,
		CheckSuiteID: 67890,
		URL:          "https://github.com/test/repo/actions/runs/1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Actor: User{
			Login: "test-user",
			ID:    1,
		},
		AttemptNumber: 1,
		HeadCommit: &Commit{
			Sha:     "abc123",
			Message: "Test commit",
			Author: CommitAuthor{
				Name:  "Test Author",
				Email: "test@example.com",
				Date:  time.Now(),
			},
		},
	}

	if run.Name != "CI/CD Pipeline" {
		t.Errorf("Expected name 'CI/CD Pipeline', got '%s'", run.Name)
	}

	if run.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", run.Status)
	}

	if run.Conclusion != "success" {
		t.Errorf("Expected conclusion 'success', got '%s'", run.Conclusion)
	}
}

// Test repository types
func TestRepository_Types(t *testing.T) {
	repo := &Repository{
		ID:           1,
		Name:         "test-repo",
		FullName:     "test-owner/test-repo",
		Description:  "Test repository",
		HTMLURL:      "https://github.com/test-owner/test-repo",
		SURL:         "https://github.com/test-owner/test-repo.git",
		CloneURL:     "https://github.com/test-owner/test-repo.git",
		GitURL:       "git://github.com/test-owner/test-repo.git",
		SSHURL:       "git@github.com:test-owner/test-repo.git",
		Language:     "go",
		Visibility:   "public",
		DefaultBranch: "main",
		Private:      false,
		Owner: User{
			Login: "test-owner",
			ID:    1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		PushedAt:  time.Now(),
		Archived:  false,
	}

	if repo.Name != "test-repo" {
		t.Errorf("Expected name 'test-repo', got '%s'", repo.Name)
	}

	if repo.FullName != "test-owner/test-repo" {
		t.Errorf("Expected full_name 'test-owner/test-repo', got '%s'", repo.FullName)
	}

	if repo.Visibility != "public" {
		t.Errorf("Expected visibility 'public', got '%s'", repo.Visibility)
	}
}

// Test PR types
func TestPREvent_Types(t *testing.T) {
	pr := &PREvent{
		Action: "opened",
		Number: 1,
		State:  "open",
		Title:  "Test PR",
		Body:   "Test body",
		Repository: Repository{
			Name:     "test-repo",
			FullName: "test-owner/test-repo",
		},
		Head: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
			User User   `json:"user"`
		}{
			Ref: "feature-branch",
			SHA: "abc123",
			User: User{
				Login: "test-user",
				ID:    1,
			},
		},
		Base: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
			User User   `json:"user"`
		}{
			Ref: "main",
			SHA: "def456",
			User: User{
				Login: "test-owner",
				ID:    1,
			},
		},
		HeadCommit: &Commit{
			Sha:     "abc123",
			Message: "Test commit",
			Author: CommitAuthor{
				Name:  "Test Author",
				Email: "test@example.com",
			},
		},
	}

	if pr.Action != "opened" {
		t.Errorf("Expected action 'opened', got '%s'", pr.Action)
	}

	if pr.Number != 1 {
		t.Errorf("Expected number 1, got %d", pr.Number)
	}

	if pr.State != "open" {
		t.Errorf("Expected state 'open', got '%s'", pr.State)
	}
}
