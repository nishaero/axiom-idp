import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Card, StatusBadge } from '@/components/dashboard'
import apiClient from '@/lib/api'
import {
  aiPromptSuggestions,
  buildAssistantFallbackResponse,
  demoCatalogServices,
} from '@/lib/demoContent'
import {
  buildAssistantPrompts,
  buildDeploymentRequestRecord,
  finalizeDeploymentRecordFromResponse,
  finalizeDeploymentRequestRecord,
  getWorkspaceSnapshot,
  resolveServiceForPrompt,
  type AssistantDeploymentSnapshot,
  type DeploymentRequestRecord,
} from '@/lib/operatorInsights'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
  source?: 'live' | 'local'
}

interface AssistantActionPlanItem {
  label: string
  status: string
  detail: string
}

interface AssistantActionPlanStep {
  name: string
  status: string
  detail: string
  owner: string
  why: string
}

interface AssistantActionPlan {
  title: string
  intent: string
  mode: string
  summary: string
  confidence: string
  execution_path: string
  focus_service?: {
    id: string
    name: string
    status: string
    release_state: string
    risk_level: string
    health_score: number
  } | null
  steps: AssistantActionPlanStep[]
  guardrails: AssistantActionPlanItem[]
  evidence: AssistantActionPlanItem[]
  observability: AssistantActionPlanItem[]
  approvals: AssistantActionPlanItem[]
  outcome_preview: AssistantActionPlanItem[]
}

const initialMessage: Message = {
  id: 'assistant-welcome',
  role: 'assistant',
  content:
    'I can help with release decisions, release briefs, ownership gaps, deployment requests, evidence packs, Argo CD rollouts, and Crossplane or Terraform infrastructure requests. Ask for a service-by-service assessment, a BSI C5 summary, or a safe next step.',
  timestamp: new Date(),
  source: 'local',
}

const deploymentHistoryStorageKey = 'axiom.deployment-history'

function loadDeploymentHistory(): DeploymentRequestRecord[] {
  if (typeof window === 'undefined') return []

  try {
    const stored = window.localStorage.getItem(deploymentHistoryStorageKey)
    if (!stored) return []

    const parsed = JSON.parse(stored) as DeploymentRequestRecord[]
    return Array.isArray(parsed) ? parsed.slice(0, 5) : []
  } catch {
    return []
  }
}

export default function AIAssistant() {
  const [searchParams] = useSearchParams()
  const [messages, setMessages] = useState<Message[]>([initialMessage])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [deploymentHistory, setDeploymentHistory] = useState<DeploymentRequestRecord[]>(() => loadDeploymentHistory())
  const [latestActionPlan, setLatestActionPlan] = useState<AssistantActionPlan | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const autoPromptRef = useRef<string | null>(null)
  const initialPrompt = searchParams.get('prompt')?.trim() ?? ''
  const autoStart = searchParams.get('autostart') === '1'

  const contextSummary = useMemo(() => {
    const workspaceSnapshot = getWorkspaceSnapshot(demoCatalogServices)

    return {
      serviceCount: workspaceSnapshot.totalServices,
      readyServices: workspaceSnapshot.readyServices,
      watchServices: workspaceSnapshot.watchServices,
      blockedServices: workspaceSnapshot.blockedServices,
      ownerlessServices: workspaceSnapshot.ownerlessServices,
      topService: workspaceSnapshot.topService,
      topDecision: workspaceSnapshot.topDecision,
    }
  }, [])

  const promptSuggestions = useMemo(
    () => [...buildAssistantPrompts(contextSummary.topService), ...aiPromptSuggestions],
    [contextSummary.topService]
  )
  const plannedWorkflows = useMemo(
    () => deploymentHistory.filter((record) => record.workflowLifecycle === 'planned'),
    [deploymentHistory]
  )
  const executedWorkflows = useMemo(
    () => deploymentHistory.filter((record) => record.workflowLifecycle === 'executed'),
    [deploymentHistory]
  )
  const activeWorkflow = executedWorkflows[0] ?? plannedWorkflows[0] ?? null
  const deploymentSummary = useMemo(
    () => ({
      planned: plannedWorkflows.length,
      executed: executedWorkflows.length,
      infra: deploymentHistory.filter((record) => record.workflowCategory === 'infrastructure').length,
    }),
    [deploymentHistory, executedWorkflows.length, plannedWorkflows.length]
  )

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    try {
      window.localStorage.setItem(deploymentHistoryStorageKey, JSON.stringify(deploymentHistory))
    } catch {
      // Ignore storage write failures.
    }
  }, [deploymentHistory])

  useEffect(() => {
    if (initialPrompt) {
      setInput(initialPrompt)
    }
  }, [initialPrompt])

  const handleSendMessage = useCallback(
    async (overrideInput?: string) => {
      const prompt = (overrideInput ?? input).trim()
      if (!prompt || loading) return
      const requestService = resolveServiceForPrompt(prompt, demoCatalogServices, contextSummary.topService)
      const deploymentRequest = /deploy|deployment|rollout|status|helm|argocd|argo cd|crossplane|terraform|infra|infrastructure|provision/i.test(
        prompt
      )
        ? buildDeploymentRequestRecord(prompt, requestService)
        : null

      const userMessage: Message = {
        id: `${Date.now()}-user`,
        role: 'user',
        content: prompt,
        timestamp: new Date(),
      }

      setMessages((previous) => [...previous, userMessage])
      setInput('')
      setLoading(true)
      if (deploymentRequest) {
        setDeploymentHistory((previous) => [deploymentRequest, ...previous].slice(0, 5))
      }

      try {
        const response = await apiClient.post('/ai/query', {
          query: prompt,
          context_limit: 2000,
          context: {
            services: contextSummary.serviceCount,
            readyServices: contextSummary.readyServices,
            ownerlessServices: contextSummary.ownerlessServices,
          },
        })

        const responseData = response.data as {
          response?: string
          answer?: string
          intent?: string
          deployment?: AssistantDeploymentSnapshot
          action_plan?: AssistantActionPlan
        }

        const assistantText =
          (responseData.response ??
            responseData.answer ??
            '').trim() || buildAssistantFallbackResponse(prompt)

        const assistantMessage: Message = {
          id: `${Date.now()}-assistant`,
          role: 'assistant',
          content: /coming soon/i.test(assistantText)
            ? buildAssistantFallbackResponse(prompt, demoCatalogServices)
            : assistantText,
          timestamp: new Date(),
          source: 'live',
        }

        setMessages((previous) => [...previous, assistantMessage])
        setLatestActionPlan(responseData.action_plan ?? null)
        if (deploymentRequest) {
          const deploymentResponse = assistantMessage.content
          setDeploymentHistory((previous) =>
            previous.map((entry) =>
              entry.id === deploymentRequest.id
                ? finalizeDeploymentRecordFromResponse(
                    entry,
                    deploymentResponse,
                    responseData.intent?.startsWith('deployment_') ? responseData.deployment ?? null : null
                  )
                : entry
            )
          )
        }
      } catch (error) {
        console.error('Failed to get AI response:', error)
        setMessages((previous) => [
          ...previous,
          {
            id: `${Date.now()}-assistant`,
            role: 'assistant',
            content: buildAssistantFallbackResponse(prompt, demoCatalogServices),
            timestamp: new Date(),
            source: 'local',
          },
        ])
        setLatestActionPlan(null)
        if (deploymentRequest) {
          setDeploymentHistory((previous) =>
            previous.map((entry) =>
              entry.id === deploymentRequest.id
                ? finalizeDeploymentRequestRecord(entry, prompt, true)
                : entry
            )
          )
        }
      } finally {
        setLoading(false)
        textareaRef.current?.focus()
      }
    },
    [
      contextSummary.ownerlessServices,
      contextSummary.readyServices,
      contextSummary.serviceCount,
      contextSummary.topService,
      input,
      loading,
    ]
  )

  useEffect(() => {
    if (!autoStart || !initialPrompt || autoPromptRef.current === initialPrompt) return

    autoPromptRef.current = initialPrompt
    void handleSendMessage(initialPrompt)
  }, [autoStart, handleSendMessage, initialPrompt])

  const formatRelativeTime = (timestamp: string) => {
    const parsed = new Date(timestamp).getTime()
    if (Number.isNaN(parsed)) return 'just now'

    const diff = Date.now() - parsed
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.max(1, Math.floor(diff / 60_000))}m ago`
    if (diff < 86_400_000) return `${Math.max(1, Math.floor(diff / 3_600_000))}h ago`
    return `${Math.max(1, Math.floor(diff / 86_400_000))}d ago`
  }

  const getWorkflowLifecycleBadge = (lifecycle: 'planned' | 'executed') =>
    lifecycle === 'executed' ? (
      <StatusBadge status="completed" label="Executed" size="sm" />
    ) : (
      <StatusBadge status="pending" label="Planned" size="sm" />
    )

  const renderPlanItemTone = (status?: string) => {
    if (status === 'ready' || status === 'active') return 'border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950/30'
    if (status === 'attention') return 'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30'
    return 'border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950/30'
  }

  return (
    <div className="grid gap-6 p-6 md:p-8 xl:grid-cols-[1.15fr_0.85fr]">
      <section className="flex min-h-[72vh] flex-col rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div className="flex items-center justify-between border-b border-gray-200 px-6 py-5 dark:border-dark-700">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">AI Assistant</h1>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Ask for release guidance, compliance evidence, ownership remediation, or a rollout summary.
            </p>
          </div>
          <StatusBadge status="running" label={autoStart ? 'Autoprompt ready' : 'Operator aware'} />
        </div>

        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          {messages.map((message) => (
            <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-2xl rounded-2xl px-4 py-3 shadow-sm ${
                  message.role === 'user'
                    ? 'bg-primary-600 text-white'
                    : 'border border-gray-200 bg-gray-50 text-gray-900 dark:border-dark-700 dark:bg-dark-700 dark:text-white'
                }`}
              >
                <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.2em] opacity-70">
                  <span>{message.role === 'user' ? 'You' : 'Axiom'}</span>
                  <span>·</span>
                  <span>{message.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                  {message.source && (
                    <>
                      <span>·</span>
                      <span>{message.source === 'live' ? 'Live' : 'Local'}</span>
                    </>
                  )}
                </div>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{message.content}</p>
              </div>
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>

        <div className="border-t border-gray-200 p-5 dark:border-dark-700">
          <div className="flex flex-wrap gap-2">
            {promptSuggestions.map((suggestion) => (
              <button
                key={suggestion}
                type="button"
                onClick={() => void handleSendMessage(suggestion)}
                className="rounded-full border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300"
              >
                {suggestion}
              </button>
            ))}
          </div>
          <div className="mt-4 flex flex-col gap-3 sm:flex-row">
            <textarea
              ref={textareaRef}
              rows={3}
              placeholder="Ask me to assess risk, generate evidence, or draft an Argo CD, Crossplane, or Terraform workflow..."
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  void handleSendMessage()
                }
              }}
              disabled={loading}
              className="min-h-[84px] flex-1 rounded-2xl border border-gray-300 bg-white px-4 py-3 text-sm text-gray-900 shadow-sm outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-700 dark:bg-dark-900 dark:text-white"
            />
            <button
              type="button"
              onClick={() => void handleSendMessage()}
              disabled={loading || !input.trim()}
              className="inline-flex items-center justify-center rounded-2xl bg-primary-600 px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? 'Thinking...' : 'Send'}
            </button>
          </div>
        </div>
      </section>

      <aside className="space-y-4">
        {latestActionPlan ? (
          <Card title={latestActionPlan.title} subtitle="AI-first interaction, deterministic execution">
            <div className="space-y-4 text-sm text-gray-600 dark:text-gray-400">
              <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                      {latestActionPlan.mode} · {latestActionPlan.intent.split('_').join(' ')}
                    </p>
                    <p className="mt-2 text-sm leading-6">{latestActionPlan.summary}</p>
                  </div>
                  <StatusBadge status="running" label={`${latestActionPlan.confidence} confidence`} size="sm" />
                </div>
                <p className="mt-3 text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  {latestActionPlan.execution_path}
                </p>
                {latestActionPlan.focus_service ? (
                  <div className="mt-4 grid gap-2 text-xs sm:grid-cols-2">
                    <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      Focus: {latestActionPlan.focus_service.name}
                    </span>
                    <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      Risk: {latestActionPlan.focus_service.risk_level}
                    </span>
                    <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      State: {latestActionPlan.focus_service.release_state}
                    </span>
                    <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                      Health: {latestActionPlan.focus_service.health_score}
                    </span>
                  </div>
                ) : null}
              </div>

              <div>
                <h3 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Execution steps</h3>
                <div className="mt-3 space-y-3">
                  {latestActionPlan.steps.map((step) => (
                    <div key={`${step.name}-${step.owner}`} className={`rounded-2xl border p-4 ${renderPlanItemTone(step.status)}`}>
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div>
                          <p className="font-medium text-gray-900 dark:text-white">{step.name}</p>
                          <p className="mt-1 text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                            Owner: {step.owner}
                          </p>
                        </div>
                        <StatusBadge status={step.status === 'attention' ? 'unhealthy' : step.status === 'ready' ? 'healthy' : 'degraded'} label={step.status} size="sm" />
                      </div>
                      <p className="mt-3 leading-6">{step.detail}</p>
                      <p className="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{step.why}</p>
                    </div>
                  ))}
                </div>
              </div>

              <div className="grid gap-4 xl:grid-cols-2">
                <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                  <h3 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Guardrails</h3>
                  <div className="mt-3 space-y-3">
                    {latestActionPlan.guardrails.map((item) => (
                      <div key={item.label} className={`rounded-2xl border p-3 ${renderPlanItemTone(item.status)}`}>
                        <p className="font-medium text-gray-900 dark:text-white">{item.label}</p>
                        <p className="mt-2 leading-6">{item.detail}</p>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                  <h3 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Evidence and approvals</h3>
                  <div className="mt-3 space-y-3">
                    {[...latestActionPlan.evidence, ...latestActionPlan.approvals].map((item) => (
                      <div key={`${item.label}-${item.detail}`} className={`rounded-2xl border p-3 ${renderPlanItemTone(item.status)}`}>
                        <p className="font-medium text-gray-900 dark:text-white">{item.label}</p>
                        <p className="mt-2 leading-6">{item.detail}</p>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <h3 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Observability and manual fallback</h3>
                <div className="mt-3 space-y-3">
                  {latestActionPlan.observability.map((item) => (
                    <div key={item.label} className={`rounded-2xl border p-3 ${renderPlanItemTone(item.status)}`}>
                      <p className="font-medium text-gray-900 dark:text-white">{item.label}</p>
                      <p className="mt-2 leading-6">{item.detail}</p>
                    </div>
                  ))}
                  <div className="rounded-2xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700/40">
                    <p className="font-medium text-gray-900 dark:text-white">Manual controls stay available</p>
                    <p className="mt-2 leading-6">
                      Developers can start with AI, then verify or continue manually in the catalog, dashboard, and settings screens without losing the execution trail.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        ) : null}

        <Card title="Assistant context" subtitle="What the assistant can reason over">
          <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
            <div className="flex items-center justify-between">
              <span>Services indexed</span>
              <span className="font-semibold text-gray-900 dark:text-white">{contextSummary.serviceCount}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Ready for release</span>
              <span className="font-semibold text-green-600 dark:text-green-400">{contextSummary.readyServices}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Needs review</span>
              <span className="font-semibold text-amber-600 dark:text-amber-400">{contextSummary.watchServices}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Blocked</span>
              <span className="font-semibold text-red-600 dark:text-red-400">{contextSummary.blockedServices}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Owner gaps</span>
              <span className="font-semibold text-amber-600 dark:text-amber-400">{contextSummary.ownerlessServices}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Planned workflows</span>
              <span className="font-semibold text-blue-600 dark:text-blue-400">{deploymentSummary.planned}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Executed workflows</span>
              <span className="font-semibold text-green-600 dark:text-green-400">{deploymentSummary.executed}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Infra requests</span>
              <span className="font-semibold text-primary-600 dark:text-primary-400">{deploymentSummary.infra}</span>
            </div>
          </div>
        </Card>

        <Card
          title="Service focus"
          subtitle={`${contextSummary.topService.name} is the highest-priority item in the workspace`}
        >
          <div className="space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
            <StatusBadge
              status={
                contextSummary.topDecision.verdict === 'go'
                  ? 'healthy'
                  : contextSummary.topDecision.verdict === 'watch'
                    ? 'degraded'
                    : 'unhealthy'
              }
              label={contextSummary.topDecision.title}
            />
            <p>{contextSummary.topDecision.summary}</p>
            <p>
              Owner: <span className="font-semibold text-gray-900 dark:text-white">{contextSummary.topService.owner}</span>
            </p>
            <div className="flex flex-wrap gap-2">
              {contextSummary.topService.aiBenefits.slice(0, 3).map((benefit) => (
                <span
                  key={benefit}
                  className="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:border-primary-900 dark:bg-primary-900/30 dark:text-primary-300"
                >
                  {benefit}
                </span>
              ))}
            </div>
          </div>
        </Card>

        <Card title="What AI does here" subtitle="It shortens decisions, not controls">
          <ul className="space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
            <li>Explains release decisions from catalog health, ownership, and evidence state.</li>
            <li>Turns a service drilldown into a ready-to-send rollout or remediation prompt.</li>
            <li>Summarizes evidence packs for approvals, audits, and change reviews.</li>
          </ul>
        </Card>

        <Card
          title="Workflow delivery trail"
          subtitle="Planned requests stay separate from executed rollouts and infra changes"
        >
          {activeWorkflow ? (
            <div className="space-y-5 text-sm text-gray-600 dark:text-gray-400">
              <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">{activeWorkflow.serviceName}</p>
                    <p className="mt-1 text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                      {activeWorkflow.route}
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-2">
                    {getWorkflowLifecycleBadge(activeWorkflow.workflowLifecycle)}
                    <StatusBadge status={activeWorkflow.statusTone} label={activeWorkflow.statusLabel} size="sm" />
                  </div>
                </div>
                <p className="mt-3 leading-6">{activeWorkflow.summary}</p>
                <div className="mt-4 grid gap-2 text-xs sm:grid-cols-2 xl:grid-cols-4">
                  <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                    {activeWorkflow.workflowCategory === 'infrastructure' ? 'Infrastructure workflow' : 'Application workflow'}
                  </span>
                  <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                    Source: {activeWorkflow.source}
                  </span>
                  <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                    Provider: {activeWorkflow.provider}
                  </span>
                  <span className="rounded-full bg-white px-3 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-300">
                    Target: {activeWorkflow.target}
                  </span>
                </div>
                <p className="mt-3 text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  {formatRelativeTime(activeWorkflow.requestedAt)}
                </p>
              </div>

                <div className="grid gap-4 xl:grid-cols-2">
                  <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                    <div className="flex items-center justify-between">
                      <div>
                      <h4 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                        Executed workflows
                      </h4>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                        Rollouts and changes already applied by the assistant
                      </p>
                    </div>
                    <StatusBadge status="completed" label={`${executedWorkflows.length} executed`} size="sm" />
                  </div>
                  <div className="mt-4 space-y-3">
                    {executedWorkflows.length > 0 ? (
                      executedWorkflows.map((request) => (
                        <div key={request.id} className="rounded-2xl border border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950/30">
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <div>
                              <p className="font-medium text-gray-900 dark:text-white">{request.serviceName}</p>
                              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{request.route}</p>
                            </div>
                            <div className="flex flex-col items-end gap-2">
                              {getWorkflowLifecycleBadge(request.workflowLifecycle)}
                              <StatusBadge status={request.statusTone} label={request.statusLabel} size="sm" />
                            </div>
                          </div>
                          <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                            {request.workflowCategory === 'infrastructure' ? 'Infrastructure' : 'Application'} · {request.source} · {request.provider} · {request.target}
                          </p>
                          <p className="mt-2 leading-6 text-gray-600 dark:text-gray-400">{request.summary}</p>
                        </div>
                      ))
                    ) : (
                      <p className="text-sm leading-6 text-gray-600 dark:text-gray-400">
                        No executed workflows yet. Run an AI-triggered deployment and the completed trail will appear here.
                      </p>
                    )}
                  </div>
                </div>

                <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                        Planned workflows
                      </h4>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                        Requests drafted by AI and waiting for operator approval
                      </p>
                    </div>
                    <StatusBadge status="pending" label={`${plannedWorkflows.length} planned`} size="sm" />
                  </div>
                  <div className="mt-4 space-y-3">
                    {plannedWorkflows.length > 0 ? (
                      plannedWorkflows.map((request) => (
                        <div key={request.id} className="rounded-2xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-900 dark:bg-blue-950/30">
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <div>
                              <p className="font-medium text-gray-900 dark:text-white">{request.serviceName}</p>
                              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{request.route}</p>
                            </div>
                            <div className="flex flex-col items-end gap-2">
                              {getWorkflowLifecycleBadge(request.workflowLifecycle)}
                              <StatusBadge status={request.statusTone} label={request.statusLabel} size="sm" />
                            </div>
                          </div>
                          <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                            {request.workflowCategory === 'infrastructure' ? 'Infrastructure' : 'Application'} · {request.source} · {request.provider} · {request.target}
                          </p>
                          <p className="mt-2 leading-6 text-gray-600 dark:text-gray-400">{request.summary}</p>
                        </div>
                      ))
                    ) : (
                      <p className="text-sm leading-6 text-gray-600 dark:text-gray-400">
                        No planned workflows yet. Ask for an Argo CD deployment or a Crossplane/Terraform request to create one.
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <p className="text-sm leading-6 text-gray-600 dark:text-gray-400">
              Ask the assistant to draft an Argo CD deployment or a Crossplane/Terraform infrastructure request and the trail will appear here with route, lifecycle, and rollout history.
            </p>
          )}
        </Card>

        <Card
          title="Connected views"
          subtitle="Jump between the control plane screens"
          actions={
            <Link
              to="/catalog"
              className="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Open catalog
            </Link>
          }
        >
          <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
            <p>
              Use the catalog to inspect release posture, then come back here for a policy-aware
              narrative and next-step recommendations.
            </p>
            <Link
              to="/settings"
              className="inline-flex rounded-lg border border-gray-300 px-3 py-2 font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-gray-300"
            >
              Review compliance settings
            </Link>
          </div>
        </Card>
      </aside>
    </div>
  )
}
