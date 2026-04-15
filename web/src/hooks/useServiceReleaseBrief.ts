import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api'
import type { ReleaseBrief } from '@/lib/operatorInsights'

interface ServiceReleaseBriefResponse {
  brief: {
    service_id: string
    service_name: string
    decision: string
    summary: string
    readiness_score: number
    evidence_coverage: number
    missing_evidence: string[]
    next_best_action: string
    next_best_owner: string
    next_best_effort: string
    next_best_impact: string
    supporting_actions: string[]
    evidence_pack: string[]
    portfolio_context: string
    assistant_prompt: string
    export_label: string
  }
}

function mapReleaseBrief(response: ServiceReleaseBriefResponse['brief']): ReleaseBrief {
  return {
    serviceName: response.service_name,
    decision: response.decision,
    summary: response.summary,
    readinessScore: response.readiness_score,
    evidenceCoverage: response.evidence_coverage,
    missingEvidence: response.missing_evidence ?? [],
    nextBestAction: response.next_best_action,
    nextBestOwner: response.next_best_owner,
    nextBestEffort: response.next_best_effort,
    nextBestImpact: response.next_best_impact,
    supportingActions: response.supporting_actions ?? [],
    evidencePack: response.evidence_pack ?? [],
    portfolioContext: response.portfolio_context,
    assistantPrompt: response.assistant_prompt,
    exportLabel: response.export_label,
  }
}

export function useServiceReleaseBrief(serviceId?: string | null) {
  return useQuery({
    queryKey: ['service-release-brief', serviceId],
    enabled: Boolean(serviceId),
    queryFn: async () => {
      const response = await apiClient.get<ServiceReleaseBriefResponse>(`/catalog/services/${serviceId}/analysis`)
      return mapReleaseBrief(response.data.brief)
    },
    staleTime: 30_000,
    retry: 1,
  })
}
