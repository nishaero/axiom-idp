import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Card, MetricDisplay, StatusBadge } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'
import { usePlatformStatus } from '@/hooks/usePlatformStatus'
import { demoCatalogServices, demoInsights } from '@/lib/demoContent'
import {
  getReleaseDecision,
  getWorkspaceSnapshot,
  sortServicesByOperationalPriority,
} from '@/lib/operatorInsights'

export default function Dashboard() {
  const { serviceHealth, teamActivity, performanceData } = useDashboardStore()
  const { data: platformStatus } = usePlatformStatus()
  const workspaceSnapshot = useMemo(() => getWorkspaceSnapshot(demoCatalogServices), [])
  const releaseQueue = useMemo(() => sortServicesByOperationalPriority(demoCatalogServices).slice(0, 3), [])

  const overview = useMemo(() => {
    const totalServices = serviceHealth.length || 1
    const healthyServices = serviceHealth.filter((service) => service.health.status === 'healthy').length
    const guardedServices = serviceHealth.filter((service) => service.health.status !== 'healthy').length
    const avgUptime = serviceHealth.reduce((sum, service) => sum + service.health.uptime, 0) / totalServices
    const avgResponseTime =
      serviceHealth.reduce((sum, service) => sum + service.health.responseTime, 0) / totalServices
    const recentChanges = teamActivity.filter(
      (activity) => Date.now() - activity.timestamp < 24 * 60 * 60 * 1000
    ).length
    const readiness = Math.max(
      52,
      Math.min(100, Math.round((healthyServices / totalServices) * 100 - guardedServices * 6 + avgUptime / 2))
    )
    const compliance = Math.max(58, Math.min(100, Math.round(100 - guardedServices * 8 - recentChanges * 2)))
    const latestThroughput = Math.round(performanceData[performanceData.length - 1]?.throughput ?? 0)

    return {
      totalServices,
      healthyServices,
      guardedServices,
      avgUptime,
      avgResponseTime,
      recentChanges,
      readiness,
      compliance,
      latestThroughput,
    }
  }, [performanceData, serviceHealth, teamActivity])

  const liveOverview = platformStatus?.overview
  const displayedOverview = {
    totalServices: liveOverview?.total_services ?? overview.totalServices,
    healthyServices: liveOverview?.ready_services ?? overview.healthyServices,
    guardedServices:
      liveOverview != null
        ? Math.max(0, liveOverview.total_services - liveOverview.ready_services)
        : overview.guardedServices,
    avgUptime: overview.avgUptime,
    avgResponseTime: overview.avgResponseTime,
    recentChanges: overview.recentChanges,
    readiness: liveOverview?.release_readiness ?? overview.readiness,
    compliance: liveOverview?.evidence_coverage ?? overview.compliance,
    latestThroughput: overview.latestThroughput,
  }

  const topDecisionTone =
    workspaceSnapshot.topDecision.verdict === 'go'
      ? 'healthy'
      : workspaceSnapshot.topDecision.verdict === 'watch'
        ? 'degraded'
        : 'unhealthy'

  const platformStatusTone =
    platformStatus?.status === 'ready'
      ? 'healthy'
      : platformStatus?.status === 'degraded'
        ? 'degraded'
        : 'unhealthy'

  const formatRelativeTime = (timestamp: number) => {
    const diff = Date.now() - timestamp
    if (diff < 60_000) return 'Just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`
    return `${Math.floor(diff / 86_400_000)}d ago`
  }

  return (
    <div className="space-y-8 p-6 md:p-8">
      <section className="relative overflow-hidden rounded-3xl border border-gray-200 bg-gradient-to-br from-dark-900 via-dark-800 to-primary-900 text-white shadow-xl dark:border-dark-700">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(14,165,233,0.28),transparent_32%),radial-gradient(circle_at_bottom_left,rgba(56,189,248,0.14),transparent_28%)]" />
        <div className="relative grid gap-8 p-8 lg:grid-cols-[1.25fr_0.9fr]">
          <div className="space-y-6">
            <div className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-white/80">
              AI-native internal developer platform
            </div>
            <div className="inline-flex items-center gap-3">
              <StatusBadge
                status={platformStatusTone}
                label={platformStatus ? `Platform ${platformStatus.status}` : 'Telemetry fallback'}
                size="sm"
              />
              <span className="text-xs uppercase tracking-[0.2em] text-white/55">
                {platformStatus ? `Live backend • ${platformStatus.ai_backend}` : 'Local workspace model'}
              </span>
            </div>
            <div className="space-y-4">
              <h1 className="max-w-3xl text-4xl font-semibold tracking-tight md:text-5xl">
                Release safely with AI-guided operations, evidence, and ownership in one place.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-white/75 md:text-lg">
                Axiom combines GitHub-native service intelligence, compliance evidence, and
                change-risk prediction so teams can move faster without losing control.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Link
                to="/catalog"
                className="inline-flex items-center justify-center rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-dark-900 transition-colors hover:bg-white/90"
              >
                Open service catalog
              </Link>
              <Link
                to="/ai"
                className="inline-flex items-center justify-center rounded-lg border border-white/20 bg-white/10 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-white/15"
              >
                Ask the AI assistant
              </Link>
            </div>
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="rounded-2xl border border-white/10 bg-white/8 p-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Health</p>
                <p className="mt-2 text-3xl font-semibold">
                  {displayedOverview.healthyServices}/{displayedOverview.totalServices}
                </p>
                <p className="mt-1 text-sm text-white/70">Services in a green state</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/8 p-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Readiness</p>
                <p className="mt-2 text-3xl font-semibold">{displayedOverview.readiness}%</p>
                <p className="mt-1 text-sm text-white/70">Release readiness forecast</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/8 p-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Evidence</p>
                <p className="mt-2 text-3xl font-semibold">{displayedOverview.compliance}%</p>
                <p className="mt-1 text-sm text-white/70">BSI C5 evidence completion</p>
              </div>
            </div>
          </div>

          <div className="space-y-4 rounded-3xl border border-white/10 bg-white/10 p-5 backdrop-blur">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.2em] text-white/55">Release decision cockpit</p>
                <p className="mt-1 text-lg font-semibold">{workspaceSnapshot.topService.name}</p>
              </div>
              <StatusBadge
                status={topDecisionTone}
                label={workspaceSnapshot.topDecision.title}
              />
            </div>
            <p className="text-sm leading-6 text-white/75">{workspaceSnapshot.topDecision.summary}</p>
            <div className="grid gap-3 sm:grid-cols-3">
              <div className="rounded-2xl bg-black/20 p-4 text-sm text-white/75">
                <p className="text-white/55">Ready</p>
                <p className="mt-1 text-xl font-semibold">{workspaceSnapshot.readyServices}</p>
              </div>
              <div className="rounded-2xl bg-black/20 p-4 text-sm text-white/75">
                <p className="text-white/55">Watch</p>
                <p className="mt-1 text-xl font-semibold">{workspaceSnapshot.watchServices}</p>
              </div>
              <div className="rounded-2xl bg-black/20 p-4 text-sm text-white/75">
                <p className="text-white/55">Blocked</p>
                <p className="mt-1 text-xl font-semibold">{workspaceSnapshot.blockedServices}</p>
              </div>
            </div>
            <div className="space-y-3 rounded-2xl bg-black/20 p-4 text-sm text-white/75">
              <p className="text-white/55">Why this matters</p>
              <ul className="space-y-2">
                {workspaceSnapshot.topDecision.reasons.slice(0, 3).map((reason) => (
                  <li key={reason}>• {reason}</li>
                ))}
              </ul>
            </div>
            <div className="flex flex-wrap gap-3">
              <Link
                to="/catalog"
                className="inline-flex items-center justify-center rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-dark-900 transition-colors hover:bg-white/90"
              >
                Open drilldown
              </Link>
              <Link
                to={`/ai?prompt=${encodeURIComponent(workspaceSnapshot.topDecision.aiPrompt)}&autostart=1`}
                className="inline-flex items-center justify-center rounded-lg border border-white/20 bg-white/10 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-white/15"
              >
                Ask AI for guidance
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <MetricDisplay
            value={`${displayedOverview.healthyServices}/${displayedOverview.totalServices}`}
            label="Services healthy"
            color="success"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={`${Math.round(overview.avgResponseTime)}ms`}
            label="Mean response time"
            trend={-6.2}
            trendLabel="vs last week"
            color="primary"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={`${displayedOverview.readiness}%`}
            label="Release readiness"
            trend={4.8}
            trendLabel="improving"
            color="warning"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={`${displayedOverview.compliance}%`}
            label="Evidence completeness"
            trend={2.1}
            trendLabel="BSI C5 aligned"
            color="success"
          />
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <Card title="Live platform status" subtitle="Backend-fed observability summary from the control plane">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40">
              <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Control plane</p>
              <p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {platformStatus?.status ?? 'fallback'}
              </p>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {platformStatus
                  ? `${platformStatus.environment} • uptime ${platformStatus.uptime}`
                  : 'Dashboard is using local fallback signals while the API is unavailable.'}
              </p>
            </div>
            <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40">
              <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Audit trail</p>
              <p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {platformStatus?.audit.entries ?? 0}
              </p>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {platformStatus
                  ? `${platformStatus.audit.error_count} errors, ${platformStatus.audit.denied_count} denied requests`
                  : 'Audit metrics will appear here once the backend is connected.'}
              </p>
            </div>
          </div>
          <div className="mt-4 grid gap-3">
            {(platformStatus?.alerts ?? []).slice(0, 3).map((alert) => (
              <div
                key={`${alert.scope}-${alert.title}`}
                className="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
              >
                <div className="flex items-center justify-between gap-3">
                  <p className="font-semibold text-gray-900 dark:text-white">{alert.title}</p>
                  <StatusBadge
                    status={alert.severity === 'high' ? 'unhealthy' : alert.severity === 'medium' ? 'degraded' : 'healthy'}
                    label={alert.severity}
                    size="sm"
                  />
                </div>
                <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">{alert.detail}</p>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Operational telemetry" subtitle="Signals surfaced directly from the running IDP">
          <div className="space-y-4">
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">AI backend</p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {platformStatus
                  ? `${platformStatus.ai_backend} in namespace ${platformStatus.kubernetes_namespace || 'default'}`
                  : 'Waiting for live backend data.'}
              </p>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">Rate limiting</p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {platformStatus
                  ? `${platformStatus.rate_limiting.requests_per_min} requests/min with ${platformStatus.rate_limiting.tracked_keys} active keys`
                  : 'Live rate limiting stats unavailable.'}
              </p>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">Observability notes</p>
              <ul className="mt-2 space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {(platformStatus?.observability_notes ?? []).slice(0, 3).map((note) => (
                  <li key={note}>• {note}</li>
                ))}
              </ul>
            </div>
          </div>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <Card title="Release queue" subtitle="The services an operator should review first">
          <div className="space-y-3">
            {releaseQueue.map((service) => {
              const decision = getReleaseDecision(service)
              const decisionTone =
                decision.verdict === 'go' ? 'healthy' : decision.verdict === 'watch' ? 'degraded' : 'unhealthy'

              return (
                <div
                  key={service.id}
                  className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-semibold text-gray-900 dark:text-white">{service.name}</p>
                      <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{decision.summary}</p>
                    </div>
                    <StatusBadge status={decisionTone} label={decision.title} size="sm" />
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                    <span>Owner: {service.owner}</span>
                    <span>•</span>
                    <span>{service.tier}</span>
                    <span>•</span>
                    <span>{service.healthScore}% health</span>
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Link
                      to="/catalog"
                      className="rounded-full border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-600 dark:text-gray-200"
                    >
                      Open drilldown
                    </Link>
                    <Link
                      to={`/ai?prompt=${encodeURIComponent(decision.aiPrompt)}&autostart=1`}
                      className="rounded-full bg-primary-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-primary-700"
                    >
                      Ask AI
                    </Link>
                  </div>
                </div>
              )
            })}
          </div>
        </Card>

        <Card title="Product differentiators" subtitle="Capabilities competitors still underserve">
          <div className="grid gap-4 md:grid-cols-3">
            {demoInsights.map((insight) => (
              <div
                key={insight.title}
                className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40"
              >
                <p className="text-sm font-semibold text-gray-900 dark:text-white">{insight.title}</p>
                <p className="mt-2 text-2xl font-bold text-primary-600 dark:text-primary-400">
                  {insight.value}
                </p>
                <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                  {insight.description}
                </p>
              </div>
            ))}
          </div>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <Card
          title="Recent activity"
          subtitle="The latest changes and release events across the control plane"
          actions={
            <Link
              to="/ai"
              className="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Ask AI for a summary
            </Link>
          }
        >
          <div className="space-y-3">
            {teamActivity.slice(0, 4).map((activity) => (
              <div
                key={activity.id}
                className="flex items-start justify-between gap-4 rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
              >
                <div>
                  <p className="font-semibold text-gray-900 dark:text-white">{activity.user}</p>
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {activity.action} <span className="font-medium text-gray-900 dark:text-white">{activity.target}</span>
                  </p>
                </div>
                <span className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  {formatRelativeTime(activity.timestamp)}
                </span>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Operational priorities" subtitle="What the platform should surface before a release">
          <div className="space-y-4">
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">1. Close owner gaps</p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                Services with stale ownership should be flagged before release plans proceed.
              </p>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">2. Export BSI C5 evidence</p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                Every deployment and approval should be convertible into an audit-ready packet.
              </p>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">3. Forecast change risk</p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                AI-assisted risk scoring should explain why a service is ready, watch, or blocked.
              </p>
            </div>
          </div>
        </Card>
      </section>
    </div>
  )
}
