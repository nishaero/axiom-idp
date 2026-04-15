package jenkins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// JenkinsClient represents a Jenkins API client
type JenkinsClient struct {
	url        string
	username   string
	apiToken   string
	httpClient *http.Client
	config     ClientConfig
	logger     *logrus.Entry
}

// ClientConfig contains Jenkins client configuration
type ClientConfig struct {
	URL           string
	Username      string
	APIToken      string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	EnableMetrics bool
}

// Job represents a Jenkins job
type Job struct {
	Name         string `json:"name"`
	Color        string `json:"color"`
	URL          string `json:"url"`
	Description  string `json:"description"`
	JobType      string `json:"job_type"`
	Buildable    bool   `json:"buildable"`
	Queueable    bool   `json:"queueable"`
	CurrentBuild *Build `json:"current_build"`
}

// Build represents a Jenkins build
type Build struct {
	Number            int       `json:"number"`
	URL               string    `json:"url"`
	Status            string    `json:"status"`
	Timestamp         time.Time `json:"timestamp"`
	Result            string    `json:"result"`
	Duration          int       `json:"duration"`
	EstimatedDuration int       `json:"estimated_duration"`
	CauseSummary      string    `json:"cause_summary"`
	Changesets        []Change  `json:"change_set"`
}

// Change represents a change in a build
type Change struct {
	Messages    []string `json:"messages"`
	Author      User     `json:"author"`
	AuthorEmail string   `json:"author_email"`
	CommitId    string   `json:"commit_id"`
	Comment     string   `json:"comment"`
}

// User represents a Jenkins user
type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// JobParams represents build parameters for a job
type JobParams struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// JobProperty represents job properties
type JobProperty struct {
	PropertyType  string          `json:"type"`
	Description   string          `json:"description"`
	Configuration json.RawMessage `json:"configuration"`
}

// JobConfig represents job configuration
type JobConfig struct {
	XML string `json:"-"`
}

// BuildRequest represents a build request
type BuildRequest struct {
	Parameters []JobParams `json:"parameters,omitempty"`
	Property   string      `json:"property,omitempty"`
}

// BuildStatus represents build status information
type BuildStatus struct {
	Status            string    `json:"status"`
	Result            string    `json:"result"`
	Duration          int       `json:"duration"`
	Timestamp         time.Time `json:"timestamp"`
	FullURL           string    `json:"full_url"`
	EstimatedDuration int       `json:"estimated_duration"`
}

// QueueItem represents a build request in the queue
type QueueItem struct {
	ID           int64     `json:"id"`
	URL          string    `json:"url"`
	Params       string    `json:"params"`
	Task         Job       `json:"task"`
	CanQueue     bool      `json:"can_queue"`
	Blocked      bool      `json:"blocked"`
	InQueueSince time.Time `json:"in_queue_since"`
	Executor     string    `json:"executor"`
	CauseBy      string    `json:"cause_by"`
}

// Pipeline represents a Jenkins Pipeline
type Pipeline struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	Type         string        `json:"type"`
	LastBuild    *Build        `json:"last_build"`
	QueueItem    *QueueItem    `json:"queue_item"`
	Properties   []JobProperty `json:"properties"`
	PipelineType string        `json:"pipeline_type"`
}

// JenkinsUser represents a Jenkins user
type JenkinsUser struct {
	Login       string   `json:"login"`
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Email       string   `json:"email"`
	URL         string   `json:"url"`
	AvatarURL   string   `json:"avatar_url"`
	Permissions []string `json:"permissions"`
}

// NewJenkinsClient creates a new Jenkins client
func NewJenkinsClient(logger *logrus.Logger, config ClientConfig) *JenkinsClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &JenkinsClient{
		url:        config.URL,
		username:   config.Username,
		apiToken:   config.APIToken,
		httpClient: client,
		config:     config,
		logger:     logger.WithField("component", "jenkins_client"),
	}
}

// Validate validates the client configuration
func (c *JenkinsClient) Validate() error {
	if c.config.URL == "" {
		return fmt.Errorf("Jenkins URL is required")
	}

	if c.config.Username == "" {
		return fmt.Errorf("username is required")
	}

	if c.config.APIToken == "" {
		return fmt.Errorf("API token is required")
	}

	if c.config.Timeout < 1*time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	return nil
}

// GetJob fetches job information
func (c *JenkinsClient) GetJob(ctx context.Context, jobName string) (*Job, error) {
	c.logger.WithField("job", jobName).Debug("Fetching job")

	url := fmt.Sprintf("%s/job/%s/api/json", c.url, url.PathEscape(jobName))

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get job (status %d): %s", resp.StatusCode, string(body))
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode job response: %w", err)
	}

	return &job, nil
}

// BuildJob triggers a build for a job
func (c *JenkinsClient) BuildJob(ctx context.Context, jobName string, params []JobParams) (*Build, error) {
	c.logger.WithFields(logrus.Fields{
		"job":    jobName,
		"params": len(params),
	}).Debug("Building job")

	url := fmt.Sprintf("%s/job/%s/build", c.url, url.PathEscape(jobName))

	// Build request body
	body := BuildRequest{Parameters: params}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to trigger build (status %d): %s", resp.StatusCode, string(body))
	}

	// Get build number from response
	buildNumber, err := c.getBuildNumber(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	// Get build status
	return c.GetBuildStatus(ctx, jobName, buildNumber)
}

// GetBuildStatus fetches build status information
func (c *JenkinsClient) GetBuildStatus(ctx context.Context, jobName string, buildNumber int) (*Build, error) {
	c.logger.WithFields(logrus.Fields{
		"job":   jobName,
		"build": buildNumber,
	}).Debug("Fetching build status")

	url := fmt.Sprintf("%s/job/%s/%d/api/json", c.url, url.PathEscape(jobName), buildNumber)

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get build status (status %d): %s", resp.StatusCode, string(body))
	}

	var build Build
	if err := json.NewDecoder(resp.Body).Decode(&build); err != nil {
		return nil, fmt.Errorf("failed to decode build response: %w", err)
	}

	return &build, nil
}

// GetBuildLog fetches build log
func (c *JenkinsClient) GetBuildLog(ctx context.Context, jobName string, buildNumber int) (string, error) {
	c.logger.WithFields(logrus.Fields{
		"job":   jobName,
		"build": buildNumber,
	}).Debug("Fetching build log")

	url := fmt.Sprintf("%s/job/%s/%d/log", c.url, url.PathEscape(jobName), buildNumber)

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get build log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get build log (status %d): %s", resp.StatusCode, string(body))
	}

	log, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read build log: %w", err)
	}

	return string(log), nil
}

// CancelBuild cancels a running build
func (c *JenkinsClient) CancelBuild(ctx context.Context, jobName string, buildNumber int) error {
	c.logger.WithFields(logrus.Fields{
		"job":   jobName,
		"build": buildNumber,
	}).Debug("Cancelling build")

	url := fmt.Sprintf("%s/job/%s/%d/stop", c.url, url.PathEscape(jobName), buildNumber)

	req, err := c.buildRequest(ctx, http.MethodPost, url)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel build (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetJobs lists all jobs
func (c *JenkinsClient) GetJobs(ctx context.Context) ([]Job, error) {
	c.logger.Debug("Fetching jobs")

	url := fmt.Sprintf("%s/api/json", c.url)

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get jobs (status %d): %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode jobs response: %w", err)
	}

	var jobs []Job
	if jobsData, exists := response["jobs"]; exists {
		if j, ok := jobsData.([]interface{}); ok {
			for _, item := range j {
				if job, ok := item.(map[string]interface{}); ok {
					jobs = append(jobs, Job{
						Name:  fmt.Sprintf("%v", job["name"]),
						URL:   fmt.Sprintf("%v", job["url"]),
						Color: fmt.Sprintf("%v", job["color"]),
					})
				}
			}
		}
	}

	return jobs, nil
}

// GetPipeline fetches pipeline information
func (c *JenkinsClient) GetPipeline(ctx context.Context, pipelineName string) (*Pipeline, error) {
	c.logger.WithField("pipeline", pipelineName).Debug("Fetching pipeline")

	url := fmt.Sprintf("%s/job/%s/api/json", c.url, url.PathEscape(pipelineName))

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get pipeline (status %d): %s", resp.StatusCode, string(body))
	}

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline response: %w", err)
	}

	return &pipeline, nil
}

// GetQueueItem fetches queue item information
func (c *JenkinsClient) GetQueueItem(ctx context.Context, queueID int64) (*QueueItem, error) {
	c.logger.WithField("queue_id", queueID).Debug("Fetching queue item")

	url := fmt.Sprintf("%s/queue/item/%d/api/json", c.url, queueID)

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get queue item (status %d): %s", resp.StatusCode, string(body))
	}

	var item QueueItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode queue item response: %w", err)
	}

	return &item, nil
}

// GetQueueItems lists all queue items
func (c *JenkinsClient) GetQueueItems(ctx context.Context) ([]QueueItem, error) {
	c.logger.Debug("Fetching queue items")

	url := fmt.Sprintf("%s/api/json?tree=queue[model,name,url]", c.url)

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get queue items (status %d): %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode queue items response: %w", err)
	}

	var items []QueueItem
	if queueData, exists := response["queue"]; exists {
		if queueRaw, ok := queueData.(map[string]interface{}); ok {
			if itemsData, exists := queueRaw["items"]; exists {
				var itemsRaw []interface{}
				if itemsJSON, err := json.Marshal(itemsData); err == nil {
					if err := json.Unmarshal(itemsJSON, &itemsRaw); err != nil {
						return nil, fmt.Errorf("failed to parse queue items: %w", err)
					}
					for _, itemRaw := range itemsRaw {
						itemJSON, err := json.Marshal(itemRaw)
						if err != nil {
							return nil, fmt.Errorf("failed to encode queue item: %w", err)
						}
						var item QueueItem
						if err := json.Unmarshal(itemJSON, &item); err != nil {
							return nil, fmt.Errorf("failed to decode queue item: %w", err)
						}
						items = append(items, item)
					}
				}
			}
		}
	}

	return items, nil
}

// GetJobConfiguration fetches job configuration
func (c *JenkinsClient) GetJobConfiguration(ctx context.Context, jobName string) (string, error) {
	c.logger.WithField("job", jobName).Debug("Fetching job configuration")

	url := fmt.Sprintf("%s/job/%s/config.xml", c.url, url.PathEscape(jobName))

	req, err := c.buildRequest(ctx, http.MethodGet, url)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get job configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get job configuration (status %d): %s", resp.StatusCode, string(body))
	}

	config, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read job configuration: %w", err)
	}

	return string(config), nil
}

// UpdateJobConfiguration updates job configuration
func (c *JenkinsClient) UpdateJobConfiguration(ctx context.Context, jobName string, config string) error {
	c.logger.WithFields(logrus.Fields{
		"job":    jobName,
		"length": len(config),
	}).Debug("Updating job configuration")

	url := fmt.Sprintf("%s/job/%s/config.xml", c.url, url.PathEscape(jobName))

	req, err := c.buildRequest(ctx, http.MethodPost, url)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Body = io.NopCloser(bytes.NewBufferString(config))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update job configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update job configuration (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetHealthStatus returns health status of Jenkins client
func (c *JenkinsClient) GetHealthStatus(ctx context.Context) map[string]interface{} {
	url := fmt.Sprintf("%s/api/json", c.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  err.Error(),
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  err.Error(),
		}
	}
	defer resp.Body.Close()

	return map[string]interface{}{
		"status":   "healthy",
		"timeout":  c.config.Timeout.String(),
		"url":      c.url,
		"has_auth": c.username != "" && c.apiToken != "",
	}
}

// CloseConnection closes the HTTP client connection
func (c *JenkinsClient) CloseConnection() {
	c.httpClient.CloseIdleConnections()
}

// Helper functions

func (c *JenkinsClient) buildRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	// Add authentication
	if c.username != "" && c.apiToken != "" {
		req.SetBasicAuth(c.username, c.apiToken)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *JenkinsClient) getBuildNumber(location string) (int, error) {
	// Extract build number from location URL
	// Example: /job/test-job/123/
	parts := bytes.Split([]byte(location), []byte("/"))
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid build location: %s", location)
	}

	var buildNumber int
	if _, err := fmt.Sscanf(string(parts[len(parts)-1]), "%d", &buildNumber); err != nil {
		return 0, fmt.Errorf("failed to scan build number from location %q: %w", location, err)
	}

	if buildNumber == 0 {
		return 0, fmt.Errorf("failed to parse build number from: %s", location)
	}

	return buildNumber, nil
}
