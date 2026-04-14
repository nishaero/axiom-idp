package server

import (
	"fmt"
	"strings"
)

type demoService struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Owner              string   `json:"owner"`
	Team               string   `json:"team"`
	Tier               string   `json:"tier"`
	Status             string   `json:"status"`
	ReleaseState       string   `json:"release_state"`
	Description        string   `json:"description"`
	RiskLevel          string   `json:"risk_level"`
	HealthScore        int      `json:"health_score"`
	ComplianceScore    int      `json:"compliance_score"`
	DeploymentVelocity string   `json:"deployment_velocity"`
	LastDeployed       string   `json:"last_deployed"`
	Signals            []string `json:"signals"`
	Evidence           []string `json:"evidence"`
	Dependencies       []string `json:"dependencies"`
}

type serviceEvidence struct {
	Type          string `json:"type"`
	Status        string `json:"status"`
	Summary       string `json:"summary"`
	Source        string `json:"source"`
	LastValidated string `json:"last_validated"`
	Required      bool   `json:"required"`
}

type ownershipDrift struct {
	State         string   `json:"state"`
	Severity      string   `json:"severity"`
	ExpectedOwner string   `json:"expected_owner"`
	ActualOwner   string   `json:"actual_owner"`
	Findings      []string `json:"findings"`
}

type releaseReadiness struct {
	State     string `json:"state"`
	Score     int    `json:"score"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
}

type remediationAction struct {
	Priority string `json:"priority"`
	Action   string `json:"action"`
	Owner    string `json:"owner"`
	Effort   string `json:"effort"`
	Impact   string `json:"impact"`
}

type serviceIntelligence struct {
	ReleaseReadiness releaseReadiness    `json:"release_readiness"`
	EvidencePack     []serviceEvidence   `json:"evidence_pack"`
	OwnershipDrift   ownershipDrift      `json:"ownership_drift"`
	NextSteps        []remediationAction `json:"next_steps"`
	Signals          []string            `json:"signals"`
	RiskSummary      string              `json:"risk_summary"`
}

type catalogServiceView struct {
	Service      demoService         `json:"service"`
	Intelligence serviceIntelligence `json:"intelligence"`
}

type portfolioIntelligence struct {
	TotalServices    int                 `json:"total_services"`
	ReadyCount       int                 `json:"ready_count"`
	WatchCount       int                 `json:"watch_count"`
	BlockedCount     int                 `json:"blocked_count"`
	OwnerGapCount    int                 `json:"owner_gap_count"`
	RiskLevel        string              `json:"risk_level"`
	HighRiskServices []string            `json:"high_risk_services"`
	NextSteps        []remediationAction `json:"next_steps"`
}

type catalogQueryResult struct {
	Intent           string                `json:"intent"`
	FocusService     *catalogServiceView   `json:"focus_service,omitempty"`
	Portfolio        portfolioIntelligence `json:"portfolio"`
	MatchingServices []catalogServiceView  `json:"matching_services,omitempty"`
	ReleaseReadiness releaseReadiness      `json:"release_readiness,omitempty"`
	EvidencePack     []serviceEvidence     `json:"evidence_pack,omitempty"`
	OwnershipDrift   ownershipDrift        `json:"ownership_drift,omitempty"`
	NextSteps        []remediationAction   `json:"next_steps,omitempty"`
	KeyFindings      []string              `json:"key_findings,omitempty"`
}

type catalogOverview struct {
	TotalServices          int      `json:"total_services"`
	ReadyServices          int      `json:"ready_services"`
	WatchServices          int      `json:"watch_services"`
	BlockedServices        int      `json:"blocked_services"`
	CriticalServices       int      `json:"critical_services"`
	OwnerGapCount          int      `json:"owner_gap_count"`
	HealthyServices        int      `json:"healthy_services"`
	DegradedServices       int      `json:"degraded_services"`
	ReleaseReadiness       int      `json:"release_readiness"`
	EvidenceCoverage       int      `json:"evidence_coverage"`
	HighestRiskServiceID   string   `json:"highest_risk_service_id"`
	HighestRiskServiceName string   `json:"highest_risk_service_name"`
	PriorityActions        []string `json:"priority_actions"`
}

type serviceInsight struct {
	Service            demoService         `json:"service"`
	ReleaseDecision    string              `json:"release_decision"`
	RiskSummary        string              `json:"risk_summary"`
	EvidenceSummary    string              `json:"evidence_summary"`
	OwnershipSummary   string              `json:"ownership_summary"`
	OperationalFocus   []string            `json:"operational_focus"`
	MissingEvidence    []string            `json:"missing_evidence"`
	RecommendedActions []string            `json:"recommended_actions"`
	ReleaseReadiness   releaseReadiness    `json:"release_readiness"`
	EvidencePack       []serviceEvidence   `json:"evidence_pack"`
	OwnershipDrift     ownershipDrift      `json:"ownership_drift"`
	NextSteps          []remediationAction `json:"next_steps"`
}

var demoCatalog = []demoService{
	{
		ID:                 "svc-auth",
		Name:               "identity-gateway",
		Owner:              "Security Platform",
		Team:               "Security Engineering",
		Tier:               "critical",
		Status:             "healthy",
		ReleaseState:       "ready",
		Description:        "Authentication and authorization gateway for internal applications.",
		RiskLevel:          "low",
		HealthScore:        96,
		ComplianceScore:    97,
		DeploymentVelocity: "controlled",
		LastDeployed:       "2 hours ago",
		Signals:            []string{"mfa enforced", "audit logging green", "sast+dast green"},
		Evidence:           []string{"owner assigned", "rollback tested", "logs retained 180d"},
		Dependencies:       []string{"audit-ledger", "notification-mesh"},
	},
	{
		ID:                 "svc-payments",
		Name:               "payments-api",
		Owner:              "Revenue Platform",
		Team:               "Commerce",
		Tier:               "critical",
		Status:             "healthy",
		ReleaseState:       "watch",
		Description:        "Revenue-critical payment orchestration service with guarded rollouts.",
		RiskLevel:          "medium",
		HealthScore:        88,
		ComplianceScore:    84,
		DeploymentVelocity: "high",
		LastDeployed:       "6 hours ago",
		Signals:            []string{"manual approval required", "feature flags active", "higher change rate"},
		Evidence:           []string{"owner assigned", "rollback plan current", "on-call coverage active"},
		Dependencies:       []string{"identity-gateway", "audit-ledger"},
	},
	{
		ID:                 "svc-orders",
		Name:               "orders-worker",
		Owner:              "Commerce Fulfilment",
		Team:               "Commerce",
		Tier:               "important",
		Status:             "degraded",
		ReleaseState:       "watch",
		Description:        "Event-driven order processing worker with backlog sensitivity.",
		RiskLevel:          "high",
		HealthScore:        74,
		ComplianceScore:    78,
		DeploymentVelocity: "moderate",
		LastDeployed:       "1 day ago",
		Signals:            []string{"queue lag elevated", "retries increasing", "dependency drift"},
		Evidence:           []string{"owner assigned", "runbook current"},
		Dependencies:       []string{"payments-api", "notification-mesh"},
	},
	{
		ID:                 "svc-ledger",
		Name:               "audit-ledger",
		Owner:              "Governance Core",
		Team:               "Governance",
		Tier:               "regulated",
		Status:             "healthy",
		ReleaseState:       "ready",
		Description:        "Append-only evidence store for deployment traces and control history.",
		RiskLevel:          "low",
		HealthScore:        99,
		ComplianceScore:    99,
		DeploymentVelocity: "controlled",
		LastDeployed:       "3 days ago",
		Signals:            []string{"immutability enforced", "integrity checks green", "exports ready"},
		Evidence:           []string{"worm retention", "tamper checks green", "export pipeline validated"},
		Dependencies:       []string{"identity-gateway"},
	},
	{
		ID:                 "svc-data",
		Name:               "data-bridge",
		Owner:              "Unassigned",
		Team:               "Data Platform",
		Tier:               "standard",
		Status:             "degraded",
		ReleaseState:       "blocked",
		Description:        "Analytics export path with stale evidence and unresolved ownership.",
		RiskLevel:          "high",
		HealthScore:        68,
		ComplianceScore:    54,
		DeploymentVelocity: "uncontrolled",
		LastDeployed:       "5 days ago",
		Signals:            []string{"missing owner", "security review pending", "slo missing"},
		Evidence:           []string{"schema checks partial"},
		Dependencies:       []string{"audit-ledger"},
	},
}

func buildCatalogViews(services []demoService) []catalogServiceView {
	views := make([]catalogServiceView, 0, len(services))
	for _, service := range services {
		views = append(views, buildCatalogView(service))
	}
	return views
}

func buildCatalogView(service demoService) catalogServiceView {
	return catalogServiceView{
		Service:      service,
		Intelligence: buildServiceIntelligence(service),
	}
}

func buildServiceIntelligence(service demoService) serviceIntelligence {
	readinessState := service.ReleaseState
	riskScore := 15
	riskLevel := service.RiskLevel
	reasons := []string{fmt.Sprintf("%s status is %s", service.Name, service.Status)}
	nextSteps := make([]remediationAction, 0, 4)
	evidence := make([]serviceEvidence, 0, len(service.Evidence)+3)

	if service.Status != "healthy" {
		riskScore += 20
		reasons = append(reasons, "runtime state is degraded")
	}
	if service.Tier == "critical" {
		riskScore += 10
		reasons = append(reasons, "service is critical tier")
	}
	if service.ComplianceScore < 80 {
		riskScore += 15
		reasons = append(reasons, "compliance evidence is stale")
	}
	if strings.EqualFold(service.Owner, "Unassigned") {
		riskScore += 20
		reasons = append(reasons, "ownership is unresolved")
	}
	if readinessState == "watch" {
		riskScore += 20
	}
	if readinessState == "blocked" {
		riskScore += 45
	}

	readinessScore := clampInt((service.HealthScore+service.ComplianceScore)/2-(riskScore/4), 0, 100)
	switch {
	case readinessScore >= 80 && readinessState == "ready" && service.Status == "healthy":
		riskLevel = "low"
	case readinessScore >= 55:
		riskLevel = "medium"
	default:
		riskLevel = "high"
	}

	ownership := ownershipDrift{
		State:         "none",
		Severity:      "none",
		ExpectedOwner: service.Owner,
		ActualOwner:   service.Owner,
		Findings:      []string{"owner matches the catalog record"},
	}
	if strings.EqualFold(service.Owner, "Unassigned") {
		ownership.State = "drift"
		ownership.Severity = "high"
		ownership.ExpectedOwner = service.Team
		ownership.Findings = []string{fmt.Sprintf("%s requires an accountable owner", service.Name)}
		nextSteps = append(nextSteps, remediationAction{
			Priority: "high",
			Action:   "assign an accountable owner and update service metadata",
			Owner:    service.Team,
			Effort:   "small",
			Impact:   "restores release approval clarity",
		})
	}

	for _, item := range service.Evidence {
		evidence = append(evidence, serviceEvidence{
			Type:          "control_evidence",
			Status:        "passed",
			Summary:       item,
			Source:        "catalog metadata",
			LastValidated: service.LastDeployed,
			Required:      true,
		})
	}

	validationStatus := "passed"
	if service.Status != "healthy" {
		validationStatus = "warn"
	}
	if service.ReleaseState == "blocked" {
		validationStatus = "failed"
	}
	evidence = append(evidence,
		serviceEvidence{
			Type:          "deployment_validation",
			Status:        validationStatus,
			Summary:       fmt.Sprintf("%s deployment validation is %s", service.Name, validationStatus),
			Source:        "release automation",
			LastValidated: service.LastDeployed,
			Required:      true,
		},
		serviceEvidence{
			Type:          "release_gate",
			Status:        service.ReleaseState,
			Summary:       fmt.Sprintf("release state is %s", service.ReleaseState),
			Source:        "catalog release state",
			LastValidated: service.LastDeployed,
			Required:      true,
		},
		serviceEvidence{
			Type:          "ownership_review",
			Status:        ownership.State,
			Summary:       ownership.Findings[0],
			Source:        "ownership metadata",
			LastValidated: service.LastDeployed,
			Required:      true,
		},
	)

	switch readinessState {
	case "blocked":
		nextSteps = append([]remediationAction{{
			Priority: "critical",
			Action:   "restore health and close the blocking release gates",
			Owner:    service.Owner,
			Effort:   "medium",
			Impact:   "unblocks the service for production use",
		}}, nextSteps...)
	case "watch":
		nextSteps = append([]remediationAction{{
			Priority: "medium",
			Action:   "review evidence and confirm the rollout window",
			Owner:    service.Owner,
			Effort:   "small",
			Impact:   "lowers release risk",
		}}, nextSteps...)
	default:
		nextSteps = append([]remediationAction{{
			Priority: "low",
			Action:   "keep the evidence pack attached to the release record",
			Owner:    service.Owner,
			Effort:   "small",
			Impact:   "keeps audit traceability intact",
		}}, nextSteps...)
	}

	return serviceIntelligence{
		ReleaseReadiness: releaseReadiness{
			State:     readinessState,
			Score:     readinessScore,
			RiskLevel: riskLevel,
			Reason:    strings.Join(reasons, "; "),
		},
		EvidencePack:   evidence,
		OwnershipDrift: ownership,
		NextSteps:      collapseActions(nextSteps),
		Signals:        append([]string(nil), service.Signals...),
		RiskSummary:    fmt.Sprintf("%s is %s risk with readiness score %d.", service.Name, riskLevel, readinessScore),
	}
}

func buildPortfolioIntelligence(services []demoService) portfolioIntelligence {
	portfolio := portfolioIntelligence{TotalServices: len(services)}
	nextSteps := make([]remediationAction, 0, len(services))
	highRisk := make([]string, 0, len(services))

	for _, service := range services {
		intel := buildServiceIntelligence(service)
		switch intel.ReleaseReadiness.State {
		case "ready":
			portfolio.ReadyCount++
		case "watch":
			portfolio.WatchCount++
		default:
			portfolio.BlockedCount++
		}
		if strings.EqualFold(service.Owner, "Unassigned") {
			portfolio.OwnerGapCount++
		}
		if intel.ReleaseReadiness.RiskLevel == "high" {
			highRisk = append(highRisk, service.Name)
		}
		nextSteps = append(nextSteps, intel.NextSteps...)
	}

	switch {
	case portfolio.BlockedCount > 0:
		portfolio.RiskLevel = "high"
	case portfolio.WatchCount > 0 || portfolio.OwnerGapCount > 0:
		portfolio.RiskLevel = "medium"
	default:
		portfolio.RiskLevel = "low"
	}

	portfolio.HighRiskServices = dedupeStrings(highRisk)
	portfolio.NextSteps = collapseActions(nextSteps)
	if len(portfolio.NextSteps) > 3 {
		portfolio.NextSteps = portfolio.NextSteps[:3]
	}

	return portfolio
}

func catalogSummary(services []demoService) catalogOverview {
	summary := catalogOverview{TotalServices: len(services)}
	if len(services) == 0 {
		return summary
	}

	totalCompliance := 0
	for _, service := range services {
		switch service.ReleaseState {
		case "ready":
			summary.ReadyServices++
		case "watch":
			summary.WatchServices++
		case "blocked":
			summary.BlockedServices++
		}
		if service.Tier == "critical" {
			summary.CriticalServices++
		}
		if service.Status == "healthy" {
			summary.HealthyServices++
		} else {
			summary.DegradedServices++
		}
		if strings.EqualFold(service.Owner, "Unassigned") {
			summary.OwnerGapCount++
		}
		totalCompliance += service.ComplianceScore
	}

	summary.ReleaseReadiness = clampInt((summary.ReadyServices*100)/summary.TotalServices-summary.BlockedServices*8-summary.OwnerGapCount*6, 35, 98)
	summary.EvidenceCoverage = totalCompliance / summary.TotalServices
	if service, ok := firstByRisk(services, "high"); ok {
		summary.HighestRiskServiceID = service.ID
		summary.HighestRiskServiceName = service.Name
	}
	summary.PriorityActions = topPriorityActions(services)

	return summary
}

func buildServiceInsight(service demoService) serviceInsight {
	intel := buildServiceIntelligence(service)
	missingEvidence := make([]string, 0, 3)
	if strings.EqualFold(service.Owner, "Unassigned") {
		missingEvidence = append(missingEvidence, "service owner assignment")
	}
	if service.ComplianceScore < 80 {
		missingEvidence = append(missingEvidence, "control evidence refresh")
	}
	if service.Status != "healthy" {
		missingEvidence = append(missingEvidence, "runtime stability proof")
	}
	if service.ReleaseState != "ready" {
		missingEvidence = append(missingEvidence, "release approval evidence")
	}

	decision := "release-ready"
	switch service.ReleaseState {
	case "watch":
		decision = "guarded-approval"
	case "blocked":
		decision = "hold-release"
	}

	ownership := fmt.Sprintf("%s owns %s within %s.", service.Owner, service.Name, service.Team)
	if strings.EqualFold(service.Owner, "Unassigned") {
		ownership = fmt.Sprintf("%s currently has no accountable service owner and requires assignment before change approval.", service.Name)
	}

	recommendedActions := []string{
		fmt.Sprintf("Review %s signals: %s.", service.Name, strings.Join(service.Signals, ", ")),
		fmt.Sprintf("Confirm dependency posture for %s.", strings.Join(service.Dependencies, ", ")),
	}
	if strings.EqualFold(service.Owner, "Unassigned") {
		recommendedActions = append(recommendedActions, fmt.Sprintf("Assign an owner and attach on-call accountability for %s.", service.Name))
	}
	if service.Status != "healthy" {
		recommendedActions = append(recommendedActions, fmt.Sprintf("Stabilize %s before promotion because the current runtime state is %s.", service.Name, service.Status))
	}
	if service.ComplianceScore < 85 {
		recommendedActions = append(recommendedActions, fmt.Sprintf("Refresh the control evidence set for %s and export a new approval packet.", service.Name))
	}

	return serviceInsight{
		Service:            service,
		ReleaseDecision:    decision,
		RiskSummary:        intel.RiskSummary,
		EvidenceSummary:    fmt.Sprintf("%s evidence status is %d%% complete. Current evidence includes %s.", service.Name, service.ComplianceScore, strings.Join(service.Evidence, ", ")),
		OwnershipSummary:   ownership,
		OperationalFocus:   append([]string(nil), service.Signals...),
		MissingEvidence:    missingEvidence,
		RecommendedActions: dedupeStrings(recommendedActions),
		ReleaseReadiness:   intel.ReleaseReadiness,
		EvidencePack:       intel.EvidencePack,
		OwnershipDrift:     intel.OwnershipDrift,
		NextSteps:          intel.NextSteps,
	}
}

func buildQueryResult(query string, services []demoService, serviceID string) catalogQueryResult {
	intent := deriveIntent(query)
	views := buildCatalogViews(services)
	portfolio := buildPortfolioIntelligence(services)
	focus := resolveServiceView(query, serviceID, views)
	matching := matchingServicesForIntent(intent, query, views)

	result := catalogQueryResult{
		Intent:           intent,
		Portfolio:        portfolio,
		MatchingServices: matching,
	}

	if focus != nil {
		result.FocusService = focus
		result.ReleaseReadiness = focus.Intelligence.ReleaseReadiness
		result.EvidencePack = append([]serviceEvidence(nil), focus.Intelligence.EvidencePack...)
		result.OwnershipDrift = focus.Intelligence.OwnershipDrift
		result.NextSteps = append([]remediationAction(nil), focus.Intelligence.NextSteps...)
		result.KeyFindings = []string{
			fmt.Sprintf("%s is %s", focus.Service.Name, focus.Intelligence.ReleaseReadiness.State),
			focus.Intelligence.ReleaseReadiness.Reason,
			focus.Intelligence.OwnershipDrift.State,
		}
	}

	if intent == "release_readiness" && len(portfolio.HighRiskServices) > 0 {
		result.KeyFindings = append(result.KeyFindings, fmt.Sprintf("high-risk services: %s", strings.Join(portfolio.HighRiskServices, ", ")))
	}

	result.KeyFindings = dedupeStrings(result.KeyFindings)
	return result
}

func matchingServicesForIntent(intent, query string, services []catalogServiceView) []catalogServiceView {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	matches := make([]catalogServiceView, 0, len(services))
	for _, service := range services {
		searchSpace := strings.ToLower(strings.Join([]string{
			service.Service.ID,
			service.Service.Name,
			service.Service.Owner,
			service.Service.Team,
			service.Service.Description,
			strings.Join(service.Service.Signals, " "),
		}, " "))

		switch {
		case lowerQuery != "" && strings.Contains(searchSpace, lowerQuery):
			matches = append(matches, service)
		case intent == "release_readiness" && service.Intelligence.ReleaseReadiness.State != "ready":
			matches = append(matches, service)
		case intent == "ownership_drift" && service.Intelligence.OwnershipDrift.State == "drift":
			matches = append(matches, service)
		case intent == "evidence_pack" && len(service.Intelligence.EvidencePack) > 0:
			matches = append(matches, service)
		}
	}
	return matches
}

func resolveServiceView(query, serviceID string, services []catalogServiceView) *catalogServiceView {
	normalizedID := strings.ToLower(strings.TrimSpace(serviceID))
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	for _, service := range services {
		if normalizedID != "" && (strings.EqualFold(service.Service.ID, normalizedID) || strings.EqualFold(service.Service.Name, normalizedID)) {
			copy := service
			return &copy
		}

		searchSpace := strings.ToLower(strings.Join([]string{
			service.Service.ID,
			service.Service.Name,
			service.Service.Owner,
			service.Service.Team,
			service.Service.Description,
		}, " "))
		if normalizedQuery != "" && (strings.Contains(searchSpace, normalizedQuery) || strings.Contains(normalizedQuery, strings.ToLower(service.Service.Name))) {
			copy := service
			return &copy
		}
	}
	return nil
}

func deriveIntent(query string) string {
	lower := strings.ToLower(strings.TrimSpace(query))
	switch {
	case containsAny(lower, "evidence", "audit", "compliance", "bsi c5", "attestation", "control"):
		return "evidence_pack"
	case containsAny(lower, "owner", "ownership", "drift", "accountable", "team"):
		return "ownership_drift"
	case containsAny(lower, "risk", "release", "deploy", "rollout", "ship", "readiness"):
		return "release_readiness"
	default:
		return "operational_guidance"
	}
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func renderActionList(actions []remediationAction, limit int) string {
	if len(actions) == 0 {
		return "no immediate actions"
	}
	if limit > 0 && len(actions) > limit {
		actions = actions[:limit]
	}
	lines := make([]string, 0, len(actions))
	for _, action := range actions {
		lines = append(lines, action.Action)
	}
	return strings.Join(lines, "; ")
}

func collapseActions(actions []remediationAction) []remediationAction {
	seen := make(map[string]struct{}, len(actions))
	out := make([]remediationAction, 0, len(actions))
	for _, action := range actions {
		key := strings.TrimSpace(strings.ToLower(action.Action))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(strings.ToLower(value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func topPriorityActions(services []demoService) []string {
	actions := []string{}
	if service, ok := firstOwnerGap(services); ok {
		actions = append(actions, fmt.Sprintf("Assign an accountable owner for %s before the next release window.", service.Name))
	}
	if service, ok := firstByReleaseState(services, "blocked"); ok {
		actions = append(actions, fmt.Sprintf("Unblock %s by closing missing evidence and security review tasks.", service.Name))
	}
	if service, ok := firstByRisk(services, "high"); ok {
		actions = append(actions, fmt.Sprintf("Review rollback readiness and dependency health for %s.", service.Name))
	}
	if len(actions) == 0 {
		actions = append(actions, "Maintain the current release cadence and keep evidence exports current.")
	}
	return actions
}

func firstOwnerGap(services []demoService) (demoService, bool) {
	for _, service := range services {
		if strings.EqualFold(service.Owner, "Unassigned") {
			return service, true
		}
	}
	return demoService{}, false
}

func firstByReleaseState(services []demoService, state string) (demoService, bool) {
	for _, service := range services {
		if service.ReleaseState == state {
			return service, true
		}
	}
	return demoService{}, false
}

func firstByRisk(services []demoService, risk string) (demoService, bool) {
	for _, service := range services {
		if service.RiskLevel == risk {
			return service, true
		}
	}
	return demoService{}, false
}
