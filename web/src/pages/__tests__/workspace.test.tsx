import { render, screen, waitFor } from '@testing-library/react'
import { fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'
import apiClient from '@/lib/api'
import Dashboard from '@/pages/Dashboard'
import Observability from '@/pages/Observability'
import Catalog from '@/pages/Catalog'
import AIAssistant from '@/pages/AIAssistant'
import Settings from '@/pages/Settings'

vi.mock('@/lib/api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

function renderWithProviders(element: ReactElement, initialEntries?: string[]) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={initialEntries}>{element}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('workspace pages', () => {
  beforeEach(() => {
    vi.mocked(apiClient.get).mockReset()
    vi.mocked(apiClient.post).mockReset()
    vi.mocked(apiClient.get).mockRejectedValue(new Error('offline'))
  })

  it('renders the dashboard as an operational overview', () => {
    renderWithProviders(<Dashboard />)

    expect(
      screen.getByRole('heading', { name: /release safely with ai-guided operations/i })
    ).toBeInTheDocument()
    expect(screen.getByText(/AI change-risk forecast/i)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /release queue/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 3, name: /recent activity/i })).toBeInTheDocument()
  })

  it('renders the observability control tower from backend data', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: {
        platform: {
          status: 'ready',
          environment: 'test',
          started_at: '2026-04-15T08:00:00Z',
          uptime: '1h',
          ai_backend: 'local',
          kubernetes_namespace: 'axiom-apps',
          checks: [
            { name: 'auth', status: 'ready', message: 'Signed session tokens and RBAC middleware are active.' },
          ],
          alerts: [],
          overview: {
            total_services: 4,
            ready_services: 3,
            watch_services: 1,
            blocked_services: 0,
            owner_gap_count: 0,
            release_readiness: 92,
            evidence_coverage: 88,
          },
          services: [],
          audit: {
            entries: 12,
            last_entry_at: '2026-04-15T08:10:00Z',
            error_count: 1,
            denied_count: 0,
            success_count: 11,
          },
          rate_limiting: {
            enabled: true,
            tracked_keys: 2,
            requests_seen: 41,
            last_cleanup_at: '2026-04-15T08:09:00Z',
            requests_per_min: 1000,
          },
          observability_notes: ['Dashboard values marked live come from the backend platform status endpoint.'],
        },
        telemetry: {
          http_requests_total: 32,
          http_error_responses_total: 1,
          http_rate_limited_total: 0,
          ai_requests_total: 8,
          ai_failures_total: 1,
          deployment_requests_total: 4,
          audit_events_total: 12,
          last_request_at: '2026-04-15T08:11:00Z',
          last_ai_request_at: '2026-04-15T08:10:00Z',
          last_deployment_at: '2026-04-15T08:08:00Z',
          last_audit_at: '2026-04-15T08:11:00Z',
        },
        endpoints: [
          { name: 'Live probe', path: '/live', status: 'healthy', description: 'Container liveness probe for Kubernetes and Docker entrypoints.' },
        ],
        metrics_endpoint: '/metrics',
        prometheus_annotations: ['prometheus.io/scrape=true'],
        notes: ['Prometheus scrapes the /metrics endpoint directly.'],
      },
    })

    renderWithProviders(<Observability />)

    expect(screen.getByText(/observability control tower/i)).toBeInTheDocument()
    expect(screen.getByText(/prometheus metrics/i)).toBeInTheDocument()
    expect(screen.getByText(/http requests/i)).toBeInTheDocument()
    expect(
      await screen.findByText(/signed session tokens and rbac middleware are active/i)
    ).toBeInTheDocument()
  })

  it('renders a release brief in the catalog drilldown', () => {
    renderWithProviders(<Catalog />)

    expect(screen.getByRole('heading', { level: 2, name: /data bridge/i })).toBeInTheDocument()
    expect(screen.getByText(/^release brief$/i)).toBeInTheDocument()
    expect(screen.getByText(/next best action/i)).toBeInTheDocument()
  })

  it('filters catalog services by search term', async () => {
    renderWithProviders(<Catalog />)

    expect(screen.getByRole('heading', { level: 2, name: /data bridge/i })).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/search services/i), { target: { value: 'payment' } })

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 2, name: /payment api/i })).toBeInTheDocument()
    })

    expect(screen.queryByRole('heading', { level: 2, name: /auth gateway/i })).not.toBeInTheDocument()
  })

  it('autostarts a prompted ai workflow from the url', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { response: 'Payment API should stay on watch until the rollback plan is reviewed.' },
    })

    renderWithProviders(
      <AIAssistant />,
      ['/ai?prompt=Assess%20the%20release%20decision%20for%20Payment%20API&autostart=1']
    )

    expect(
      await screen.findByText(/payment api should stay on watch until the rollback plan is reviewed/i)
    ).toBeInTheDocument()
    expect(apiClient.post).toHaveBeenCalledWith(
      '/ai/query',
      expect.objectContaining({
        query: expect.stringContaining('Payment API'),
      })
    )
  })

  it('falls back to a local response when the ai api is unavailable', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('offline'))

    renderWithProviders(<AIAssistant />)

    fireEvent.click(screen.getByRole('button', { name: /generate a bsi c5 evidence pack for audit ledger/i }))

    expect(await screen.findByText(/i assembled the compliance view/i)).toBeInTheDocument()
  })

  it('exposes a release brief prompt in the assistant', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { response: 'I prepared a release brief for Audit Ledger.' },
    })

    renderWithProviders(<AIAssistant />)

    fireEvent.click(screen.getByRole('button', { name: /generate a release brief for audit ledger/i }))

    expect(await screen.findByText(/i prepared a release brief/i)).toBeInTheDocument()
  })

  it('captures argo cd deployment requests in the assistant trail as planned workflows', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: {
        response: 'Argo CD deployment request queued for review.',
        action_plan: {
          title: 'AI-guided GitOps rollout',
          intent: 'deployment_apply_argocd',
          mode: 'delivery',
          summary: 'Payment API is routed through GitHub and Argo CD so the rollout stays reviewable and observable.',
          confidence: 'high',
          execution_path: 'Axiom intent -> GitHub branch -> Argo CD application -> Kubernetes rollout',
          steps: [
            {
              name: 'Intent normalized',
              status: 'ready',
              detail: 'Deployment request was translated into a GitOps-safe rollout specification.',
              owner: 'Axiom',
              why: 'Removes manual YAML authoring while keeping rollout semantics explicit.',
            },
          ],
          guardrails: [{ label: 'GitOps provenance', status: 'active', detail: 'Branch-backed delivery artifacts are created.' }],
          evidence: [{ label: 'Service evidence', status: 'ready', detail: 'Evidence pack is attached.' }],
          observability: [{ label: 'Operational trail', status: 'active', detail: 'Requests and rollouts are visible.' }],
          approvals: [{ label: 'Human-in-the-loop release control', status: 'recommended', detail: 'High-risk changes remain approval gated.' }],
          outcome_preview: [{ label: 'Delivery result', status: 'ready', detail: 'Developers can request and verify from one workspace.' }],
        },
      },
    })

    renderWithProviders(
      <AIAssistant />,
      ['/ai?prompt=Deploy%20Payment%20API%20to%20Minikube%20via%20GitHub%20and%20Argo%20CD&autostart=1']
    )

    expect(await screen.findByRole('heading', { name: /workflow delivery trail/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /ai-guided gitops rollout/i })).toBeInTheDocument()
    expect(screen.getByText(/manual controls stay available/i)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /planned workflows/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /executed workflows/i })).toBeInTheDocument()
    expect(screen.getAllByText(/github → argo cd → minikube/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/provider: argo cd/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/planned/i).length).toBeGreaterThan(0)
  })

  it('captures terraform infrastructure requests in the assistant trail', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { response: 'Terraform infrastructure request queued for approval.' },
    })

    renderWithProviders(
      <AIAssistant />,
      ['/ai?prompt=Provision%20Audit%20Ledger%20infrastructure%20with%20Terraform%20on%20Minikube&autostart=1']
    )

    expect(await screen.findByRole('heading', { name: /workflow delivery trail/i })).toBeInTheDocument()
    expect(screen.getAllByText(/infrastructure workflow/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/provider: terraform/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/axiom idp → terraform → minikube/i).length).toBeGreaterThan(0)
  })

  it('explains the ai operating model in settings', () => {
    renderWithProviders(<Settings />)

    expect(
      screen.getByRole('heading', { name: /control access, evidence, ai routing, and registry trust from one place/i })
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /where ai is used/i })).toBeInTheDocument()
  })
})
