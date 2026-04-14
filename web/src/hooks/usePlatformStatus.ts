import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api'

export interface PlatformAlert {
  severity: string
  scope: string
  title: string
  detail: string
}

export interface PlatformAuditStats {
  entries: number
  last_entry_at?: string
  error_count: number
  denied_count: number
  success_count: number
}

export interface PlatformRateLimiting {
  enabled: boolean
  tracked_keys: number
  requests_seen: number
  last_cleanup_at?: string
  requests_per_min: number
}

export interface PlatformServiceStatus {
  id: string
  name: string
  status: string
  release_state: string
  risk_level: string
  health_score: number
}

export interface PlatformOverview {
  total_services: number
  ready_services: number
  watch_services: number
  blocked_services: number
  owner_gap_count: number
  release_readiness: number
  evidence_coverage: number
}

export interface PlatformStatusResponse {
  status: string
  environment: string
  started_at: string
  uptime: string
  ai_backend: string
  kubernetes_namespace: string
  alerts: PlatformAlert[]
  overview: PlatformOverview
  services: PlatformServiceStatus[]
  audit: PlatformAuditStats
  rate_limiting: PlatformRateLimiting
  observability_notes: string[]
}

export function usePlatformStatus() {
  return useQuery({
    queryKey: ['platform-status'],
    queryFn: async () => {
      const response = await apiClient.get<PlatformStatusResponse>('/platform/status')
      return response.data
    },
    staleTime: 30_000,
    retry: 1,
  })
}
