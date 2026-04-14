import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Card, MetricDisplay, ProgressBar, StatusBadge } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'
import { demoInsights } from '@/lib/demoContent'

function formatCount(value: number) {
  return new Intl.NumberFormat('en-US').format(value)
}

export default function Dashboard() {
  const { serviceHealth, teamActivity, performanceData } = useDashboardStore()

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

  const healthTone =
    overview.guardedServices === 0 ? 'healthy' : overview.guardedServices === 1 ? 'degraded' : 'unhealthy'

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
                  {overview.healthyServices}/{overview.totalServices}
                </p>
                <p className="mt-1 text-sm text-white/70">Services in a green state</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/8 p-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Readiness</p>
                <p className="mt-2 text-3xl font-semibold">{overview.readiness}%</p>
                <p className="mt-1 text-sm text-white/70">Release readiness forecast</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/8 p-4 backdrop-blur">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">Evidence</p>
                <p className="mt-2 text-3xl font-semibold">{overview.compliance}%</p>
                <p className="mt-1 text-sm text-white/70">BSI C5 evidence completion</p>
              </div>
            </div>
          </div>

          <div className="space-y-4 rounded-3xl border border-white/10 bg-white/10 p-5 backdrop-blur">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.2em] text-white/55">Platform posture</p>
                <p className="mt-1 text-lg font-semibold">Live readiness snapshot</p>
              </div>
              <StatusBadge status={healthTone} label={healthTone === 'healthy' ? 'Stable' : 'Needs attention'} />
            </div>
            <div className="space-y-4">
              <ProgressBar
                value={overview.readiness}
                label="Release readiness"
                color={overview.readiness > 85 ? 'success' : overview.readiness > 70 ? 'warning' : 'danger'}
              />
              <ProgressBar
                value={overview.compliance}
                label="Compliance evidence"
                color={overview.compliance > 85 ? 'success' : 'warning'}
              />
              <ProgressBar
                value={Math.max(35, 100 - overview.guardedServices * 24)}
                label="Operational risk"
                color={overview.guardedServices > 1 ? 'danger' : 'warning'}
              />
            </div>
            <div className="grid gap-3 text-sm text-white/75">
              <div className="rounded-2xl bg-black/20 p-4">
                <p className="text-white/55">Latest throughput</p>
                <p className="mt-1 text-xl font-semibold">{formatCount(overview.latestThroughput)} req/min</p>
              </div>
              <div className="rounded-2xl bg-black/20 p-4">
                <p className="text-white/55">Change events today</p>
                <p className="mt-1 text-xl font-semibold">{formatCount(overview.recentChanges)}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <MetricDisplay
            value={`${overview.healthyServices}/${overview.totalServices}`}
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
            value={`${overview.readiness}%`}
            label="Release readiness"
            trend={4.8}
            trendLabel="improving"
            color="warning"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={`${overview.compliance}%`}
            label="Evidence completeness"
            trend={2.1}
            trendLabel="BSI C5 aligned"
            color="success"
          />
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
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

        <Card
          title="Top services"
          subtitle="The first signals an operator needs before a release decision"
          actions={
            <Link
              to="/catalog"
              className="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Open catalog
            </Link>
          }
        >
          <div className="space-y-3">
            {serviceHealth.slice(0, 3).map((service) => (
              <div
                key={service.id}
                className="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">{service.name}</p>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                      {service.health.responseTime.toFixed(0)} ms response time, {service.uptime.toFixed(1)}%
                      uptime
                    </p>
                  </div>
                  <StatusBadge status={service.health.status} size="sm" />
                </div>
                <div className="mt-3">
                  <ProgressBar
                    value={service.health.uptime}
                    color={service.health.status === 'healthy' ? 'success' : 'warning'}
                    size="sm"
                    showValue={false}
                  />
                </div>
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
