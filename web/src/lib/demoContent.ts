export interface DemoCatalogService {
  id: string
  name: string
  description: string
  owner: string
  team: string
  tier: 'critical' | 'important' | 'standard' | 'regulated'
  releaseState: 'ready' | 'watch' | 'blocked'
  healthScore: number
  riskLevel: 'low' | 'medium' | 'high'
  monthlyCost: string
  tags: string[]
  signals: string[]
  evidence: string[]
  aiBenefits: string[]
  lastDeployed: string
}

export interface DemoInsight {
  title: string
  description: string
  value: string
}

export const demoCatalogServices: DemoCatalogService[] = [
  {
    id: 'auth-gateway',
    name: 'Auth Gateway',
    description: 'Front-door identity layer for SSO, SCIM, and policy-enforced access.',
    owner: 'Platform Identity',
    team: 'Security Engineering',
    tier: 'critical',
    releaseState: 'ready',
    healthScore: 96,
    riskLevel: 'low',
    monthlyCost: '$4.2k',
    tags: ['OIDC', 'SCIM', 'Kubernetes', 'BSI C5'],
    signals: ['golden path', 'multi-region', 'audit logging'],
    evidence: ['MFA enforced', 'log retention 180d', 'SAST + DAST green'],
    aiBenefits: ['blast-radius forecast', 'policy replay', 'release summarizer'],
    lastDeployed: '2 hours ago',
  },
  {
    id: 'developer-portal',
    name: 'Developer Portal',
    description: 'Self-service entry point for catalog, templates, and approvals.',
    owner: 'Developer Experience',
    team: 'Platform',
    tier: 'important',
    releaseState: 'ready',
    healthScore: 94,
    riskLevel: 'low',
    monthlyCost: '$2.9k',
    tags: ['Backstage-style catalog', 'templates', 'workflow'],
    signals: ['high adoption', 'low support load', 'AI assistant ready'],
    evidence: ['ownership scorecard', 'approval trail', 'API schema checks'],
    aiBenefits: ['search-by-intent', 'change summarization', 'release advisor'],
    lastDeployed: '1 day ago',
  },
  {
    id: 'payment-api',
    name: 'Payment API',
    description: 'Revenue-critical API with guarded rollout and stricter release controls.',
    owner: 'Revenue Platform',
    team: 'Commerce',
    tier: 'critical',
    releaseState: 'watch',
    healthScore: 87,
    riskLevel: 'medium',
    monthlyCost: '$8.1k',
    tags: ['PCI', 'canary', 'rate-limit'],
    signals: ['higher change rate', 'manual approval required', 'feature flags'],
    evidence: ['4/5 controls green', 'rollback plan current', 'owner on-call'],
    aiBenefits: ['approval recommendation', 'risk heatmap', 'evidence pack'],
    lastDeployed: '6 hours ago',
  },
  {
    id: 'audit-ledger',
    name: 'Audit Ledger',
    description: 'Append-only evidence store for compliance events and deployment traces.',
    owner: 'Security Engineering',
    team: 'Governance',
    tier: 'regulated',
    releaseState: 'ready',
    healthScore: 99,
    riskLevel: 'low',
    monthlyCost: '$3.4k',
    tags: ['immutable', 'evidence', 'retention'],
    signals: ['audit-grade logs', 'exportable controls', 'tamper detection'],
    evidence: ['WORM policy', 'retention locked', 'integrity checks green'],
    aiBenefits: ['control mapping', 'audit packet export', 'queryable lineage'],
    lastDeployed: '3 days ago',
  },
  {
    id: 'notification-mesh',
    name: 'Notification Mesh',
    description: 'Multi-channel event delivery for email, Slack, and webhook notifications.',
    owner: 'Communications Platform',
    team: 'Developer Experience',
    tier: 'standard',
    releaseState: 'ready',
    healthScore: 92,
    riskLevel: 'low',
    monthlyCost: '$1.8k',
    tags: ['eventing', 'webhooks', 'multi-channel'],
    signals: ['low latency', 'high fan-out', 'policy templates'],
    evidence: ['delivery retries green', 'dead-letter queue empty', 'ownership current'],
    aiBenefits: ['incident summarizer', 'template suggestions', 'consumer impact check'],
    lastDeployed: '8 hours ago',
  },
  {
    id: 'data-bridge',
    name: 'Data Bridge',
    description: 'Analytics export path with stricter owner and evidence requirements.',
    owner: 'Unassigned',
    team: 'Data Platform',
    tier: 'standard',
    releaseState: 'blocked',
    healthScore: 71,
    riskLevel: 'high',
    monthlyCost: '$5.0k',
    tags: ['analytics', 'exports', 'needs-owner'],
    signals: ['missing owner', 'stale evidence', 'manual follow-up required'],
    evidence: ['schema checks partial', 'SLO missing', 'security review pending'],
    aiBenefits: ['owner detection', 'remediation plan', 'compliance gap summary'],
    lastDeployed: '5 days ago',
  },
]

export const demoInsights: DemoInsight[] = [
  {
    title: 'AI change-risk forecast',
    description: 'Predicts rollout blast radius from live health, ownership, and recent activity signals.',
    value: '94% precision',
  },
  {
    title: 'Compliance evidence pack',
    description: 'Generates BSI C5-ready audit trails from deployments, approvals, and control status.',
    value: '12 packs ready',
  },
  {
    title: 'Ownership drift detector',
    description: 'Finds services with stale owners or missing evidence before they become incidents.',
    value: '1 service blocked',
  },
]

export const aiPromptSuggestions = [
  'Assess the release risk for Payment API this week',
  'Generate a BSI C5 evidence pack for the catalog',
  'Find services missing an owner or stale controls',
  'Summarize the last 24 hours of changes',
]

export function buildAssistantFallbackResponse(
  query: string,
  services: DemoCatalogService[] = demoCatalogServices
): string {
  const normalized = query.toLowerCase()
  const selectedService =
    services.find(
      (service) =>
        normalized.includes(service.id.replace(/-/g, ' ')) ||
        normalized.includes(service.name.toLowerCase()) ||
        service.name
          .toLowerCase()
          .split(' ')
          .some((segment) => segment.length > 4 && normalized.includes(segment))
    ) ?? services[0]

  const readyServices = services.filter((service) => service.releaseState === 'ready').length
  const blockedServices = services.filter((service) => service.releaseState === 'blocked').length
  const criticalServices = services.filter((service) => service.tier === 'critical').length
  const ownerlessServices = services.filter((service) => service.owner === 'Unassigned').length

  if (normalized.includes('compliance') || normalized.includes('c5') || normalized.includes('audit')) {
    return [
      `I assembled the compliance view for ${selectedService.name}.`,
      `The catalog snapshot has ${readyServices} ready services, ${blockedServices} blocked services, and ${ownerlessServices} ownership gaps.`,
      `For BSI C5 evidence, I would export deployment approvals, control ownership, release notes, and the audit trail from ${selectedService.name}.`,
      `The highest-value follow-up is to close the ownership gap on ${services.find((service) => service.owner === 'Unassigned')?.name ?? 'the blocked service'} before the next release window.`,
    ].join('\n\n')
  }

  if (normalized.includes('risk') || normalized.includes('blast') || normalized.includes('rollback')) {
    return [
      `I assessed ${selectedService.name} as the primary risk focus.`,
      `It is marked ${selectedService.releaseState}, with a ${selectedService.riskLevel} risk profile and ${selectedService.healthScore}% health score.`,
      `Recommended action: keep the rollout gated, require a rollback plan, and review the ${selectedService.signals.slice(0, 2).join(' and ')} signals before promotion.`,
    ].join('\n\n')
  }

  if (normalized.includes('owner') || normalized.includes('drift') || normalized.includes('stale')) {
    const ownerless = services.filter((service) => service.owner === 'Unassigned')
    return [
      `I found ${ownerless.length} service(s) without a stable owner.`,
      ownerless.length > 0
        ? `The main gap is ${ownerless.map((service) => service.name).join(', ')}.`
        : 'No service is currently missing an owner.',
      `Suggested next step: open a remediation task, attach the control owner, and generate a fresh evidence pack before the next change.`,
    ].join('\n\n')
  }

  return [
    `I reviewed ${services.length} services and focused on ${selectedService.name}.`,
    `The current catalog snapshot shows ${readyServices} ready services and ${criticalServices} critical services that need stronger release discipline.`,
    `Useful follow-up actions: generate a scorecard, review evidence, or ask me to draft a rollout plan for ${selectedService.name}.`,
  ].join('\n\n')
}
