package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OrchestrationController coordinates GitLab CI with Axiom's CI orchestration system
type OrchestrationController struct {
	logger   *logrus.Logger
	client   *GitLabClient
	handler  *WebhookHandler
	config   *OrchestrationConfig
	metrics  *OrchestrationMetrics
	mu       sync.RWMutex
	running  bool
	cancel   context.CancelFunc
}

// OrchestrationConfig contains orchestration controller configuration
type OrchestrationConfig struct {
	EnableMetrics         bool
	MetricsCollectionInterval time.Duration
	MaxConcurrentPipelines int
	EnableCostTracking    bool
	CostPerMinute         float64
	EnableServiceDiscovery bool
	AutoRegisterRunners   bool
	Runners               []RunnerConfig
}

// RunnerConfig contains runner configuration for auto-registration
type RunnerConfig struct {
	Token         string
	Description   string
	Tags          string
	Shared        bool
	Timeout       time.Duration
	Enabled       bool
}

// PipelineExecution tracks a pipeline execution
type PipelineExecution struct {
	ID             int64
	PipelineID     int64
	ProjectID      int
	ProjectName    string
	Status         string
	Ref            string
	Sha            string
	Source         string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	Duration       float64
	Stages         []string
	Jobs           []JobExecution
	Cost           float64
	Metrics        map[string]interface{}
}

// JobExecution tracks a job execution
type JobExecution struct {
	ID          int64
	Name        string
	Status      string
	Stage       string
	Duration    float64
	StartedAt   *time.Time
	FinishedAt  *time.Time
	AllowFailure bool
	Artifacts   []string
}

// OrchestrationMetrics tracks orchestration metrics
type OrchestrationMetrics struct {
	mu              sync.RWMutex
	TotalPipelines  int64
	SuccessfulPipelines  int64
	FailedPipelines   int64
	PendingPipelines int64
	TotalDuration   time.Duration
	LastPipelineAt  time.Time
	PipelinesByDay  map[string]int64
	PipelinesByStatus map[string]int64
	TotalCost       float64
}

// ServiceInfo contains service information for discovery
type ServiceInfo struct {
	ID           string
	Name         string
	Type         string
	Source       string
	ProjectID    int
	ProjectName  string
	Namespace    string
	ExternalURL  string
	Environment  string
	Tags         []string
	Labels       map[string]string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]interface{}
}

// OrchestrationController is the main coordinator for GitLab CI orchestration
func NewOrchestrationController(logger *logrus.Logger, client *GitLabClient, handler *WebhookHandler, config OrchestrationConfig) *OrchestrationController {
	if config.MetricsCollectionInterval == 0 {
		config.MetricsCollectionInterval = 5 * time.Minute
	}

	if config.MaxConcurrentPipelines == 0 {
		config.MaxConcurrentPipelines = 10
	}

	if config.EnableCostTracking {
		if config.CostPerMinute == 0 {
			config.CostPerMinute = 0.002 // Default $0.002 per minute
		}
	}

	return &OrchestrationController{
		logger:   logger,
		client:   client,
		handler:  handler,
		config:   &config,
		metrics: &OrchestrationMetrics{
			PipelinesByDay:      make(map[string]int64),
			PipelinesByStatus:   make(map[string]int64),
		},
	}
}

// Start starts the orchestration controller
func (c *OrchestrationController) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		c.logger.Warn("Orchestration controller already running")
		return nil
	}
	c.running = true
	c.mu.Unlock()

	c.logger.Info("Starting GitLab orchestration controller")

	// Set up cancellation
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Register webhook handlers with orchestrator integration
	c.registerHandlers()

	// Start metrics collection if enabled
	if c.config.EnableMetrics {
		go c.startMetricsCollection(ctx)
	}

	// Start service discovery if enabled
	if c.config.EnableServiceDiscovery {
		go c.startServiceDiscovery(ctx)
	}

	// Auto-register runners if configured
	if c.config.AutoRegisterRunners {
		go c.startRunnerRegistration(ctx)
	}

	c.logger.Info("GitLab orchestration controller started")
	return nil
}

// Stop stops the orchestration controller
func (c *OrchestrationController) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	c.logger.Info("Stopping GitLab orchestration controller")
	return nil
}

// registerHandlers sets up webhook handlers for orchestration integration
func (c *OrchestrationController) registerHandlers() {
	// Register push event handler for CI triggers
	c.handler.RegisterHandler("push", c.handlePushEvent)

	// Register pipeline event handler for status tracking
	c.handler.RegisterHandler("pipeline", c.handlePipelineEvent)

	// Register job event handler for job monitoring
	c.handler.RegisterHandler("job", c.handleJobEvent)

	// Register merge request event handler for MR integration
	c.handler.RegisterHandler("merge_request", c.handleMergeRequestEvent)

	c.logger.Info("Webhook handlers registered")
}

// startServiceDiscovery starts service discovery loop
func (c *OrchestrationController) startServiceDiscovery(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.discoverServices()
		}
	}
}

// discoverServices discovers services from GitLab projects
func (c *OrchestrationController) discoverServices() {
	// In production, would query and register services
	c.logger.Debug("Running service discovery")
}

// handlePushEvent handles push events for CI triggers
func (c *OrchestrationController) handlePushEvent(event *WebhookEvent) {
	if event.Project.ID == 0 {
		c.logger.Warn("Project not set in push event")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"project": event.Project.FullName,
		"ref":     event.Ref,
		"commits": len(event.Commits),
	}).Info("Processing push event for CI trigger")

	// Trigger CI pipeline
	pipeline, err := c.client.CreatePipeline(context.Background(), event.Project.ID, event.Ref, "Triggered by Axiom IDP")
	if err != nil {
		c.logger.WithError(err).Error("Failed to trigger pipeline")
		return
	}

	// Track pipeline execution
	c.trackPipelineExecution(&PipelineExecution{
		PipelineID: pipeline.ID,
		ProjectID:  event.Project.ID,
		ProjectName: event.Project.FullName,
		Status:     pipeline.Status,
		Ref:        pipeline.Ref,
		Sha:        pipeline.Sha,
		Source:     pipeline.Source,
		CreatedAt:  time.Now(),
	})

	// Update service catalog if service discovery enabled
	if c.config.EnableServiceDiscovery {
		c.discoverService(event, pipeline)
	}
}

// handlePipelineEvent handles pipeline status events
func (c *OrchestrationController) handlePipelineEvent(event *WebhookEvent) {
	if event.Pipeline == nil {
		c.logger.Warn("Pipeline not set in event")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"pipeline_id": event.Pipeline.ID,
		"status":      event.Pipeline.Status,
		"duration":    event.Pipeline.Duration,
	}).Info("Processing pipeline event")

	// Update pipeline execution status
	c.updatePipelineStatus(&PipelineExecution{
		PipelineID: event.Pipeline.ID,
		ProjectID:  int(event.Pipeline.ProjectID),
		Status:     event.Pipeline.Status,
		Ref:        event.Pipeline.Ref,
		Sha:        event.Pipeline.Sha,
		Duration:   event.Pipeline.Duration,
	})

	// Calculate cost if tracking enabled
	if c.config.EnableCostTracking {
		c.calculateCost(event.Pipeline)
	}

	// Get job details
	jobs, err := c.client.GetPipelineJobs(context.Background(), event.Project.ID, event.Pipeline.ID)
	if err == nil {
		c.updatePipelineJobs(jobs)
	}
}

// handleJobEvent handles job events
func (c *OrchestrationController) handleJobEvent(event *WebhookEvent) {
	if event.Job == nil {
		c.logger.Warn("Job not set in event")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"job_id": event.Job.ID,
		"status": event.Job.Status,
		"stage":  event.Job.Stage,
	}).Info("Processing job event")

	// Track job execution
	// This would connect to the CI orchestration system
}

// handleMergeRequestEvent handles merge request events
func (c *OrchestrationController) handleMergeRequestEvent(event *WebhookEvent) {
	if event.MergeRequest == nil {
		c.logger.Warn("Merge request not set in event")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"mr_iid":  event.MergeRequest.IID,
		"state":   event.MergeRequest.State,
		"source":  event.MergeRequest.SourceBranch,
		"target":  event.MergeRequest.TargetBranch,
	}).Info("Processing merge request event")

	// Check MR status
	if event.MergeRequest.State == "merged" {
		// Update service catalog for merged deployments
		c.updateMergedService(event.MergeRequest)
	}
}

// trackPipelineExecution tracks a pipeline execution
func (c *OrchestrationController) trackPipelineExecution(exec *PipelineExecution) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalPipelines++
	c.metrics.LastPipelineAt = time.Now()
	c.metrics.PipelinesByDay[exec.CreatedAt.Format("2006-01-02")]++
	c.metrics.PipelinesByStatus[exec.Status]++

	// Store execution info for tracking
	c.logger.WithFields(logrus.Fields{
		"pipeline_id": exec.PipelineID,
		"project":     exec.ProjectName,
		"status":      exec.Status,
	}).Debug("Tracked pipeline execution")
}

// updatePipelineStatus updates pipeline status tracking
func (c *OrchestrationController) updatePipelineStatus(exec *PipelineExecution) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update duration tracking
	if exec.Duration > 0 {
		c.metrics.TotalDuration += time.Duration(exec.Duration) * time.Second
	}

	// Update status counts
	status := exec.Status
	if status == "success" {
		c.metrics.SuccessfulPipelines++
	} else if status == "failed" {
		c.metrics.FailedPipelines++
	}

	// Update cost tracking
	if c.config.EnableCostTracking {
		c.metrics.TotalCost += exec.Cost
	}

	c.logger.WithFields(logrus.Fields{
		"pipeline_id": exec.PipelineID,
		"status":      status,
		"cost":        exec.Cost,
	}).Debug("Updated pipeline status")
}

// updatePipelineJobs updates job information for a pipeline
func (c *OrchestrationController) updatePipelineJobs(jobs []Job) {
	// Collect job information
	for _, job := range jobs {
		// Update job status tracking
		c.logger.WithFields(logrus.Fields{
			"job_id": job.ID,
			"status": job.Status,
			"stage":  job.Stage,
		}).Debug("Pipeline job updated")
	}
}

// calculateCost calculates the cost of a pipeline
func (c *OrchestrationController) calculateCost(pipeline *Pipeline) {
	if !c.config.EnableCostTracking {
		return
	}

	// Calculate cost based on duration
	durationMinutes := pipeline.Duration / 60.0
	cost := durationMinutes * c.config.CostPerMinute

	c.logger.WithFields(logrus.Fields{
		"pipeline_id": pipeline.ID,
		"duration":    pipeline.Duration,
		"minutes":     durationMinutes,
		"cost":        cost,
	}).Debug("Calculated pipeline cost")
}

// discoverService discovers a service from GitLab project
func (c *OrchestrationController) discoverService(event *WebhookEvent, pipeline *Pipeline) {
	service := &ServiceInfo{
		ID:          fmt.Sprintf("gitlab-%d", event.Project.ID),
		Name:        event.Project.Name,
		Type:        "gitlab-project",
		Source:      "gitlab",
		ProjectID:   event.Project.ID,
		ProjectName: event.Project.FullName,
		Namespace:   event.Project.PathWithNamespace,
		ExternalURL: event.Project.WebURL,
		Environment: "ci",
		Tags:        []string{"gitlab", "ci", "project"},
		CreatedAt:   event.Project.CreatedAt,
		UpdatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"pipeline_id":      pipeline.ID,
			"pipeline_status":  pipeline.Status,
			"default_branch":   event.Project.DefaultBranch,
			"visibility":       event.Project.Visibility,
		},
	}

	// This would register with the service discovery system
	c.logger.WithField("service_id", service.ID).Info("Discovered GitLab service")
}

// updateMergedService updates service info for merged merge requests
func (c *OrchestrationController) updateMergedService(mr *MergeRequest) {
	c.logger.WithField("mr_iid", mr.IID).Info("Updating service for merged MR")
	// This would update service catalog with deployment info from MR
}

// startMetricsCollection starts metrics collection
func (c *OrchestrationController) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(c.config.MetricsCollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectMetrics()
		}
	}
}

// collectMetrics collects and reports orchestration metrics
func (c *OrchestrationController) collectMetrics() {
	c.mu.RLock()
	metrics := *c.metrics
	c.mu.RUnlock()

	c.logger.WithFields(logrus.Fields{
		"total_pipelines": metrics.TotalPipelines,
		"successful":      metrics.SuccessfulPipelines,
		"failed":          metrics.FailedPipelines,
		"total_cost":      metrics.TotalCost,
	}).Info("Collected orchestration metrics")
}

// startRunnerRegistration starts runner registration loop
func (c *OrchestrationController) startRunnerRegistration(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.registerRunners()
		}
	}
}

// registerRunners registers configured runners
func (c *OrchestrationController) registerRunners() {
	for _, runnerConfig := range c.config.Runners {
		if !runnerConfig.Enabled {
			continue
		}

		runner, err := c.client.RegisterRunner(
			context.Background(),
			runnerConfig.Token,
			runnerConfig.Description,
			runnerConfig.Tags,
		)
		if err != nil {
			c.logger.WithError(err).Error("Failed to register runner")
			continue
		}

		c.logger.WithFields(logrus.Fields{
			"runner_id":   runner.ID,
			"description": runner.Description,
		}).Info("Registered GitLab runner")
	}
}

// GetMetrics returns current orchestration metrics
func (c *OrchestrationController) GetMetrics() *OrchestrationMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy
	metrics := *c.metrics
	return &metrics
}

// GetExecution returns pipeline execution info
func (c *OrchestrationController) GetExecution(pipelineID int64) (*PipelineExecution, error) {
	// In production, would look up from execution store
	return &PipelineExecution{
		PipelineID: pipelineID,
	}, nil
}

// SetMetricsCollectionInterval sets the metrics collection interval
func (c *OrchestrationController) SetMetricsCollectionInterval(interval time.Duration) {
	c.config.MetricsCollectionInterval = interval
}

// SetEnableCostTracking enables or disables cost tracking
func (c *OrchestrationController) SetEnableCostTracking(enable bool, costPerMinute float64) {
	c.config.EnableCostTracking = enable
	c.config.CostPerMinute = costPerMinute
}

// SetEnableServiceDiscovery enables or disables service discovery
func (c *OrchestrationController) SetEnableServiceDiscovery(enable bool) {
	c.config.EnableServiceDiscovery = enable
}

// GetClient returns the GitLab client
func (c *OrchestrationController) GetClient() *GitLabClient {
	return c.client
}

// GetHandler returns the webhook handler
func (c *OrchestrationController) GetHandler() *WebhookHandler {
	return c.handler
}

// IsRunning returns whether the controller is running
func (c *OrchestrationController) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetTotalCost returns the total tracked cost
func (c *OrchestrationController) GetTotalCost() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics.TotalCost
}

// GetSuccessRate returns the success rate of pipelines
func (c *OrchestrationController) GetSuccessRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.metrics.TotalPipelines == 0 {
		return 0
	}

	return float64(c.metrics.SuccessfulPipelines) / float64(c.metrics.TotalPipelines) * 100
}

// GetAverageDuration returns the average pipeline duration
func (c *OrchestrationController) GetAverageDuration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.metrics.TotalPipelines == 0 {
		return 0
	}

	return c.metrics.TotalDuration / time.Duration(c.metrics.TotalPipelines)
}

// MarshalJSON implements custom JSON marshaling for metrics
func (m *OrchestrationMetrics) MarshalJSON() ([]byte, error) {
	type Alias struct {
		TotalPipelines        int64            `json:"total_pipelines"`
		SuccessfulPipelines   int64            `json:"successful_pipelines"`
		FailedPipelines       int64            `json:"failed_pipelines"`
		PendingPipelines      int64            `json:"pending_pipelines"`
		TotalDuration         float64          `json:"total_duration_seconds"`
		LastPipelineAt        string           `json:"last_pipeline_at"`
		PipelinesByDay        map[string]int64 `json:"pipelines_by_day"`
		PipelinesByStatus     map[string]int64 `json:"pipelines_by_status"`
		TotalCost             float64          `json:"total_cost"`
	}

	return json.Marshal(&Alias{
		TotalPipelines:    m.TotalPipelines,
		SuccessfulPipelines: m.SuccessfulPipelines,
		FailedPipelines:   m.FailedPipelines,
		PendingPipelines:  m.PendingPipelines,
		TotalDuration:     m.TotalDuration.Seconds(),
		LastPipelineAt:    m.LastPipelineAt.Format(time.RFC3339),
		PipelinesByDay:    m.PipelinesByDay,
		PipelinesByStatus: m.PipelinesByStatus,
		TotalCost:         m.TotalCost,
	})
}

// ServiceDiscovery returns service information from GitLab
func (c *OrchestrationController) ServiceDiscovery() ([]ServiceInfo, error) {
	if !c.config.EnableServiceDiscovery {
		return []ServiceInfo{}, nil
	}

	// In production, would query the service catalog
	// This is a placeholder for integration
	return []ServiceInfo{}, nil
}
