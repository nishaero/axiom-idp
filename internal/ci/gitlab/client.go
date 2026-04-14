package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// GitLabClient represents a GitLab API v4 client
type GitLabClient struct {
	apiURL       string
	apiToken     string
	httpClient   *http.Client
	config       ClientConfig
	logger       *logrus.Logger
}

// ClientConfig contains GitLab client configuration
type ClientConfig struct {
	APIURL        string
	APIToken      string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	EnableMetrics bool
	RateLimit     int
}

// WebhookConfig contains webhook configuration
type WebhookConfig struct {
	Path           string
	Secret         string
	VerifySSL      bool
	AllowedEvents  []string
	EnableAuditLog bool
}


// Project represents a GitLab project
type Project struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Path            string    `json:"path"`
	Namespace       string    `json:"namespace"`
	PathWithNamespace string  `json:"path_with_namespace"`
	Description     string    `json:"description"`
	Visibility      string    `json:"visibility"`
	WebURL          string    `json:"web_url"`
	HTTPURL         string    `json:"http_url_to_repo"`
	SSHURL          string    `json:"ssh_url_to_repo"`
	DefaultBranch   string    `json:"default_branch"`
	Archived        bool      `json:"archived"`
	LastActivityAt  time.Time `json:"last_activity_at"`
	StarCount       int       `json:"star_count"`
	ForkCount       int       `json:"fork_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Pipeline represents a GitLab CI/CD pipeline
type Pipeline struct {
	ID            int64       `json:"id"`
	 IID          int64       `json:"iid"`
	ProjectID     int64       `json:"project_id"`
	Ref           string      `json:"ref"`
	Sha           string      `json:"sha"`
	Status        string      `json:"status"`
	Source        string      `json:"source"`
	BeforeSha     string      `json:"before_sha"`
	Tag           bool        `json:"tag"`
	YamlErrors    string      `json:"yaml_errors"`
	User          User        `json:"user"`
	StartedAt     time.Time   `json:"started_at"`
	FinishedAt    time.Time   `json:"finished_at"`
	Duration      float64     `json:"duration"`
	QueuedDuration float64    `json:"queued_duration"`
	Quality       float64     `json:"quality"`
	Failures      int         `json:"failed_job_count"`
	StatusChangedAt time.Time  `json:"status_changed_at"`
	Stages        []string    `json:"stages"`
	JobStatus     string      `json:"job_status"`
}

// Job represents a GitLab CI/CD job
type Job struct {
	ID             int64     `json:"id"`
	IID            int       `json:"iid"`
	ProjectID      int64     `json:"project_id"`
	PipelineID     int64     `json:"pipeline_id"`
	Status         string    `json:"status"`
	Stage          string    `json:"stage"`
	Name           string    `json:"name"`
	Ref            string    `json:"ref"`
	Tag            bool      `json:"tag"`
	Failed         bool      `json:"failed"`
	Errored        bool      `json:"errored"`
	Skipped        bool      `json:"skipped"`
	AllowFailure   bool      `json:"allow_failure"`
	Duration       float64   `json:"duration"`
	QueuedDuration float64   `json:"queued_duration"`
	QueuedAt       time.Time `json:"queued_at"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	ArtifactsCount int       `json:"artifacts_count"`
	Pipeline       Pipeline  `json:"pipeline"`
	Project        Project   `json:"project"`
	User           User      `json:"user"`
}

// Runner represents a GitLab runner
type Runner struct {
	ID             int64     `json:"id"`
	Description    string    `json:"description"`
	IsShared       bool      `json:"shared"`
	Active         bool      `json:"active"`
	Status         string    `json:"status"`
	Token          string    `json:"token"`
	TokenExpiresAt time.Time `json:"token_expires_at"`
	Online         bool      `json:"online"`
	Type           string    `json:"type"`
	Limit          int       `json:"limit"`
	Tags           []string  `json:"tags"`
	Locked         bool      `json:"locked"`
}

// MergeRequest represents a GitLab merge request (MR)
type MergeRequest struct {
	ID              int64     `json:"id"`
	IID             int       `json:"iid"`
	ProjectID       int64     `json:"project_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	State           string    `json:"state"`
	SourceBranch    string    `json:"source_branch"`
	TargetBranch    string    `json:"target_branch"`
	Merged          bool      `json:"merged"`
	MergedBy        *User     `json:"merged_by"`
	MergedAt        time.Time `json:"merged_at"`
	Sha             string    `json:"sha"`
	DiffSha         string    `json:"diff_sha"`
	Subscribed      bool      `json:"subscribed"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClosedAt        time.Time `json:"closed_at"`
	TargetProjectID int64     `json:"target_project_id"`
	User            User      `json:"user"`
	Author          User      `json:"author"`
	Assignee        *User     `json:"assignee"`
	MergeWhenPipelineSucceeds bool `json:"merge_when_pipeline_succeeds"`
	Repository      struct {
		URL      string `json:"url"`
		Format   string `json:"format"`
	} `json:"repository"`
}

// User represents a GitLab user
type User struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	State      string    `json:"state"`
	AvatarURL  string    `json:"avatar_url"`
	WebURL     string    `json:"web_url"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
}

// Variable represents a GitLab CI variable
type Variable struct {
	Key            string    `json:"key"`
	Value          string    `json:"value"`
	Protected      bool      `json:"protected"`
	Masked         bool      `json:"masked"`
	EnvironmentScope string  `json:"environment_scope"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// WebhookEvent represents a GitLab webhook event
type WebhookEvent struct {
	EventType      string              `json:"object_kind"`
	EventName      string              `json:"event_name"`
	Project        Project             `json:"project"`
	Pipeline       *Pipeline           `json:"pipeline,omitempty"`
	Job            *Job                `json:"build,omitempty"`
	Runner         *Runner             `json:"runner,omitempty"`
	MergeRequest   *MergeRequest       `json:"merge_request,omitempty"`
	Ref            string              `json:"ref,omitempty"`
	BeforeSha      string              `json:"before_sha,omitempty"`
	AfterSha       string              `json:"after_sha,omitempty"`
	Commits        []Commit            `json:"commits,omitempty"`
	Author         *User               `json:"author,omitempty"`
	Labels         []map[string]interface{} `json:"labels,omitempty"`
	TriggerType    string              `json:"ref_protected,omitempty"`
}

// Commit represents a commit in GitLab
type Commit struct {
	ID        string    `json:"id"`
	SHA       string    `json:"short_id"`
	Message   string    `json:"message"`
	Author    CommitAuthor `json:"author"`
	Committer CommitAuthor `json:"committer"`
	Stats     CommitStats `json:"stats"`
	Duration  float64   `json:"duration_seconds,omitempty"`
	Status    string    `json:"status,omitempty"`
}

// CommitAuthor represents commit author information
type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CommitStats represents commit statistics
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

// PipelineJob represents a job in a pipeline
type PipelineJob struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Stage       string    `json:"stage"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Duration    float64   `json:"duration"`
	AllowFailure bool     `json:"allow_failure"`
	Artifacts   []Artifact `json:"artifacts"`
}

// Artifact represents a pipeline artifact
type Artifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int    `json:"size"`
}

// Environment represents a GitLab environment
type Environment struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	ExternalURL    string    `json:"external_url"`
	State          string    `json:"state"`
	Visibility     string    `json:"visibility"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastDeployment time.Time `json:"last_deployment_timestamp"`
}

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(logger *logrus.Logger, config ClientConfig) *GitLabClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &GitLabClient{
		apiURL:     config.APIURL,
		apiToken:   config.APIToken,
		httpClient: client,
		config:     config,
		logger:     logger,
	}
}

// Validate validates the client configuration
func (c *GitLabClient) Validate() error {
	if c.config.APIURL == "" {
		return fmt.Errorf("GitLab API URL is required")
	}

	if c.config.APIToken == "" {
		return fmt.Errorf("API token is required")
	}

	if c.config.Timeout < 1*time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	// Validate API URL format
	if !strings.HasPrefix(c.config.APIURL, "http://") && !strings.HasPrefix(c.config.APIURL, "https://") {
		return fmt.Errorf("API URL must include protocol (http:// or https://)")
	}

	return nil
}

// GetBaseURL returns the base API URL
func (c *GitLabClient) GetBaseURL() string {
	return c.apiURL
}

// buildRequest creates an HTTP request with authentication
func (c *GitLabClient) buildRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	fullURL := c.apiURL
	if !strings.HasSuffix(c.apiURL, "/") {
		fullURL = c.apiURL + "/"
	}
	fullURL += strings.TrimPrefix(endpoint, "/")

	var req *http.Request
	var err error

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.apiToken)

	return req, nil
}

// executeRequest executes an HTTP request with retry logic
func (c *GitLabClient) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= c.config.RetryCount; i++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			c.logger.WithFields(logrus.Fields{
				"error":  err,
				"attempt": i + 1,
			}).Warn("Request failed, will retry")

			if i < c.config.RetryCount {
				time.Sleep(c.config.RetryDelay)
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", i+1, err)
		}

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			c.logger.Warn("Rate limit exceeded, will retry")
			if i < c.config.RetryCount {
				retryAfter := resp.Header.Get("Retry-After")
				var sleepTime time.Duration
				if retryAfter != "" {
					if seconds, err := strconv.Atoi(retryAfter); err == nil {
						sleepTime = time.Duration(seconds) * time.Second
					}
				}
				if sleepTime == 0 {
					sleepTime = c.config.RetryDelay
				}
				time.Sleep(sleepTime)
				continue
			}
			return nil, fmt.Errorf("rate limit exceeded")
		}

		break
	}

	return resp, nil
}

// readResponse reads and parses a JSON response
func readResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("API request failed (status %d): %s", resp.StatusCode, string(body))
	}

	if len(body) > 0 && target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GetProject fetches project information
func (c *GitLabClient) GetProject(ctx context.Context, projectID int) (*Project, error) {
	c.logger.WithField("project_id", projectID).Debug("Fetching project")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d", projectID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := readResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectByPath fetches project information by path
func (c *GitLabClient) GetProjectByPath(ctx context.Context, namespace, name string) (*Project, error) {
	c.logger.WithFields(logrus.Fields{
		"namespace": namespace,
		"name":      name,
	}).Debug("Fetching project by path")

	encodedPath := url.PathEscape(namespace) + "/" + url.PathEscape(name)

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%s", encodedPath), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := readResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// ListProjects lists projects
func (c *GitLabClient) ListProjects(ctx context.Context, opts *ListProjectsOptions) ([]Project, error) {
	c.logger.Debug("Listing projects")

	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Search != "" {
			params.Set("search", opts.Search)
		}
		if opts.Owned != nil && *opts.Owned {
			params.Set("owned", "true")
		}
		if opts.Membership != nil && *opts.Membership {
			params.Set("membership", "true")
		}
		if opts.Visibility != "" {
			params.Set("visibility", opts.Visibility)
		}
	}

	requestURL := "projects?" + params.Encode()

	req, err := c.buildRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := readResponse(resp, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// ListProjectsOptions contains options for listing projects
type ListProjectsOptions struct {
	Page        int
	PerPage     int
	Search      string
	Owned       *bool
	Membership  *bool
	Visibility  string
}

// CreatePipeline creates a new pipeline
func (c *GitLabClient) CreatePipeline(ctx context.Context, projectID int, ref, message string) (*Pipeline, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"ref":        ref,
		"message":    message,
	}).Debug("Creating pipeline")

	body := map[string]string{
		"ref":       ref,
		"variables": `{"AXIOM_TRIGGER":"true"}`,
	}

	if message != "" {
		body["status"] = "running"
		body["message"] = message
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf("projects/%d/pipeline", projectID), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := readResponse(resp, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

// GetPipeline fetches pipeline information
func (c *GitLabClient) GetPipeline(ctx context.Context, projectID int, pipelineID int64) (*Pipeline, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"pipeline_id": pipelineID,
	}).Debug("Fetching pipeline")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/pipelines/%d", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := readResponse(resp, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

// GetPipelineByIID fetches pipeline by its IID (project-level ID)
func (c *GitLabClient) GetPipelineByIID(ctx context.Context, projectID int, pipelineIID int64) (*Pipeline, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id":   projectID,
		"pipeline_iid": pipelineIID,
	}).Debug("Fetching pipeline by IID")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/pipelines/%d", projectID, pipelineIID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := readResponse(resp, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

// ListPipelines lists pipelines for a project
func (c *GitLabClient) ListPipelines(ctx context.Context, projectID int, opts *ListPipelinesOptions) ([]Pipeline, error) {
	c.logger.WithField("project_id", projectID).Debug("Listing pipelines")

	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Ref != "" {
			params.Set("ref", opts.Ref)
		}
		if opts.Branch != "" {
			params.Set("branch", opts.Branch)
		}
		if opts.SHA != "" {
			params.Set("sha", opts.SHA)
		}
	}

	requestURL := fmt.Sprintf("projects/%d/pipelines?%s", projectID, params.Encode())

	req, err := c.buildRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var pipelines []Pipeline
	if err := readResponse(resp, &pipelines); err != nil {
		return nil, err
	}

	return pipelines, nil
}

// ListPipelinesOptions contains options for listing pipelines
type ListPipelinesOptions struct {
	Page   int
	PerPage int
	Status string
	Ref    string
	Branch string
	SHA    string
}

// GetPipelineJobs fetches jobs in a pipeline
func (c *GitLabClient) GetPipelineJobs(ctx context.Context, projectID int, pipelineID int64) ([]Job, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"pipeline_id": pipelineID,
	}).Debug("Fetching pipeline jobs")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/pipelines/%d/jobs", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var jobs []Job
	if err := readResponse(resp, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetJob fetches job information
func (c *GitLabClient) GetJob(ctx context.Context, projectID int, jobID int64) (*Job, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"job_id":     jobID,
	}).Debug("Fetching job")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/jobs/%d", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var job Job
	if err := readResponse(resp, &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// CancelJob cancels a running job
func (c *GitLabClient) CancelJob(ctx context.Context, projectID int, jobID int64) error {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"job_id":     jobID,
	}).Debug("Cancelling job")

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf("projects/%d/jobs/%d/cancel", projectID, jobID), nil)
	if err != nil {
		return err
	}

	_, err = c.executeRequest(ctx, req)
	if err != nil {
		return err
	}

	// Cancel returns empty response on success
	return nil
}

// RetryJob retries a failed job
func (c *GitLabClient) RetryJob(ctx context.Context, projectID int, jobID int64) (*Job, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"job_id":     jobID,
	}).Debug("Retrying job")

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf("projects/%d/jobs/%d/retry", projectID, jobID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var job Job
	if err := readResponse(resp, &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// GetJobLogs fetches job logs
func (c *GitLabClient) GetJobLogs(ctx context.Context, projectID int, jobID int64) (string, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"job_id":     jobID,
	}).Debug("Fetching job logs")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/jobs/%d/trace", projectID, jobID), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read job logs: %w", err)
	}

	return string(logs), nil
}

// ListJobs lists jobs for a project
func (c *GitLabClient) ListJobs(ctx context.Context, projectID int, opts *ListJobsOptions) ([]Job, error) {
	c.logger.WithField("project_id", projectID).Debug("Listing jobs")

	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Ref != "" {
			params.Set("ref", opts.Ref)
		}
	}

	requestURL := fmt.Sprintf("projects/%d/jobs?%s", projectID, params.Encode())

	req, err := c.buildRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var jobs []Job
	if err := readResponse(resp, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// ListJobsOptions contains options for listing jobs
type ListJobsOptions struct {
	Page    int
	PerPage int
	Status  string
	Ref     string
}

// ListRunners lists all runners
func (c *GitLabClient) ListRunners(ctx context.Context) ([]Runner, error) {
	c.logger.Debug("Listing runners")

	req, err := c.buildRequest(ctx, http.MethodGet, "runners", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var runners []Runner
	if err := readResponse(resp, &runners); err != nil {
		return nil, err
	}

	return runners, nil
}

// GetRunner fetches runner information
func (c *GitLabClient) GetRunner(ctx context.Context, runnerID int64) (*Runner, error) {
	c.logger.WithField("runner_id", runnerID).Debug("Fetching runner")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("runners/%d", runnerID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var runner Runner
	if err := readResponse(resp, &runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

// ListProjectRunners lists project-specific runners
func (c *GitLabClient) ListProjectRunners(ctx context.Context, projectID int) ([]Runner, error) {
	c.logger.WithField("project_id", projectID).Debug("Listing project runners")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/runners", projectID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var runners []Runner
	if err := readResponse(resp, &runners); err != nil {
		return nil, err
	}

	return runners, nil
}

// RegisterRunner registers a new runner
func (c *GitLabClient) RegisterRunner(ctx context.Context, token, description, tagList string) (*Runner, error) {
	c.logger.WithFields(logrus.Fields{
		"description": description,
		"tags":        tagList,
	}).Debug("Registering runner")

	body := map[string]interface{}{
		"token":      token,
		"description": description,
		"tag_list":   strings.Split(tagList, ","),
	}

	req, err := c.buildRequest(ctx, http.MethodPost, "runners", body)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var runner Runner
	if err := readResponse(resp, &runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

// UnregisterRunner unregisters a runner
func (c *GitLabClient) UnregisterRunner(ctx context.Context, runnerID int64) error {
	c.logger.WithField("runner_id", runnerID).Debug("Unregistering runner")

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf("runners/%d", runnerID), nil)
	if err != nil {
		return err
	}

	_, err = c.executeRequest(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// GetMergeRequest fetches merge request information
func (c *GitLabClient) GetMergeRequest(ctx context.Context, projectID int, MRIID int) (*MergeRequest, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"mr_iid":     MRIID,
	}).Debug("Fetching merge request")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/merge_requests/%d", projectID, MRIID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var mr MergeRequest
	if err := readResponse(resp, &mr); err != nil {
		return nil, err
	}

	return &mr, nil
}

// ListMergeRequests lists merge requests for a project
func (c *GitLabClient) ListMergeRequests(ctx context.Context, projectID int, opts *ListMergeRequestsOptions) ([]MergeRequest, error) {
	c.logger.WithField("project_id", projectID).Debug("Listing merge requests")

	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.State != "" {
			params.Set("state", opts.State)
		}
		if opts.SourceBranch != "" {
			params.Set("source_branch", opts.SourceBranch)
		}
		if opts.TargetBranch != "" {
			params.Set("target_branch", opts.TargetBranch)
		}
		if opts.SortBy != "" {
			params.Set("sort", opts.SortBy)
		}
	}

	requestURL := fmt.Sprintf("projects/%d/merge_requests?%s", projectID, params.Encode())

	req, err := c.buildRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var mrs []MergeRequest
	if err := readResponse(resp, &mrs); err != nil {
		return nil, err
	}

	return mrs, nil
}

// ListMergeRequestsOptions contains options for listing merge requests
type ListMergeRequestsOptions struct {
	Page         int
	PerPage      int
	State        string
	SourceBranch string
	TargetBranch string
	SortBy       string
}

// GetMergeRequestDiff fetches merge request diff
func (c *GitLabClient) GetMergeRequestDiff(ctx context.Context, projectID int, MRIID int) ([]interface{}, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"mr_iid":     MRIID,
	}).Debug("Fetching merge request diff")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/merge_requests/%d/diffs", projectID, MRIID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var diffs []interface{}
	if err := readResponse(resp, &diffs); err != nil {
		return nil, err
	}

	return diffs, nil
}

// AcceptMergeRequest accepts a merge request
func (c *GitLabClient) AcceptMergeRequest(ctx context.Context, projectID int, MRIID int, mergeOpts map[string]interface{}) error {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"mr_iid":     MRIID,
	}).Debug("Accepting merge request")

	body := make(map[string]interface{})
	if len(mergeOpts) > 0 {
		body = mergeOpts
	}

	// Set default values if not provided
	if _, exists := body["merge_commit_message"]; !exists {
		body["merge_commit_message"] = "MR merge"
	}

	req, err := c.buildRequest(ctx, http.MethodPut, fmt.Sprintf("projects/%d/merge_requests/%d", projectID, MRIID), body)
	if err != nil {
		return err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return err
	}

	// Check for successful merge
	var result MergeRequest
	if err := readResponse(resp, &result); err != nil {
		return err
	}

	if !result.Merged {
		return fmt.Errorf("merge request was not merged")
	}

	return nil
}

// CreateVariable creates a CI variable
func (c *GitLabClient) CreateVariable(ctx context.Context, projectID int, key, value, envScope string) (*Variable, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id":    projectID,
		"key":           key,
		"env_scope":     envScope,
	}).Debug("Creating variable")

	body := map[string]interface{}{
		"key":            key,
		"value":          value,
		"environment_scope": envScope,
	}

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf("projects/%d/variables", projectID), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var variable Variable
	if err := readResponse(resp, &variable); err != nil {
		return nil, err
	}

	return &variable, nil
}

// ListVariables lists CI variables
func (c *GitLabClient) ListVariables(ctx context.Context, projectID int) ([]Variable, error) {
	c.logger.WithField("project_id", projectID).Debug("Listing variables")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/variables", projectID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var variables []Variable
	if err := readResponse(resp, &variables); err != nil {
		return nil, err
	}

	return variables, nil
}

// GetVariable fetches a specific variable
func (c *GitLabClient) GetVariable(ctx context.Context, projectID int, key string) (*Variable, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"key":        key,
	}).Debug("Fetching variable")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/variables/%s", projectID, url.PathEscape(key)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var variable Variable
	if err := readResponse(resp, &variable); err != nil {
		return nil, err
	}

	return &variable, nil
}

// UpdateVariable updates a CI variable
func (c *GitLabClient) UpdateVariable(ctx context.Context, projectID int, key, value, envScope string) (*Variable, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"key":        key,
		"env_scope":  envScope,
	}).Debug("Updating variable")

	body := map[string]interface{}{
		"value": value,
	}

	if envScope != "" {
		body["environment_scope"] = envScope
	}

	req, err := c.buildRequest(ctx, http.MethodPut, fmt.Sprintf("projects/%d/variables/%s", projectID, url.PathEscape(key)), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var variable Variable
	if err := readResponse(resp, &variable); err != nil {
		return nil, err
	}

	return &variable, nil
}

// DeleteVariable deletes a CI variable
func (c *GitLabClient) DeleteVariable(ctx context.Context, projectID int, key string) error {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"key":        key,
	}).Debug("Deleting variable")

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf("projects/%d/variables/%s", projectID, url.PathEscape(key)), nil)
	if err != nil {
		return err
	}

	_, err = c.executeRequest(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// GetEnvironments lists environments
func (c *GitLabClient) GetEnvironments(ctx context.Context, projectID int) ([]Environment, error) {
	c.logger.WithField("project_id", projectID).Debug("Fetching environments")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/environments", projectID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var environments []Environment
	if err := readResponse(resp, &environments); err != nil {
		return nil, err
	}

	return environments, nil
}

// GetPipelineArtifacts fetches pipeline artifacts
func (c *GitLabClient) GetPipelineArtifacts(ctx context.Context, projectID int, pipelineID int64) ([]Artifact, error) {
	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"pipeline_id": pipelineID,
	}).Debug("Fetching pipeline artifacts")

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf("projects/%d/pipelines/%d/artifacts", projectID, pipelineID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.executeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var artifacts []Artifact
	if err := readResponse(resp, &artifacts); err != nil {
		return nil, err
	}

	return artifacts, nil
}

// GetHealthStatus returns health status of GitLab client
func (c *GitLabClient) GetHealthStatus(ctx context.Context) map[string]interface{} {
	url := fmt.Sprintf("%s/api/v4/application", c.apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  err.Error(),
		}
	}

	req.Header.Set("PRIVATE-TOKEN", c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  err.Error(),
		}
	}
	defer resp.Body.Close()

	return map[string]interface{}{
		"status":    "healthy",
		"timeout":   c.config.Timeout.String(),
		"api_url":   c.apiURL,
		"has_auth":  c.apiToken != "",
		"version":   resp.Header.Get("X-GitLab-Version"),
	}
}

// CloseConnection closes the HTTP client connection
func (c *GitLabClient) CloseConnection() {
	c.httpClient.CloseIdleConnections()
}

// GetMRStatus gets the status of a merge request
func (c *GitLabClient) GetMRStatus(ctx context.Context, projectID int, MRIID int) (*MergeRequest, error) {
	return c.GetMergeRequest(ctx, projectID, MRIID)
}

// GetMRJobs gets all jobs related to a merge request
func (c *GitLabClient) GetMRJobs(ctx context.Context, projectID int, MRIID int) ([]Job, error) {
	// Get pipelines filtered by source branch
	pipelines, err := c.ListPipelines(ctx, projectID, &ListPipelinesOptions{
		Status:   "success",
		Ref:      "refs/merge-requests/" + strconv.Itoa(MRIID) + "/merge",
	})
	if err != nil {
		return nil, err
	}

	// Return jobs from most recent pipeline
	if len(pipelines) > 0 {
		return c.GetPipelineJobs(ctx, int(pipelines[0].ID), pipelines[0].ID)
	}

	return []Job{}, nil
}

// GetProjectStatus gets the overall status of a project
func (c *GitLabClient) GetProjectStatus(ctx context.Context, projectID int) (map[string]interface{}, error) {
	project, err := c.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	pipelines, err := c.ListPipelines(ctx, projectID, &ListPipelinesOptions{
		PerPage: 1,
	})
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"project_id":      project.ID,
		"name":            project.Name,
		"visibility":      project.Visibility,
		"archived":        project.Archived,
		"default_branch":  project.DefaultBranch,
		"last_activity":   project.LastActivityAt,
		"latest_pipeline": nil,
	}

	if len(pipelines) > 0 {
		status["latest_pipeline"] = map[string]interface{}{
			"id":       pipelines[0].ID,
			"status":   pipelines[0].Status,
			"ref":      pipelines[0].Ref,
			"sha":      pipelines[0].Sha,
			"duration": pipelines[0].Duration,
		}
	}

	return status, nil
}
