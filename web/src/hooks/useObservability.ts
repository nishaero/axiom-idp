import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api'
import type { PlatformStatusResponse } from './usePlatformStatus'

export interface ObservabilityTelemetry {
  http_requests_total: number
  http_error_responses_total: number
  http_rate_limited_total: number
  ai_requests_total: number
  ai_failures_total: number
  deployment_requests_total: number
  audit_events_total: number
  last_request_at?: string
  last_ai_request_at?: string
  last_deployment_at?: string
  last_audit_at?: string
}

export interface ObservabilityEndpoint {
  name: string
  path: string
  status: string
  description: string
}

export interface ObservabilityResponse {
  platform: PlatformStatusResponse
  telemetry: ObservabilityTelemetry
  endpoints: ObservabilityEndpoint[]
  metrics_endpoint: string
  prometheus_annotations: string[]
  notes: string[]
}

export function useObservability() {
  return useQuery({
    queryKey: ['observability'],
    queryFn: async () => {
      const response = await apiClient.get<ObservabilityResponse>('/platform/observability')
      return response.data
    },
    staleTime: 30_000,
    retry: 1,
  })
}
