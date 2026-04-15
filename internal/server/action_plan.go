package server

import (
	"fmt"
	"strings"
)

type actionPlan struct {
	Title          string                `json:"title"`
	Intent         string                `json:"intent"`
	Mode           string                `json:"mode"`
	Summary        string                `json:"summary"`
	Confidence     string                `json:"confidence"`
	FocusService   *serviceStatusSummary `json:"focus_service,omitempty"`
	ExecutionPath  string                `json:"execution_path"`
	Steps          []actionPlanStep      `json:"steps"`
	Guardrails     []actionPlanItem      `json:"guardrails"`
	Evidence       []actionPlanItem      `json:"evidence"`
	Observability  []actionPlanItem      `json:"observability"`
	Approvals      []actionPlanItem      `json:"approvals"`
	OutcomePreview []actionPlanItem      `json:"outcome_preview"`
}

type actionPlanStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Owner  string `json:"owner"`
	Why    string `json:"why"`
}

type actionPlanItem struct {
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func buildActionPlan(
	queryResult catalogQueryResult,
	deployment *deploymentRecord,
	infrastructure *infrastructureRecord,
) actionPlan {
	plan := actionPlan{
		Title:         "Evidence-backed operator plan",
		Intent:        queryResult.Intent,
		Mode:          "advisory",
		Summary:       "Axiom uses AI to explain the safest next action while policy and execution remain deterministic.",
		Confidence:    actionPlanConfidence(queryResult, deployment, infrastructure),
		ExecutionPath: "Axiom reasoning -> deterministic policy gates -> selected execution engine",
		Guardrails: []actionPlanItem{
			{
				Label:  "Deterministic execution only",
				Status: "active",
				Detail: "AI can propose and summarize actions, but rollout and infrastructure changes are executed through explicit control-plane integrations.",
			},
			{
				Label:  "Release gate awareness",
				Status: "active",
				Detail: "Ownership gaps, evidence freshness, and runtime health remain first-class blockers before promotion.",
			},
		},
		Observability: []actionPlanItem{
			{
				Label:  "Health endpoints",
				Status: "active",
				Detail: "Runtime liveness, readiness, and platform status are exposed through backend endpoints.",
			},
			{
				Label:  "Operational trail",
				Status: "active",
				Detail: "Requests, delivery paths, and audit events are surfaced so operators can verify what Axiom recommended and what actually ran.",
			},
		},
		Approvals: []actionPlanItem{
			{
				Label:  "Human-in-the-loop release control",
				Status: "recommended",
				Detail: "High-risk or unclear changes should stay approval-gated even when AI can draft the path.",
			},
		},
		OutcomePreview: []actionPlanItem{
			{
				Label:  "Operator outcome",
				Status: "ready",
				Detail: "The assistant should reduce routing, evidence, and status-check effort without hiding execution details.",
			},
		},
	}

	if queryResult.FocusService != nil {
		service := queryResult.FocusService.Service
		plan.FocusService = &serviceStatusSummary{
			ID:           service.ID,
			Name:         service.Name,
			Status:       service.Status,
			ReleaseState: service.ReleaseState,
			RiskLevel:    service.RiskLevel,
			HealthScore:  service.HealthScore,
		}
		plan.Evidence = append(plan.Evidence, actionPlanItem{
			Label:  "Service evidence",
			Status: normalizePlanState(service.ReleaseState),
			Detail: fmt.Sprintf("%s has %d evidence controls attached and a compliance score of %d.", service.Name, len(queryResult.EvidencePack), service.ComplianceScore),
		})
	}

	switch {
	case strings.HasPrefix(queryResult.Intent, "deployment_apply_argocd"):
		return populateArgoDeploymentPlan(plan, queryResult, deployment)
	case strings.HasPrefix(queryResult.Intent, "deployment_status_argocd"):
		return populateArgoStatusPlan(plan, queryResult, deployment)
	case strings.HasPrefix(queryResult.Intent, "deployment_apply"):
		return populateDirectDeploymentPlan(plan, queryResult, deployment)
	case strings.HasPrefix(queryResult.Intent, "deployment_status"):
		return populateDirectStatusPlan(plan, queryResult, deployment)
	case strings.HasPrefix(queryResult.Intent, "infrastructure_apply"):
		return populateInfrastructurePlan(plan, queryResult, infrastructure)
	default:
		return populateAdvisoryPlan(plan, queryResult)
	}
}

func populateArgoDeploymentPlan(plan actionPlan, queryResult catalogQueryResult, deployment *deploymentRecord) actionPlan {
	plan.Mode = "delivery"
	plan.Title = "AI-guided GitOps rollout"
	plan.ExecutionPath = "Axiom intent -> GitHub branch -> Argo CD application -> Kubernetes rollout"
	if deployment != nil {
		if isQueuedExecution(deployment.ExecutionState, deployment.Phase) {
			plan.Summary = fmt.Sprintf("%s has been accepted and queued through GitHub and Argo CD.", deployment.Name)
		} else {
			plan.Summary = fmt.Sprintf("%s is routed through GitHub and Argo CD so the rollout stays reviewable and observable.", deployment.Name)
		}
		plan.Steps = []actionPlanStep{
			{
				Name:   "Intent normalized",
				Status: "completed",
				Detail: fmt.Sprintf("Deployment request for %s was translated into a GitOps-safe rollout specification.", deployment.Name),
				Owner:  "Axiom",
				Why:    "Removes manual YAML authoring while keeping rollout semantics explicit.",
			},
			{
				Name:   "GitHub delivery branch",
				Status: planStatusForPhase(deployment.Phase),
				Detail: queuedOrLiveDetail(deployment.ExecutionState, fmt.Sprintf("Branch %s and path %s track the desired state for the rollout.", fallbackString(deployment.Revision, "generated"), fallbackString(deployment.Path, "deployments/ai-delivery")), "The job is queued and the delivery branch will be created by the worker."),
				Owner:  "GitHub",
				Why:    "Operators can audit and reproduce the exact desired state.",
			},
			{
				Name:   "Argo CD reconciliation",
				Status: planStatusForArgo(deployment.SyncStatus, deployment.HealthStatus),
				Detail: fmt.Sprintf("Application %s is reporting sync %s and health %s.", fallbackString(deployment.ApplicationName, deployment.Name), fallbackString(deployment.SyncStatus, "unknown"), fallbackString(deployment.HealthStatus, deployment.Phase)),
				Owner:  "Argo CD",
				Why:    "GitOps keeps drift correction and rollout state outside the AI layer.",
			},
		}
	}
	plan.Guardrails = append(plan.Guardrails, actionPlanItem{
		Label:  "GitOps provenance",
		Status: "active",
		Detail: "Every AI-generated deployment request becomes a branch-backed artifact before reconciliation.",
	})
	plan.OutcomePreview = append(plan.OutcomePreview, actionPlanItem{
		Label:  "Delivery result",
		Status: normalizePlanState(fallbackString(deployment.Phase, "watch")),
		Detail: queuedOrLiveDetail(deployment.ExecutionState, "Expected operator experience: ask for a rollout, inspect the branch/application trail, then confirm live status from the same workspace.", "Expected operator experience: the job is queued and the worker will publish the final rollout trail."),
	})
	return plan
}

func populateArgoStatusPlan(plan actionPlan, queryResult catalogQueryResult, deployment *deploymentRecord) actionPlan {
	plan.Mode = "delivery"
	plan.Title = "GitOps deployment status review"
	plan.ExecutionPath = "Axiom status query -> Argo CD status -> Kubernetes deployment state"
	if deployment != nil {
		plan.Summary = fmt.Sprintf("%s status is derived from live Argo CD and Kubernetes signals.", deployment.Name)
		plan.Steps = []actionPlanStep{
			{
				Name:   "Application status fetched",
				Status: "completed",
				Detail: fmt.Sprintf("Sync %s and health %s were read from application %s.", fallbackString(deployment.SyncStatus, "unknown"), fallbackString(deployment.HealthStatus, "unknown"), fallbackString(deployment.ApplicationName, deployment.Name)),
				Owner:  "Argo CD",
				Why:    "Status remains grounded in control-plane data instead of model inference.",
			},
			{
				Name:   "Replica state verified",
				Status: planStatusForPhase(deployment.Phase),
				Detail: fmt.Sprintf("%d/%d replicas are ready in namespace %s.", deployment.ReadyReplicas, deployment.Replicas, deployment.Namespace),
				Owner:  "Kubernetes",
				Why:    "Operators need execution truth, not just narrative summaries.",
			},
		}
	}
	return plan
}

func populateDirectDeploymentPlan(plan actionPlan, queryResult catalogQueryResult, deployment *deploymentRecord) actionPlan {
	plan.Mode = "delivery"
	plan.Title = "AI-guided direct deployment"
	plan.ExecutionPath = "Axiom intent -> Kubernetes deployment manager -> rollout verification"
	if deployment != nil {
		if isQueuedExecution(deployment.ExecutionState, deployment.Phase) {
			plan.Summary = fmt.Sprintf("%s has been accepted and queued for Kubernetes deployment execution.", deployment.Name)
		} else {
			plan.Summary = fmt.Sprintf("%s was applied directly to Kubernetes and then verified through rollout status.", deployment.Name)
		}
		plan.Steps = []actionPlanStep{
			{
				Name:   "Deployment spec prepared",
				Status: "completed",
				Detail: queuedOrLiveDetail(deployment.ExecutionState, fmt.Sprintf("The request was normalized to image %s with service type %s.", deployment.Image, fallbackString(deployment.ServiceType, "ClusterIP")), "The request is queued and will be normalized by the worker."),
				Owner:  "Axiom",
				Why:    "Developers can request a deployment without hand-authoring manifests.",
			},
			{
				Name:   "Rollout checked",
				Status: planStatusForPhase(deployment.Phase),
				Detail: fmt.Sprintf("%d/%d replicas are ready in namespace %s.", deployment.ReadyReplicas, deployment.Replicas, deployment.Namespace),
				Owner:  "Kubernetes",
				Why:    "Execution remains verifiable after AI translates intent.",
			},
		}
	}
	plan.Guardrails = append(plan.Guardrails, actionPlanItem{
		Label:  "Direct path warning",
		Status: "review",
		Detail: "Direct deployment is useful for controlled environments; GitOps remains the preferred path for production change history.",
	})
	return plan
}

func populateDirectStatusPlan(plan actionPlan, queryResult catalogQueryResult, deployment *deploymentRecord) actionPlan {
	plan.Mode = "delivery"
	plan.Title = "Runtime deployment status"
	plan.ExecutionPath = "Axiom status query -> Kubernetes deployment status"
	if deployment != nil {
		plan.Summary = fmt.Sprintf("%s status is read directly from Kubernetes rollout state.", deployment.Name)
		plan.Steps = []actionPlanStep{
			{
				Name:   "Deployment state read",
				Status: "completed",
				Detail: fmt.Sprintf("%d/%d replicas are ready with phase %s.", deployment.ReadyReplicas, deployment.Replicas, deployment.Phase),
				Owner:  "Kubernetes",
				Why:    "Status checks should stay deterministic and current.",
			},
		}
	}
	return plan
}

func populateInfrastructurePlan(plan actionPlan, queryResult catalogQueryResult, infrastructure *infrastructureRecord) actionPlan {
	plan.Mode = "infrastructure"
	plan.Title = "AI-guided infrastructure request"
	plan.ExecutionPath = "Axiom intent -> GitHub delivery artifacts -> Argo CD and selected infra engine"
	if infrastructure != nil {
		if isQueuedExecution(infrastructure.ExecutionState, infrastructure.Phase) {
			plan.Summary = fmt.Sprintf("%s infrastructure for %s has been accepted and queued for async execution.", strings.Title(infrastructure.Provider), infrastructure.TargetNamespace)
		} else {
			plan.Summary = fmt.Sprintf("%s infrastructure for %s is routed through the %s path.", strings.Title(infrastructure.Provider), infrastructure.TargetNamespace, strings.Title(infrastructure.Provider))
		}
		plan.Steps = []actionPlanStep{
			{
				Name:   "Infrastructure intent normalized",
				Status: "completed",
				Detail: queuedOrLiveDetail(infrastructure.ExecutionState, fmt.Sprintf("The request targets namespace %s using %s.", infrastructure.TargetNamespace, strings.Title(infrastructure.Provider)), "The request is queued and will be normalized by the worker."),
				Owner:  "Axiom",
				Why:    "AI reduces form-filling and routing complexity for platform consumers.",
			},
			{
				Name:   "GitHub-backed delivery artifact",
				Status: "completed",
				Detail: fmt.Sprintf("Revision %s and path %s capture the requested infrastructure state.", fallbackString(infrastructure.Revision, "generated"), fallbackString(infrastructure.Path, "deployments/ai-delivery")),
				Owner:  "GitHub",
				Why:    "Infrastructure requests should be reviewable and reproducible.",
			},
			{
				Name:   "Execution engine",
				Status: normalizePlanState(infrastructure.Phase),
				Detail: infrastructure.Message,
				Owner:  strings.Title(infrastructure.Provider),
				Why:    "The selected control plane, not the model, is responsible for reconciliation.",
			},
		}
	}
	if infrastructure != nil && infrastructure.Provider == "crossplane" {
		plan.Guardrails = append(plan.Guardrails, actionPlanItem{
			Label:  "Crossplane prerequisite",
			Status: "review",
			Detail: "Crossplane claims require installed providers/configurations in the target cluster before execution can complete.",
		})
	} else {
		plan.Guardrails = append(plan.Guardrails, actionPlanItem{
			Label:  "Terraform isolation",
			Status: "active",
			Detail: "Terraform execution is isolated behind a job-driven control path instead of running inline with the assistant request.",
		})
	}
	plan.Observability = append(plan.Observability, actionPlanItem{
		Label:  "Infra reconciliation visibility",
		Status: "active",
		Detail: "Application sync, job status, and target namespace data are retained so operators can confirm execution without leaving Axiom.",
	})
	plan.OutcomePreview = append(plan.OutcomePreview, actionPlanItem{
		Label:  "Platform outcome",
		Status: normalizePlanState(fallbackString(infrastructure.Phase, "watch")),
		Detail: queuedOrLiveDetail(infrastructure.ExecutionState, "Infrastructure should feel request-driven to developers while staying reviewable and operator-controlled for the platform team.", "Infrastructure is queued and will be reconciled by the worker."),
	})
	return plan
}

func isQueuedExecution(state, phase string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "queued", "pending":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "queued", "pending":
		return true
	}
	return false
}

func queuedOrLiveDetail(state, liveDetail, queuedDetail string) string {
	if isQueuedExecution(state, "") {
		return queuedDetail
	}
	return liveDetail
}

func populateAdvisoryPlan(plan actionPlan, queryResult catalogQueryResult) actionPlan {
	plan.Mode = "advisory"
	plan.Title = "AI-guided release and remediation plan"
	plan.ExecutionPath = "Axiom intent -> service intelligence -> evidence-backed next best actions"
	plan.Summary = "Axiom uses AI where it removes coordination and interpretation effort: release decisions, evidence synthesis, and remediation guidance."

	steps := []actionPlanStep{
		{
			Name:   "Intent classified",
			Status: "completed",
			Detail: fmt.Sprintf("The request was classified as %s.", humanizeIntent(queryResult.Intent)),
			Owner:  "Axiom",
			Why:    "Operators should not need to navigate multiple screens to decide the next step.",
		},
	}

	if queryResult.FocusService != nil {
		steps = append(steps,
			actionPlanStep{
				Name:   "Risk and readiness reviewed",
				Status: normalizePlanState(queryResult.ReleaseReadiness.State),
				Detail: fmt.Sprintf("%s is %s with risk level %s.", queryResult.FocusService.Service.Name, queryResult.ReleaseReadiness.State, queryResult.ReleaseReadiness.RiskLevel),
				Owner:  "Axiom",
				Why:    "AI is valuable when it compresses multiple health and compliance signals into an operator decision.",
			},
			actionPlanStep{
				Name:   "Next-best actions prepared",
				Status: "ready",
				Detail: renderActionList(queryResult.NextSteps, 3),
				Owner:  "Team",
				Why:    "Axiom should reduce follow-up coordination, not just point out problems.",
			},
		)
	}
	plan.Steps = steps

	for _, evidence := range queryResult.EvidencePack {
		plan.Evidence = append(plan.Evidence, actionPlanItem{
			Label:  evidence.Type,
			Status: normalizePlanState(evidence.Status),
			Detail: evidence.Summary,
		})
	}
	if len(plan.Evidence) == 0 {
		plan.Evidence = append(plan.Evidence, actionPlanItem{
			Label:  "Portfolio evidence",
			Status: "review",
			Detail: "No service-specific evidence pack was attached to this request, so Axiom fell back to portfolio-level guidance.",
		})
	}

	for _, action := range queryResult.NextSteps {
		plan.OutcomePreview = append(plan.OutcomePreview, actionPlanItem{
			Label:  strings.Title(action.Priority) + " next step",
			Status: "ready",
			Detail: fmt.Sprintf("%s (%s, %s impact)", action.Action, action.Owner, action.Impact),
		})
	}

	return plan
}

func normalizePlanState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready", "healthy", "active", "completed", "synced":
		return "ready"
	case "progressing", "watch", "warning", "pending", "review":
		return "review"
	case "blocked", "failed", "degraded", "unready":
		return "attention"
	default:
		return "review"
	}
}

func planStatusForPhase(phase string) string {
	return normalizePlanState(phase)
}

func planStatusForArgo(syncStatus, healthStatus string) string {
	if strings.EqualFold(syncStatus, "synced") && strings.EqualFold(healthStatus, "healthy") {
		return "ready"
	}
	if strings.EqualFold(healthStatus, "degraded") || strings.EqualFold(syncStatus, "outofsync") {
		return "attention"
	}
	return "review"
}

func actionPlanConfidence(queryResult catalogQueryResult, deployment *deploymentRecord, infrastructure *infrastructureRecord) string {
	switch {
	case deployment != nil && strings.TrimSpace(deployment.Phase) != "":
		return "high"
	case infrastructure != nil && strings.TrimSpace(infrastructure.Phase) != "":
		return "high"
	case queryResult.FocusService != nil:
		return "medium"
	default:
		return "medium"
	}
}

func humanizeIntent(intent string) string {
	if intent == "" {
		return "general guidance"
	}
	return strings.ReplaceAll(intent, "_", " ")
}

func withIntent(result catalogQueryResult, intent string) catalogQueryResult {
	result.Intent = intent
	return result
}
