import { render, screen, waitFor } from '@testing-library/react'
import { fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'
import apiClient from '@/lib/api'
import Dashboard from '@/pages/Dashboard'
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

  it('captures argo cd deployment requests in the assistant trail as planned workflows', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { response: 'Argo CD deployment request queued for review.' },
    })

    renderWithProviders(
      <AIAssistant />,
      ['/ai?prompt=Deploy%20Payment%20API%20to%20Minikube%20via%20GitHub%20and%20Argo%20CD&autostart=1']
    )

    expect(await screen.findByRole('heading', { name: /workflow delivery trail/i })).toBeInTheDocument()
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
