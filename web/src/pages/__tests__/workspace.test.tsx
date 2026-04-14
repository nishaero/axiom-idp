import { render, screen, waitFor } from '@testing-library/react'
import { fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'
import apiClient from '@/lib/api'
import Dashboard from '@/pages/Dashboard'
import Catalog from '@/pages/Catalog'
import AIAssistant from '@/pages/AIAssistant'

vi.mock('@/lib/api', () => ({
  default: {
    post: vi.fn(),
  },
}))

describe('workspace pages', () => {
  beforeEach(() => {
    vi.mocked(apiClient.post).mockReset()
  })

  it('renders the dashboard as an operational overview', () => {
    render(
      <MemoryRouter>
        <Dashboard />
      </MemoryRouter>
    )

    expect(
      screen.getByRole('heading', { name: /release safely with ai-guided operations/i })
    ).toBeInTheDocument()
    expect(screen.getByText(/AI change-risk forecast/i)).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 3, name: /recent activity/i })).toBeInTheDocument()
  })

  it('filters catalog services by search term', async () => {
    render(
      <MemoryRouter>
        <Catalog />
      </MemoryRouter>
    )

    expect(screen.getByRole('heading', { level: 2, name: /auth gateway/i })).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/search services/i), { target: { value: 'payment' } })

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 2, name: /payment api/i })).toBeInTheDocument()
    })

    expect(screen.queryByRole('heading', { level: 2, name: /auth gateway/i })).not.toBeInTheDocument()
  })

  it('falls back to a local response when the ai api is unavailable', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('offline'))

    render(
      <MemoryRouter>
        <AIAssistant />
      </MemoryRouter>
    )

    fireEvent.click(screen.getByRole('button', { name: /generate a bsi c5 evidence pack for the catalog/i }))

    expect(await screen.findByText(/i assembled the compliance view/i)).toBeInTheDocument()
  })
})
