import type { DemoCatalogService } from './demoContent'

export type ReleaseVerdict = 'go' | 'watch' | 'block'
export type DeploymentStatusTone = 'pending' | 'running' | 'completed' | 'degraded' | 'error'
export type WorkflowCategory = 'application' | 'infrastructure'
export type WorkflowLifecycle = 'planned' | 'executed'
export type WorkflowSource = 'Axiom IDP' | 'GitHub' | 'GitLab'
export type WorkflowProvider = 'Argo CD' | 'Crossplane' | 'Terraform' | 'Kubernetes API'

export interface ReleaseDecision {
  verdict: ReleaseVerdict
  title: string
  summary: string
  reasons: string[]
  nextSteps: string[]
  evidence: string[]
  aiPrompt: string
}

export interface WorkspaceSnapshot {
  totalServices: number
  readyServices: number
  watchServices: number
  blockedServices: number
  ownerlessServices: number
  criticalServices: number
  topService: DemoCatalogService
  topDecision: ReleaseDecision
}

export interface DeploymentRequestRecord {
  id: string
  serviceId: string
  serviceName: string
  workflowCategory: WorkflowCategory
  workflowLifecycle: WorkflowLifecycle
  source: WorkflowSource
  provider: WorkflowProvider
  route: string
  delivery: string
  target: string
  statusTone: DeploymentStatusTone
  statusLabel: string
  requestedAt: string
  summary: string
  timeline: string[]
  prompt: string
}

export interface AssistantDeploymentSnapshot {
  name: string
  namespace: string
  image: string
  replicas: number
  ready_replicas: number
  phase: string
  service_type?: string
}

function detectWorkflowCategory(prompt: string): WorkflowCategory {
  const normalized = prompt.toLowerCase()

  if (
    /infra|infrastructure|provision|provisioning|crossplane|terraform|cluster|network|bucket|database|db|secret|dns|ingress/i.test(
      normalized
    )
  ) {
    return 'infrastructure'
  }

  return 'application'
}

function detectWorkflowSource(prompt: string): WorkflowSource {
  const normalized = prompt.toLowerCase()

  if (normalized.includes('github')) {
    return 'GitHub'
  }

  if (normalized.includes('gitlab')) {
    return 'GitLab'
  }

  return 'Axiom IDP'
}

function detectWorkflowProvider(prompt: string, category: WorkflowCategory): WorkflowProvider {
  const normalized = prompt.toLowerCase()

  if (normalized.includes('argocd') || normalized.includes('argo cd') || normalized.includes('argo')) {
    return 'Argo CD'
  }

  if (normalized.includes('crossplane')) {
    return 'Crossplane'
  }

  if (normalized.includes('terraform')) {
    return 'Terraform'
  }

  return category === 'infrastructure' ? 'Crossplane' : 'Argo CD'
}

function normalizeWorkflowProvider(value: string, category: WorkflowCategory): WorkflowProvider {
  const normalized = value.toLowerCase()

  if (normalized.includes('argocd') || normalized.includes('argo cd') || normalized.includes('argo')) {
    return 'Argo CD'
  }

  if (normalized.includes('crossplane')) {
    return 'Crossplane'
  }

  if (normalized.includes('terraform')) {
    return 'Terraform'
  }

  if (normalized.includes('kubernetes')) {
    return 'Kubernetes API'
  }

  return category === 'infrastructure' ? 'Crossplane' : 'Argo CD'
}

function detectWorkflowTarget(prompt: string): string {
  const normalized = prompt.toLowerCase()

  if (normalized.includes('minikube')) {
    return 'Minikube'
  }

  if (normalized.includes('prod') || normalized.includes('production')) {
    return 'Production cluster'
  }

  return 'Kubernetes'
}

function detectWorkflowLifecycle(text: string, hasExecutedSnapshot: boolean): WorkflowLifecycle {
  const normalized = text.toLowerCase()

  if (hasExecutedSnapshot) {
    return 'executed'
  }

  if (/deployed|applied|rolled out|provisioned|completed|executed|ready/i.test(normalized)) {
    return 'executed'
  }

  return 'planned'
}

function buildReasons(service: DemoCatalogService) {
  const reasons = [
    `Health score: ${service.healthScore}%`,
    `Risk level: ${service.riskLevel}`,
    `Owner: ${service.owner}`,
  ]

  if (service.releaseState === 'blocked') {
    reasons.unshift('Current release state is blocked')
  }

  if (service.releaseState === 'watch') {
    reasons.unshift('Current release state needs review')
  }

  if (service.evidence.length > 0) {
    reasons.push(`Evidence: ${service.evidence.slice(0, 2).join(', ')}`)
  }

  if (service.signals.length > 0) {
    reasons.push(`Signals: ${service.signals.slice(0, 2).join(', ')}`)
  }

  return reasons
}

function buildNextSteps(service: DemoCatalogService, verdict: ReleaseVerdict) {
  if (verdict === 'block') {
    return [
      'Assign or confirm the service owner before the next release window.',
      'Generate a fresh evidence pack and close the stale control gap.',
      'Ask AI for a remediation plan that explains the fastest safe path forward.',
    ]
  }

  if (verdict === 'watch') {
    return [
      'Review the rollback plan and the latest change signals before promotion.',
      'Keep the release gated until the service returns to a stable scorecard.',
      'Use AI to summarize the approval rationale for reviewers.',
    ]
  }

  return [
    'Proceed with the standard release gate and capture the approval summary.',
    'Export the evidence bundle for audit traceability.',
    'Keep the service in the catalog watchlist for post-release verification.',
  ]
}

export function getReleaseDecision(service: DemoCatalogService): ReleaseDecision {
  const ownerless = service.owner === 'Unassigned'
  const shouldBlock =
    ownerless ||
    service.releaseState === 'blocked' ||
    service.healthScore < 78 ||
    (service.riskLevel === 'high' && service.healthScore < 88)

  const shouldWatch =
    !shouldBlock &&
    (service.releaseState === 'watch' ||
      service.riskLevel === 'medium' ||
      service.healthScore < 92 ||
      service.tier === 'critical' ||
      service.tier === 'regulated')

  const verdict: ReleaseVerdict = shouldBlock ? 'block' : shouldWatch ? 'watch' : 'go'

  const title =
    verdict === 'block'
      ? 'Release blocked'
      : verdict === 'watch'
        ? 'Review before release'
        : 'Release ready'

  const summary =
    verdict === 'block'
      ? `${service.name} is not ready for promotion until the ownership or evidence gap is resolved.`
      : verdict === 'watch'
        ? `${service.name} can move forward with review because it still carries notable release risk.`
        : `${service.name} is ready to proceed with the normal release process.`

  return {
    verdict,
    title,
    summary,
    reasons: buildReasons(service),
    nextSteps: buildNextSteps(service, verdict),
    evidence: service.evidence,
    aiPrompt: `Assess the release decision for ${service.name} and explain the blockers, evidence, and next step.`,
  }
}

function scoreService(service: DemoCatalogService) {
  let score = 0

  if (service.owner === 'Unassigned') score += 100
  if (service.releaseState === 'blocked') score += 80
  if (service.releaseState === 'watch') score += 28
  if (service.riskLevel === 'high') score += 55
  if (service.riskLevel === 'medium') score += 22
  if (service.tier === 'critical') score += 20
  if (service.tier === 'regulated') score += 16
  score += Math.max(0, 100 - service.healthScore)
  score += service.signals.some((signal) => /stale|manual|follow-up/i.test(signal)) ? 12 : 0

  return score
}

export function sortServicesByOperationalPriority(services: DemoCatalogService[]) {
  return [...services].sort((left, right) => {
    const difference = scoreService(right) - scoreService(left)
    return difference !== 0 ? difference : left.name.localeCompare(right.name)
  })
}

export function getWorkspaceSnapshot(services: DemoCatalogService[]): WorkspaceSnapshot {
  const sortedServices = sortServicesByOperationalPriority(services)

  const decisions = services.map((service) => getReleaseDecision(service))

  return {
    totalServices: services.length,
    readyServices: decisions.filter((decision) => decision.verdict === 'go').length,
    watchServices: decisions.filter((decision) => decision.verdict === 'watch').length,
    blockedServices: decisions.filter((decision) => decision.verdict === 'block').length,
    ownerlessServices: services.filter((service) => service.owner === 'Unassigned').length,
    criticalServices: services.filter((service) => service.tier === 'critical').length,
    topService: sortedServices[0] ?? services[0],
    topDecision: getReleaseDecision(sortedServices[0] ?? services[0]),
  }
}

export function buildAssistantPrompts(service?: DemoCatalogService) {
  if (!service) {
    return [
      'Summarize the top release risks across the catalog',
      'Generate a BSI C5 evidence pack for the workspace',
      'Show planned vs executed workflows for the current release queue',
      'Find owner gaps and stale controls before release',
      'Draft a GitHub and Argo CD deployment request for a new application',
      'Plan a Crossplane or Terraform infrastructure request',
    ]
  }

  return [
    `Assess the release decision for ${service.name}`,
    `Generate an evidence pack for ${service.name}`,
    `Request an Argo CD deployment for ${service.name}`,
    `Show deployment status for ${service.name}`,
    `Draft Crossplane infrastructure for ${service.name}`,
    `Draft a Terraform plan for ${service.name}`,
    `Draft the next safe action for ${service.name}`,
    `Explain why ${service.name} is ${service.releaseState}`,
  ]
}

export function resolveServiceForPrompt(
  query: string,
  services: DemoCatalogService[],
  fallbackService?: DemoCatalogService
) {
  const normalized = query.toLowerCase()

  return (
    services.find((service) => {
      const searchableTokens = [
        service.id.replace(/-/g, ' '),
        service.name,
        service.owner,
        service.team,
      ]

      return searchableTokens.some((token) => normalized.includes(token.toLowerCase()))
    }) ?? fallbackService ?? services[0]
  )
}

function classifyDeploymentTone(summary: string): {
  tone: DeploymentStatusTone
  label: string
} {
  if (/blocked|cannot|fail|error|stop/i.test(summary)) {
    return { tone: 'error', label: 'Blocked' }
  }

  if (/review|approval|guard|wait/i.test(summary)) {
    return { tone: 'degraded', label: 'Needs review' }
  }

  if (/completed|deployed|applied|rolled out/i.test(summary)) {
    return { tone: 'completed', label: 'Completed' }
  }

  if (/draft|plan|queued|request/i.test(summary)) {
    return { tone: 'running', label: 'Planned' }
  }

  return { tone: 'pending', label: 'Requested' }
}

export function buildDeploymentRequestRecord(
  prompt: string,
  service: DemoCatalogService,
  options?: {
    requestedAt?: string
    summary?: string
    responseText?: string
  }
): DeploymentRequestRecord {
  const normalized = prompt.toLowerCase()
  const workflowCategory = detectWorkflowCategory(prompt)
  const source = detectWorkflowSource(prompt)
  const delivery = detectWorkflowProvider(prompt, workflowCategory)
  const target = detectWorkflowTarget(prompt)
  const route = `${source} → ${delivery} → ${target}`
  const baseSummary =
    workflowCategory === 'infrastructure'
      ? `Infrastructure request captured for ${service.name}.`
      : `Deployment request captured for ${service.name}.`
  const summary = options?.summary ?? baseSummary
  const toneSource = options?.responseText ?? summary
  const status = classifyDeploymentTone(toneSource)
  const lifecycle = detectWorkflowLifecycle(toneSource, false)

  return {
    id: `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`,
    serviceId: service.id,
    serviceName: service.name,
    workflowCategory,
    workflowLifecycle: lifecycle,
    source,
    provider: delivery,
    route,
    delivery,
    target,
    statusTone: status.tone,
    statusLabel: status.label,
    requestedAt: options?.requestedAt ?? new Date().toISOString(),
    summary,
    timeline: [
      `Request drafted from: ${prompt}`,
      `Delivery path: ${route}`,
      lifecycle === 'executed'
        ? 'Execution record completed and captured for the operator trail'
        : 'Awaiting operator review before any rollout occurs',
    ],
    prompt,
  }
}

export function finalizeDeploymentRecordFromResponse(
  record: DeploymentRequestRecord,
  responseText: string,
  deployment?: AssistantDeploymentSnapshot | null
): DeploymentRequestRecord {
  if (!deployment) {
    return finalizeDeploymentRequestRecord(record, responseText)
  }

  const tone =
    deployment.phase === 'ready'
      ? { tone: 'completed' as const, label: 'Ready' }
      : deployment.phase === 'progressing'
        ? { tone: 'running' as const, label: 'Progressing' }
        : { tone: 'degraded' as const, label: 'Pending' }

  return {
    ...record,
    workflowLifecycle: 'executed',
    route: 'Axiom IDP → Kubernetes API → Minikube',
    delivery: normalizeWorkflowProvider(deployment.service_type || record.provider, record.workflowCategory),
    provider: normalizeWorkflowProvider(deployment.service_type || record.provider, record.workflowCategory),
    target: 'Minikube',
    statusTone: tone.tone,
    statusLabel: tone.label,
    summary:
      responseText.trim() ||
      `${deployment.name} is ${deployment.phase} in namespace ${deployment.namespace} with ${deployment.ready_replicas}/${deployment.replicas} ready replicas.`,
    timeline: [
      record.timeline[0],
      `Deployment applied: ${deployment.name} in namespace ${deployment.namespace}`,
      `Rollout phase: ${deployment.phase} with ${deployment.ready_replicas}/${deployment.replicas} ready replicas`,
    ],
  }
}

export function finalizeDeploymentRequestRecord(
  record: DeploymentRequestRecord,
  responseText: string,
  isError = false
): DeploymentRequestRecord {
  if (isError) {
    return {
      ...record,
      workflowLifecycle: 'planned',
      statusTone: 'error',
      statusLabel: 'Blocked',
      summary: `${record.serviceName} request could not be finalized right now.`,
      timeline: [...record.timeline.slice(0, 2), 'AI assistant unavailable; request remains in draft'],
    }
  }

  const status = classifyDeploymentTone(responseText)
  const lifecycle = detectWorkflowLifecycle(responseText, /completed|deployed|applied|rolled out|provisioned|ready/i.test(responseText))

  return {
    ...record,
    workflowLifecycle: lifecycle,
    statusTone: status.tone,
    statusLabel: status.label,
    summary: responseText.trim() || record.summary,
    timeline: [
      record.timeline[0],
      `AI response: ${responseText.trim() || 'No additional details returned'}`,
      lifecycle === 'executed'
        ? 'Execution record completed and captured for the operator trail'
        : status.tone === 'completed'
        ? 'Deployment handoff marked complete'
        : status.tone === 'error'
          ? 'Deployment remains blocked for review'
          : 'Deployment request remains ready for operator approval',
    ],
  }
}
