package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

type asyncJobStatus string

const (
	asyncJobStatusQueued    asyncJobStatus = "queued"
	asyncJobStatusRunning   asyncJobStatus = "running"
	asyncJobStatusSucceeded asyncJobStatus = "succeeded"
	asyncJobStatusFailed    asyncJobStatus = "failed"
)

type asyncJobKind string

const (
	asyncJobKindDeploymentApply          asyncJobKind = "deployment_apply"
	asyncJobKindDeploymentApplyArgoCD    asyncJobKind = "deployment_apply_argocd"
	asyncJobKindInfrastructureApply      asyncJobKind = "infrastructure_apply"
	asyncJobKindInfrastructureTerraform  asyncJobKind = "infrastructure_apply_terraform"
	asyncJobKindInfrastructureCrossplane asyncJobKind = "infrastructure_apply_crossplane"
)

var (
	errAsyncJobQueueFull = errors.New("async job queue is full")
	errAsyncJobStopped   = errors.New("async job manager is stopped")
)

type asyncJobResult struct {
	Deployment     *deploymentRecord
	Infrastructure *infrastructureRecord
}

type asyncJobTask func(context.Context) (*asyncJobResult, error)

type asyncJobSubmission struct {
	Kind          asyncJobKind
	Intent        string
	Backend       string
	Source        string
	UserID        string
	Roles         []string
	Summary       string
	Detail        string
	ResourceType  string
	ResourceName  string
	Namespace     string
	Provider      string
	Route         string
	Mode          string
	ExecutionPlan *executionPlan
	Task          asyncJobTask
}

type asyncJob struct {
	ID             string                `json:"id"`
	Kind           string                `json:"kind"`
	Intent         string                `json:"intent,omitempty"`
	Backend        string                `json:"backend,omitempty"`
	Source         string                `json:"source,omitempty"`
	UserID         string                `json:"user_id,omitempty"`
	Roles          []string              `json:"roles,omitempty"`
	Status         string                `json:"status"`
	Summary        string                `json:"summary"`
	Detail         string                `json:"detail,omitempty"`
	ResourceType   string                `json:"resource_type,omitempty"`
	ResourceName   string                `json:"resource_name,omitempty"`
	Namespace      string                `json:"namespace,omitempty"`
	Provider       string                `json:"provider,omitempty"`
	Route          string                `json:"route,omitempty"`
	Mode           string                `json:"mode,omitempty"`
	QueuePosition  int                   `json:"queue_position,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	StartedAt      time.Time             `json:"started_at,omitempty"`
	CompletedAt    time.Time             `json:"completed_at,omitempty"`
	UpdatedAt      time.Time             `json:"updated_at"`
	Error          string                `json:"error,omitempty"`
	ExecutionPlan  *executionPlan        `json:"execution_plan,omitempty"`
	Deployment     *deploymentRecord     `json:"deployment,omitempty"`
	Infrastructure *infrastructureRecord `json:"infrastructure,omitempty"`
}

type asyncJobStats struct {
	QueueSize     int `json:"queue_size"`
	QueueDepth    int `json:"queue_depth"`
	WorkerCount   int `json:"worker_count"`
	HistorySize   int `json:"history_size"`
	ActiveJobs    int `json:"active_jobs"`
	QueuedJobs    int `json:"queued_jobs"`
	RunningJobs   int `json:"running_jobs"`
	SucceededJobs int `json:"succeeded_jobs"`
	FailedJobs    int `json:"failed_jobs"`
}

type asyncJobManager struct {
	logger *logrus.Logger

	mu             sync.RWMutex
	jobs           map[string]*asyncJob
	tasks          map[string]asyncJobTask
	completedOrder []string
	queue          chan string
	queueSize      int
	workerCount    int
	historySize    int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	stopped        bool
}

func newAsyncJobManager(cfg *config.Config, logger *logrus.Logger) *asyncJobManager {
	queueSize := 32
	workerCount := 2
	historySize := 50
	if cfg != nil {
		if cfg.JobQueueSize > 0 {
			queueSize = cfg.JobQueueSize
		}
		if cfg.JobWorkerCount > 0 {
			workerCount = cfg.JobWorkerCount
		}
		if cfg.JobHistorySize > 0 {
			historySize = cfg.JobHistorySize
		}
	}
	if queueSize < workerCount {
		queueSize = workerCount
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &asyncJobManager{
		logger:      logger,
		jobs:        make(map[string]*asyncJob),
		tasks:       make(map[string]asyncJobTask),
		queue:       make(chan string, queueSize),
		queueSize:   queueSize,
		workerCount: workerCount,
		historySize: historySize,
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < workerCount; i++ {
		m.wg.Add(1)
		go m.worker()
	}

	return m
}

func (m *asyncJobManager) Submit(_ context.Context, submission asyncJobSubmission) (*asyncJob, error) {
	if submission.Task == nil {
		return nil, errors.New("async job task is required")
	}

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil, errAsyncJobStopped
	}

	job := &asyncJob{
		ID:            newAsyncJobID(),
		Kind:          string(submission.Kind),
		Intent:        submission.Intent,
		Backend:       submission.Backend,
		Source:        submission.Source,
		UserID:        submission.UserID,
		Roles:         append([]string(nil), submission.Roles...),
		Status:        string(asyncJobStatusQueued),
		Summary:       submission.Summary,
		Detail:        submission.Detail,
		ResourceType:  submission.ResourceType,
		ResourceName:  submission.ResourceName,
		Namespace:     submission.Namespace,
		Provider:      submission.Provider,
		Route:         submission.Route,
		Mode:          submission.Mode,
		QueuePosition: len(m.queue) + 1,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		ExecutionPlan: cloneExecutionPlan(submission.ExecutionPlan),
	}
	m.jobs[job.ID] = job
	m.tasks[job.ID] = submission.Task
	if len(m.queue) >= cap(m.queue) {
		delete(m.jobs, job.ID)
		delete(m.tasks, job.ID)
		m.mu.Unlock()
		return nil, errAsyncJobQueueFull
	}
	m.mu.Unlock()

	select {
	case m.queue <- job.ID:
	case <-m.ctx.Done():
		m.mu.Lock()
		delete(m.jobs, job.ID)
		delete(m.tasks, job.ID)
		m.mu.Unlock()
		return nil, errAsyncJobStopped
	}

	if submitted, ok := m.Get(job.ID); ok {
		return submitted, nil
	}
	return nil, errAsyncJobStopped
}

func (m *asyncJobManager) Get(id string) (*asyncJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}

	return cloneAsyncJob(job), true
}

func (m *asyncJobManager) List() []asyncJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]asyncJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, *cloneAsyncJob(job))
	}

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID > jobs[j].ID
		}
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})
	return jobs
}

func (m *asyncJobManager) Stats() asyncJobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := asyncJobStats{
		QueueSize:   cap(m.queue),
		QueueDepth:  len(m.queue),
		WorkerCount: m.workerCount,
		HistorySize: m.historySize,
	}

	for _, job := range m.jobs {
		stats.ActiveJobs++
		switch job.Status {
		case string(asyncJobStatusQueued):
			stats.QueuedJobs++
		case string(asyncJobStatusRunning):
			stats.RunningJobs++
		case string(asyncJobStatusSucceeded):
			stats.SucceededJobs++
		case string(asyncJobStatusFailed):
			stats.FailedJobs++
		}
	}

	return stats
}

func (m *asyncJobManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return nil
	}
	m.stopped = true
	m.mu.Unlock()

	m.cancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *asyncJobManager) worker() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case id := <-m.queue:
			if id == "" {
				continue
			}
			m.run(id)
		}
	}
}

func (m *asyncJobManager) run(id string) {
	m.mu.Lock()
	job := m.jobs[id]
	task := m.tasks[id]
	if job == nil || task == nil {
		m.mu.Unlock()
		return
	}

	now := time.Now().UTC()
	job.Status = string(asyncJobStatusRunning)
	job.StartedAt = now
	job.UpdatedAt = now
	m.mu.Unlock()

	result, err := task(m.ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	job = m.jobs[id]
	if job == nil {
		delete(m.tasks, id)
		return
	}

	now = time.Now().UTC()
	job.CompletedAt = now
	job.UpdatedAt = now
	delete(m.tasks, id)

	if err != nil {
		job.Status = string(asyncJobStatusFailed)
		job.Error = err.Error()
		m.appendCompletedLocked(id)
		return
	}

	if result != nil {
		job.Deployment = cloneDeploymentRecord(result.Deployment)
		job.Infrastructure = cloneInfrastructureRecord(result.Infrastructure)
	}
	job.Status = string(asyncJobStatusSucceeded)
	job.Error = ""
	m.appendCompletedLocked(id)
}

func (m *asyncJobManager) appendCompletedLocked(id string) {
	job := m.jobs[id]
	if job == nil {
		return
	}

	if job.Status != string(asyncJobStatusSucceeded) && job.Status != string(asyncJobStatusFailed) {
		return
	}

	m.completedOrder = append(m.completedOrder, id)
	if m.historySize <= 0 || len(m.completedOrder) <= m.historySize {
		return
	}

	for len(m.completedOrder) > m.historySize {
		oldest := m.completedOrder[0]
		m.completedOrder = m.completedOrder[1:]
		if oldest == id {
			continue
		}
		if stale := m.jobs[oldest]; stale != nil && (stale.Status == string(asyncJobStatusSucceeded) || stale.Status == string(asyncJobStatusFailed)) {
			delete(m.jobs, oldest)
		}
	}
}

func newAsyncJobID() string {
	var entropy [6]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("job-%d-%s", time.Now().UnixNano(), hex.EncodeToString(entropy[:]))
}

func cloneAsyncJob(job *asyncJob) *asyncJob {
	if job == nil {
		return nil
	}
	clone := *job
	clone.Roles = append([]string(nil), job.Roles...)
	clone.ExecutionPlan = cloneExecutionPlan(job.ExecutionPlan)
	clone.Deployment = cloneDeploymentRecord(job.Deployment)
	clone.Infrastructure = cloneInfrastructureRecord(job.Infrastructure)
	return &clone
}

func cloneExecutionPlan(plan *executionPlan) *executionPlan {
	if plan == nil {
		return nil
	}
	clone := *plan
	clone.Notes = append([]string(nil), plan.Notes...)
	return &clone
}

func cloneDeploymentRecord(record *deploymentRecord) *deploymentRecord {
	if record == nil {
		return nil
	}
	clone := *record
	clone.Conditions = append([]string(nil), record.Conditions...)
	clone.ExecutionPlan = cloneExecutionPlan(record.ExecutionPlan)
	return &clone
}

func cloneInfrastructureRecord(record *infrastructureRecord) *infrastructureRecord {
	if record == nil {
		return nil
	}
	clone := *record
	clone.Artifacts = append([]string(nil), record.Artifacts...)
	if record.Inputs != nil {
		clone.Inputs = make(map[string]string, len(record.Inputs))
		for key, value := range record.Inputs {
			clone.Inputs[key] = value
		}
	}
	if record.Outputs != nil {
		clone.Outputs = make(map[string]string, len(record.Outputs))
		for key, value := range record.Outputs {
			clone.Outputs[key] = value
		}
	}
	clone.ExecutionPlan = cloneExecutionPlan(record.ExecutionPlan)
	return &clone
}
