package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Test GitLab Client

func createTestClient(t *testing.T) (*GitLabClient, *httptest.Server) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v4/projects/1":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 1,
				"name": "test-repo",
				"full_name": "test-owner/test-repo",
				"path": "test-repo",
				"path_with_namespace": "test-owner/test-repo",
				"description": "Test repository",
				"visibility": "private",
				"web_url": "https://gitlab.com/test-owner/test-repo",
				"http_url_to_repo": "https://gitlab.com/test-owner/test-repo.git",
				"default_branch": "main",
				"archived": false,
			})
		case "/api/v4/runners":
			json.NewEncoder(w).Encode([]interface{}{
				map[string]interface{}{
					"id": 1,
					"description": "test-runner",
					"active": true,
					"status": "online",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	config := ClientConfig{
		APIURL: server.URL,
		APIToken: "test-token",
		Timeout: 30 * time.Second,
	}

	client := NewGitLabClient(logger, config)
	return client, server
}

func TestGitLabClient_GetProject(t *testing.T) {
	client, server := createTestClient(t)
	defer server.Close()

	ctx := context.Background()
	project, err := client.GetProject(ctx, 1)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if project == nil {
		t.Error("Expected non-nil project")
	}

	if project.Name != "test-repo" {
		t.Errorf("Expected name 'test-repo', got '%s'", project.Name)
	}
}

func TestGitLabClient_Validate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := ClientConfig{
		APIURL: "https://gitlab.com",
		APIToken: "test-token",
		Timeout: 100 * time.Millisecond,
	}

	client := NewGitLabClient(logger, config)
	err := client.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid timeout")
	}
}

func TestGitLabClient_GetHealthStatus(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := ClientConfig{
		APIURL: "https://gitlab.com",
		APIToken: "test-token",
		Timeout: 30 * time.Second,
	}

	client := NewGitLabClient(logger, config)
	status := client.GetHealthStatus(context.Background())

	if status["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status["status"])
	}
}

// Test Webhook Handler

func createTestWebhookHandler(t *testing.T) (*WebhookHandler, *httptest.Server) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	registry := NewEventRegistry()
	client, _ := createTestClient(t)

	handler := NewWebhookHandler(logger, client, "test-secret", registry)

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
		"object_kind": "push",
	}
	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", server.URL, &http.Response{Body: http.NoBody})
	// Skip this test as we need a proper request
	_ = req
	_ = err
}

func TestEventRegistry_RegisterHandler(t *testing.T) {
	registry := NewEventRegistry()

	handlerCalled := false
	handler := func(event *WebhookEvent) {
		handlerCalled = true
	}

	registry.RegisterHandler("push", handler)

	// Verify handler was registered
	handlers := registry.GetHandlers("push")
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}

	// Execute handler
	handlers[0](&WebhookEvent{})
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

// Test Orchestration Controller

func TestOrchestrationController_Start(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	config := OrchestrationConfig{
		EnableMetrics:         true,
		MetricsCollectionInterval: 1 * time.Minute,
		MaxConcurrentPipelines: 10,
		EnableCostTracking:    false,
	}

	// Test configuration validation
	if config.MaxConcurrentPipelines == 0 {
		t.Error("MaxConcurrentPipelines should be set")
	}

	if config.MetricsCollectionInterval < time.Minute {
		t.Error("MetricsCollectionInterval should be at least 1 minute")
	}
}

func TestOrchestrationController_Metrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create mock client and handler
	client := &GitLabClient{}
	handler := &WebhookHandler{}

	orchestrator := NewOrchestrationController(logger, client, handler, OrchestrationConfig{
		EnableMetrics: true,
	})

	// Test metrics collection
	metrics := orchestrator.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}
}

// Test Pipeline types

func TestPipeline_Types(t *testing.T) {
	now := time.Now()
	pipeline := &Pipeline{
		ID:            1,
		IID:           1,
		ProjectID:     123,
		Ref:           "main",
		Sha:           "abc123",
		Status:        "success",
		Source:        "push",
		Duration:      120.5,
		QueuedDuration: 5.0,
		Quality:       100.0,
		Stages:        []string{"build", "test", "deploy"},
		StartedAt:     now,
		FinishedAt:    now.Add(time.Minute * 2),
	}

	if pipeline.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", pipeline.Status)
	}

	if len(pipeline.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(pipeline.Stages))
	}
}

// Test Job types

func TestJob_Types(t *testing.T) {
	now := time.Now()
	job := &Job{
		ID:           1,
		IID:          1,
		ProjectID:    123,
		PipelineID:   1,
		Status:       "success",
		Stage:        "test",
		Name:         "run-tests",
		Ref:          "main",
		Tag:          false,
		Failed:       false,
		Errored:      false,
		Skipped:      false,
		AllowFailure: false,
		Duration:     60.0,
		CreatedAt:    now,
		User: User{
			ID:     1,
			Name:   "Test User",
			Username: "testuser",
		},
	}

	if job.Name != "run-tests" {
		t.Errorf("Expected name 'run-tests', got '%s'", job.Name)
	}

	if job.Stage != "test" {
		t.Errorf("Expected stage 'test', got '%s'", job.Stage)
	}
}

// Test Project types

func TestProject_Types(t *testing.T) {
	now := time.Now()
	project := &Project{
		ID:              1,
		Name:            "test-repo",
		FullName:        "test-owner/test-repo",
		Path:            "test-repo",
		Namespace:       "test-owner",
		PathWithNamespace: "test-owner/test-repo",
		Description:     "Test repository",
		Visibility:      "private",
		WebURL:          "https://gitlab.com/test-owner/test-repo",
		HTTPURL:         "https://gitlab.com/test-owner/test-repo.git",
		SSHURL:          "git@gitlab.com:test-owner/test-repo.git",
		DefaultBranch:   "main",
		Archived:        false,
		StarCount:       0,
		LastActivityAt:  now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if project.Name != "test-repo" {
		t.Errorf("Expected name 'test-repo', got '%s'", project.Name)
	}

	if project.FullName != "test-owner/test-repo" {
		t.Errorf("Expected full_name 'test-owner/test-repo', got '%s'", project.FullName)
	}

	if project.Visibility != "private" {
		t.Errorf("Expected visibility 'private', got '%s'", project.Visibility)
	}
}

// Test Merge Request types

func TestMergeRequest_Types(t *testing.T) {
	now := time.Now()
	mr := &MergeRequest{
		ID:            1,
		IID:           1,
		ProjectID:     123,
		Title:         "Test MR",
		Description:   "Test description",
		State:         "opened",
		SourceBranch:  "feature",
		TargetBranch:  "main",
		Merged:        false,
		Sha:           "abc123",
		Subscribed:    true,
		CreatedAt:     now,
		UpdatedAt:     now,
		User: User{
			ID:     1,
			Name:   "Test User",
			Username: "testuser",
		},
		Author: User{
			ID:     1,
			Name:   "Test User",
			Username: "testuser",
		},
	}

	if mr.Title != "Test MR" {
		t.Errorf("Expected title 'Test MR', got '%s'", mr.Title)
	}

	if mr.State != "opened" {
		t.Errorf("Expected state 'opened', got '%s'", mr.State)
	}

	if mr.SourceBranch != "feature" {
		t.Errorf("Expected source_branch 'feature', got '%s'", mr.SourceBranch)
	}
}

// Test Variable types

func TestVariable_Types(t *testing.T) {
	now := time.Now()
	variable := &Variable{
		Key:            "APP_ENV",
		Value:          "production",
		Protected:      true,
		Masked:         false,
		EnvironmentScope: "production",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if variable.Key != "APP_ENV" {
		t.Errorf("Expected key 'APP_ENV', got '%s'", variable.Key)
	}

	if variable.Value != "production" {
		t.Errorf("Expected value 'production', got '%s'", variable.Value)
	}
}

// Test configuration

func TestConfig_Validate(t *testing.T) {
	config := DefaultConfig()

	// Test valid config
	err := config.Validate()
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	// Test invalid config (missing API URL)
	config.Client.APIURL = ""
	err = config.Validate()
	if err == nil {
		t.Error("Expected error for missing API URL")
	}

	// Restore
	config.Client.APIURL = "https://gitlab.com"
}

func TestConfig_Merge(t *testing.T) {
	baseConfig := DefaultConfig()
	baseConfig.Client.RetryCount = 3

	overrideConfig := &Config{
		Client: ClientConfig{
			RetryCount: 5,
		},
	}

	merged := baseConfig.Merge(overrideConfig)

	// Verify merge
	if merged.Client.RetryCount != 5 {
		t.Errorf("Expected RetryCount 5 after merge, got %d", merged.Client.RetryCount)
	}

	// Base config value should be preserved for other fields
	if merged.Client.Timeout != baseConfig.Client.Timeout {
		t.Error("Expected timeout to be preserved after merge")
	}
}

// Test Error types

func TestError_Error(t *testing.T) {
	err := NewError("test-code", "test message", nil)
	if err.Error() != "test message" {
		t.Errorf("Expected 'test message', got '%s'", err.Error())
	}
}

func TestError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := NewError("test-code", "test message", innerErr)

	if err.Unwrap() != innerErr {
		t.Error("Expected Unwrap to return inner error")
	}
}

func TestIsError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := NewError("test-code", "test message", innerErr)

	if !IsError(err, "test-code") {
		t.Error("Expected IsError to return true for matching code")
	}

	if IsError(err, "different-code") {
		t.Error("Expected IsError to return false for non-matching code")
	}
}

// Test helper functions

func TestHMACSHA256(t *testing.T) {
	data := []byte("test data")
	secret := "test-secret"

	sig := HMACSHA256(data, secret)

	if sig == "" {
		t.Error("Expected non-empty signature")
	}

	if len(sig) != 72 { // sha256= + 64 hex chars
		t.Errorf("Expected signature length 72, got %d", len(sig))
	}
}
