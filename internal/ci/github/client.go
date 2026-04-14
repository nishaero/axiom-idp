package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v56/github"
	"github.com/sirupsen/logrus"
)

// GitHubClient represents a GitHub API client
type GitHubClient struct {
	client        *github.Client
	baseURL       string
	webhookSecret string
	config        ClientConfig
	logger        *logrus.Logger
	httpClient    *http.Client
}

// ClientConfig contains GitHub client configuration
type ClientConfig struct {
	BaseURL         string
	WebhookSecret   string
	Token           string
	Timeout         time.Duration
	RateLimit       int
	RateLimitWindow time.Duration
	EnableMetrics   bool
}

// PREvent represents a GitHub pull request event
type PREvent struct {
	Action         string     `json:"action"`
	Number         int        `json:"number,omitempty"`
	State          string     `json:"state,omitempty"`
	Title          string     `json:"title,omitempty"`
	Body           string     `json:"body,omitempty"`
	Repository     Repository `json:"repository"`
	Organization   string     `json:"organization"`
	Sender         User       `json:"sender"`
	HookID         int64      `json:"hook_id"`
	InstallationID int64      `json:"installation_id"`
	Head           struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		User User   `json:"user"`
	} `json:"head,omitempty"`
	Base struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		User User   `json:"user"`
	} `json:"base,omitempty"`
	HeadCommit *Commit     `json:"head_commit,omitempty"`
	PR         PullRequest `json:"pull_request,omitempty"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID            int64      `json:"id"`
	Number        int        `json:"number"`
	State         string     `json:"state"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	User          User       `json:"user"`
	Head          Ref        `json:"head"`
	Base          Ref        `json:"base"`
	Links         Links      `json:"links"`
	HeadCommit    *Commit    `json:"head_commit,omitempty"`
	Merged        bool       `json:"merged"`
	MergedAt      *time.Time `json:"merged_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	AddedFiles    []string   `json:"added_files,omitempty"`
	ModifiedFiles []string   `json:"modified_files,omitempty"`
	DeletedFiles  []string   `json:"deleted_files,omitempty"`
}

// Ref represents a git reference
type Ref struct {
	Label string     `json:"label"`
	Ref   string     `json:"ref"`
	Sha   string     `json:"sha"`
	Repo  Repository `json:"repo"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description,omitempty"`
	Owner         User      `json:"owner"`
	HTMLURL       string    `json:"html_url"`
	Language      string    `json:"language"`
	Visibility    string    `json:"visibility,omitempty"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	Private       bool      `json:"private"`
	Archived      bool      `json:"archived"`
	Disabled      bool      `json:"disabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	PushedAt      time.Time `json:"pushed_at,omitempty"`
	CloneURL      string    `json:"clone_url"`
	GitURL        string    `json:"git_url"`
	SSHURL        string    `json:"ssh_url,omitempty"`
	SURL          string    `json:"surl,omitempty"`
}

// User represents a GitHub user
type User struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
}

// Commit represents a git commit
type Commit struct {
	Sha       string       `json:"sha"`
	Message   string       `json:"message"`
	Author    CommitAuthor `json:"author"`
	Committer CommitAuthor `json:"committer"`
	HTMLURL   string       `json:"html_url"`
	TreeSha   string       `json:"tree_sha"`
}

// CommitAuthor represents commit author information
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// Links represents PR links
type Links struct {
	Self     Link `json:"self"`
	HTML     Link `json:"html"`
	Comments Link `json:"comments"`
}

// Link represents a URL link
type Link struct {
	HRef string `json:"href"`
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID               int64         `json:"id"`
	Name             string        `json:"name"`
	NodeID           string        `json:"node_id"`
	HeadBranch       string        `json:"head_branch"`
	HeadSha          string        `json:"head_sha"`
	RunNumber        int           `json:"run_number"`
	Event            string        `json:"event"`
	Status           string        `json:"status"`
	Conclusion       string        `json:"conclusion"`
	WorkflowID       int64         `json:"workflow_id"`
	CheckSuiteID     int64         `json:"check_suite_id"`
	CheckSuiteNodeID string        `json:"check_suite_node_id"`
	URL              string        `json:"url"`
	PullRequests     []PullRequest `json:"pull_requests"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Actor            User          `json:"actor"`
	AttemptNumber    int           `json:"attempt_number"`
	RunStartedAt     time.Time     `json:"run_started_at"`
	JobsURL          string        `json:"jobs_url"`
	LogsURL          string        `json:"logs_url"`
	CheckSuiteURL    string        `json:"check_suite_url"`
	HeadCommit       *Commit       `json:"head_commit,omitempty"`
}

// CheckRun represents a check run from GitHub Actions
type CheckRun struct {
	ID           int64          `json:"id"`
	HeadSHA      string         `json:"head_sha"`
	Status       string         `json:"status"`
	Conclusion   string         `json:"conclusion"`
	Name         string         `json:"name"`
	URL          string         `json:"url"`
	HTMLURL      string         `json:"html_url"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  time.Time      `json:"completed_at"`
	Output       CheckRunOutput `json:"output"`
	CheckSuite   CheckSuite     `json:"check_suite"`
	Repository   Repository     `json:"repository"`
	PullRequests []PullRequest  `json:"pull_requests"`
}

// CheckRunOutput represents check run output
type CheckRunOutput struct {
	Title            string `json:"title"`
	Summary          string `json:"summary"`
	Text             string `json:"text"`
	AnnotationsCount int    `json:"annotations_count"`
	AnnotationsURL   string `json:"annotations_url"`
}

// CheckSuite represents a check suite
type CheckSuite struct {
	ID         int64      `json:"id"`
	HeadBranch string     `json:"head_branch"`
	HeadSHA    string     `json:"head_sha"`
	Status     string     `json:"status"`
	Conclusion string     `json:"conclusion"`
	URL        string     `json:"url"`
	Repository Repository `json:"repository"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Deployment represents a GitHub deployment
type Deployment struct {
	ID             int64           `json:"id"`
	RepositoryID   int64           `json:"repository_id"`
	sha            string          `json:"sha"`
	Ref            string          `json:"ref"`
	Task           string          `json:"task"`
	Payload        json.RawMessage `json:"payload"`
	PromotionLevel string          `json:"promotion_level,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	URL            string          `json:"url"`
	HTMLURL        string          `json:"html_url"`
	Status         string          `json:"status"`
}

// DeploymentStatus represents a deployment status
type DeploymentStatus struct {
	ID          int64     `json:"id"`
	State       string    `json:"state"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
	TargetURL   string    `json:"target_url,omitempty"`
	Description string    `json:"description"`
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(logger *logrus.Logger, config ClientConfig) *GitHubClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Create GitHub API client
	var gitHubClient *github.Client
	if config.Token != "" {
		gitHubClient = github.NewClient(client).WithAuthToken(config.Token)
	} else {
		gitHubClient = github.NewClient(client)
	}

	if config.BaseURL != "" {
		if baseURL, err := url.Parse(config.BaseURL); err == nil {
			if !strings.HasSuffix(baseURL.Path, "/") {
				baseURL.Path += "/"
			}
			gitHubClient.BaseURL = baseURL
			gitHubClient.UploadURL = baseURL
		}
	}

	return &GitHubClient{
		client:        gitHubClient,
		baseURL:       config.BaseURL,
		webhookSecret: config.WebhookSecret,
		config:        config,
		logger:        logger,
		httpClient:    client,
	}
}

// Validate validates the client configuration
func (c *GitHubClient) Validate() error {
	if c.config.Timeout < 1*time.Second {
		return fmt.Errorf("invalid timeout: must be at least 1 second")
	}

	return nil
}

// GetRepository fetches repository information
func (c *GitHubClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	c.logger.WithFields(logrus.Fields{
		"owner": owner,
		"repo":  repo,
	}).Debug("Fetching repository")

	gitHubRepo, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return gitHubRepoToModel(gitHubRepo), nil
}

// GetPullRequest fetches pull request details
func (c *GitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	c.logger.WithFields(logrus.Fields{
		"owner":  owner,
		"repo":   repo,
		"number": number,
	}).Debug("Fetching pull request")

	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	return prToModel(pr), nil
}

// ListPullRequests lists pull requests for a repository
func (c *GitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]PullRequest, error) {
	c.logger.WithFields(logrus.Fields{
		"owner": owner,
		"repo":  repo,
	}).Debug("Listing pull requests")

	if opts == nil {
		opts = &github.PullRequestListOptions{}
	}

	var allPRs []PullRequest
	page := 0

	for {
		page++
		opts.Page = page
		prs, resp, err := c.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		for _, pr := range prs {
			allPRs = append(allPRs, *prToModel(pr))
		}

		if resp.NextPage == 0 || len(prs) == 0 {
			break
		}
	}

	return allPRs, nil
}

// GetWorkflowRuns lists workflow runs for a repository
func (c *GitHubClient) GetWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64, opts *github.ListWorkflowRunsOptions) ([]WorkflowRun, error) {
	c.logger.WithFields(logrus.Fields{
		"owner":       owner,
		"repo":        repo,
		"workflow_id": workflowID,
	}).Debug("Fetching workflow runs")

	if opts == nil {
		opts = &github.ListWorkflowRunsOptions{}
	}

	runs, _, err := c.client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	var result []WorkflowRun
	for _, run := range runs.WorkflowRuns {
		result = append(result, *workflowRunToModel(run))
	}

	return result, nil
}

// GetWorkflowRun fetches a single workflow run by ID.
func (c *GitHubClient) GetWorkflowRun(ctx context.Context, owner, repo string, workflowRunID int64) (*WorkflowRun, error) {
	run, _, err := c.client.Actions.GetWorkflowRunByID(ctx, owner, repo, workflowRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	return workflowRunToModel(run), nil
}

// GetCheckRuns lists check runs for a workflow run
func (c *GitHubClient) GetCheckRuns(ctx context.Context, owner, repo string, workflowRunID int64) ([]CheckRun, error) {
	c.logger.WithFields(logrus.Fields{
		"owner":           owner,
		"repo":            repo,
		"workflow_run_id": workflowRunID,
	}).Debug("Fetching check runs")

	run, err := c.GetWorkflowRun(ctx, owner, repo, workflowRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to list check runs: %w", err)
	}

	checkRuns, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, run.HeadSha, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list check runs: %w", err)
	}

	var result []CheckRun
	for _, checkRun := range checkRuns.CheckRuns {
		result = append(result, *checkRunToModel(checkRun))
	}

	return result, nil
}

// DispatchWorkflowRun triggers a workflow run
func (c *GitHubClient) DispatchWorkflowRun(ctx context.Context, owner, repo string, workflowID int64, inputs map[string]string) (*WorkflowRun, error) {
	c.logger.WithFields(logrus.Fields{
		"owner":       owner,
		"repo":        repo,
		"workflow_id": workflowID,
		"inputs":      inputs,
	}).Debug("Dispatching workflow run")

	request := github.CreateWorkflowDispatchEventRequest{
		Ref:    "main",
		Inputs: make(map[string]interface{}, len(inputs)),
	}
	for k, v := range inputs {
		request.Inputs[k] = v
	}

	_, err := c.client.Actions.CreateWorkflowDispatchEventByID(ctx, owner, repo, workflowID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to dispatch workflow run: %w", err)
	}

	return &WorkflowRun{
		ID:         workflowID,
		Name:       "workflow-dispatch",
		HeadBranch: request.Ref,
		Event:      "workflow_dispatch",
		Status:     "queued",
		Conclusion: "",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// CreateDeploymentStatus updates deployment status
func (c *GitHubClient) CreateDeploymentStatus(ctx context.Context, owner, repo string, deploymentID int64, state string, targetURL string, description string) (*DeploymentStatus, error) {
	c.logger.WithFields(logrus.Fields{
		"owner":      owner,
		"repo":       repo,
		"deployment": deploymentID,
		"state":      state,
	}).Debug("Creating deployment status")

	statusRequest := github.DeploymentStatusRequest{
		State:       github.String(state),
		LogURL:      github.String(targetURL),
		Description: github.String(description),
	}

	status, _, err := c.client.Repositories.CreateDeploymentStatus(ctx, owner, repo, deploymentID, &statusRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment status: %w", err)
	}

	return deploymentStatusToModel(status), nil
}

// VerifyWebhookSignature verifies a webhook signature
func (c *GitHubClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	if c.webhookSecret == "" {
		return true
	}

	signature = "sha256=" + signature
	expected := c.hmacSHA256(payload, c.webhookSecret)

	return signature == expected
}

// hmacSHA256 computes HMAC-SHA256 signature
func (c *GitHubClient) hmacSHA256(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(data)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// GetHealthStatus returns health status of GitHub client
func (c *GitHubClient) GetHealthStatus(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"status":      "healthy",
		"timeout":     c.config.Timeout.String(),
		"rate_limit":  c.config.RateLimit,
		"has_token":   c.config.Token != "",
		"has_webhook": c.webhookSecret != "",
		"base_url":    c.baseURL,
	}
}

// Convert helper functions

func gitHubRepoToModel(repo *github.Repository) *Repository {
	return &Repository{
		ID:            repo.GetID(),
		Name:          repo.GetName(),
		FullName:      repo.GetFullName(),
		Description:   repo.GetDescription(),
		HTMLURL:       repo.GetHTMLURL(),
		Language:      repo.GetLanguage(),
		Visibility:    repo.GetVisibility(),
		DefaultBranch: repo.GetDefaultBranch(),
		Private:       repo.GetPrivate(),
		Archived:      repo.GetArchived(),
		Disabled:      repo.GetDisabled(),
		CreatedAt:     repo.GetCreatedAt().Time,
		UpdatedAt:     repo.GetUpdatedAt().Time,
		PushedAt:      repo.GetPushedAt().Time,
		CloneURL:      repo.GetCloneURL(),
		GitURL:        repo.GetGitURL(),
		SSHURL:        repo.GetSSHURL(),
		SURL:          repo.GetCloneURL(),
		Owner: User{
			Login:     repo.GetOwner().GetLogin(),
			ID:        repo.GetOwner().GetID(),
			AvatarURL: repo.GetOwner().GetAvatarURL(),
			HTMLURL:   repo.GetOwner().GetHTMLURL(),
			Type:      repo.GetOwner().GetType(),
		},
	}
}

func prToModel(pr *github.PullRequest) *PullRequest {
	var mergedAt *time.Time
	mergedAtValue := pr.GetMergedAt()
	if !mergedAtValue.Time.IsZero() {
		t := mergedAtValue.Time
		mergedAt = &t
	}

	return &PullRequest{
		ID:        pr.GetID(),
		Number:    pr.GetNumber(),
		State:     pr.GetState(),
		Title:     pr.GetTitle(),
		Body:      pr.GetBody(),
		CreatedAt: pr.GetCreatedAt().Time,
		UpdatedAt: pr.GetUpdatedAt().Time,
		Merged:    pr.GetMerged(),
		MergedAt:  mergedAt,
		Head: Ref{
			Label: pr.GetHead().GetLabel(),
			Ref:   pr.GetHead().GetRef(),
			Sha:   pr.GetHead().GetSHA(),
			Repo:  *gitHubRepoToModel(pr.GetHead().GetRepo()),
		},
		Base: Ref{
			Label: pr.GetBase().GetLabel(),
			Ref:   pr.GetBase().GetRef(),
			Sha:   pr.GetBase().GetSHA(),
			Repo:  *gitHubRepoToModel(pr.GetBase().GetRepo()),
		},
		User: User{
			Login:     pr.GetUser().GetLogin(),
			ID:        pr.GetUser().GetID(),
			AvatarURL: pr.GetUser().GetAvatarURL(),
			HTMLURL:   pr.GetUser().GetHTMLURL(),
			Type:      pr.GetUser().GetType(),
		},
		Links: Links{
			Self:     Link{HRef: pr.GetLinks().GetSelf().GetHRef()},
			HTML:     Link{HRef: pr.GetHTMLURL()},
			Comments: Link{HRef: pr.GetLinks().GetComments().GetHRef()},
		},
	}
}

func workflowRunToModel(run *github.WorkflowRun) *WorkflowRun {
	headCommit := headCommitToModel(run.GetHeadCommit())
	prs := make([]PullRequest, 0, len(run.PullRequests))
	for _, pr := range run.PullRequests {
		if pr != nil {
			prs = append(prs, *prToModel(pr))
		}
	}

	return &WorkflowRun{
		ID:               run.GetID(),
		Name:             run.GetName(),
		NodeID:           run.GetNodeID(),
		HeadBranch:       run.GetHeadBranch(),
		HeadSha:          run.GetHeadSHA(),
		RunNumber:        run.GetRunNumber(),
		Event:            run.GetEvent(),
		Status:           run.GetStatus(),
		Conclusion:       run.GetConclusion(),
		WorkflowID:       run.GetWorkflowID(),
		CheckSuiteID:     run.GetCheckSuiteID(),
		CheckSuiteNodeID: run.GetCheckSuiteNodeID(),
		URL:              run.GetURL(),
		PullRequests:     prs,
		CreatedAt:        run.GetCreatedAt().Time,
		UpdatedAt:        run.GetUpdatedAt().Time,
		Actor: User{
			Login: run.GetActor().GetLogin(),
			ID:    run.GetActor().GetID(),
			Type:  run.GetActor().GetType(),
		},
		AttemptNumber: run.GetRunAttempt(),
		RunStartedAt:  run.GetRunStartedAt().Time,
		JobsURL:       run.GetJobsURL(),
		LogsURL:       run.GetURL() + "/logs",
		CheckSuiteURL: run.GetCheckSuiteURL(),
		HeadCommit:    headCommit,
	}
}

func checkRunToModel(run *github.CheckRun) *CheckRun {
	prs := make([]PullRequest, 0, len(run.PullRequests))
	for _, pr := range run.PullRequests {
		if pr != nil {
			prs = append(prs, *prToModel(pr))
		}
	}

	return &CheckRun{
		ID:          run.GetID(),
		HeadSHA:     run.GetHeadSHA(),
		Status:      run.GetStatus(),
		Conclusion:  run.GetConclusion(),
		Name:        run.GetName(),
		URL:         run.GetURL(),
		HTMLURL:     run.GetHTMLURL(),
		StartedAt:   run.GetStartedAt().Time,
		CompletedAt: run.GetCompletedAt().Time,
		Output: CheckRunOutput{
			Title:            run.GetOutput().GetTitle(),
			Summary:          run.GetOutput().GetSummary(),
			Text:             run.GetOutput().GetText(),
			AnnotationsCount: run.GetOutput().GetAnnotationsCount(),
			AnnotationsURL:   run.GetOutput().GetAnnotationsURL(),
		},
		CheckSuite: CheckSuite{
			ID:         run.GetCheckSuite().GetID(),
			HeadBranch: run.GetCheckSuite().GetHeadBranch(),
			HeadSHA:    run.GetCheckSuite().GetHeadSHA(),
			Status:     run.GetCheckSuite().GetStatus(),
			Conclusion: run.GetCheckSuite().GetConclusion(),
			URL:        run.GetCheckSuite().GetURL(),
			Repository: *gitHubRepoToModel(run.GetCheckSuite().GetRepository()),
		},
		PullRequests: prs,
		Repository:   *gitHubRepoToModel(run.GetCheckSuite().GetRepository()),
	}
}

func deploymentStatusToModel(status *github.DeploymentStatus) *DeploymentStatus {
	return &DeploymentStatus{
		ID:        status.GetID(),
		State:     status.GetState(),
		Name:      status.GetEnvironment(),
		URL:       status.GetURL(),
		CreatedAt: status.GetCreatedAt().Time,
		UpdatedAt: status.GetUpdatedAt().Time,
		Status:    status.GetState(),
		TargetURL: status.GetTargetURL(),
	}
}

func headCommitToModel(commit *github.HeadCommit) *Commit {
	if commit == nil {
		return nil
	}

	var authorDate time.Time
	authorName := ""
	authorEmail := ""
	if author := commit.GetAuthor(); author != nil {
		authorName = author.GetName()
		authorEmail = author.GetEmail()
		authorDate = author.GetDate().Time
	}

	var committerDate time.Time
	committerName := ""
	committerEmail := ""
	if committer := commit.GetCommitter(); committer != nil {
		committerName = committer.GetName()
		committerEmail = committer.GetEmail()
		committerDate = committer.GetDate().Time
	}

	return &Commit{
		Sha:     commit.GetSHA(),
		Message: commit.GetMessage(),
		HTMLURL: commit.GetURL(),
		Author: CommitAuthor{
			Name:  authorName,
			Email: authorEmail,
			Date:  authorDate,
		},
		Committer: CommitAuthor{
			Name:  committerName,
			Email: committerEmail,
			Date:  committerDate,
		},
	}
}

// CloseConnection closes the HTTP client connection
func (c *GitHubClient) CloseConnection() {
	c.httpClient.CloseIdleConnections()
}
