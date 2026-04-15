import { Link } from 'react-router-dom'
import { Card, MetricDisplay, ProgressBar, StatusBadge } from '@/components/dashboard'
import { useObservability } from '@/hooks/useObservability'

function mapStatusToBadge(status: string): 'healthy' | 'degraded' | 'unhealthy' | 'unknown' {
  switch (status) {
    case 'ready':
    case 'alive':
    case 'healthy':
      return 'healthy'
    case 'degraded':
      return 'degraded'
    case 'unready':
    case 'unhealthy':
      return 'unhealthy'
    default:
      return 'unknown'
  }
}

function formatTime(value?: string) {
  if (!value) return 'Not yet recorded'

  const parsed = Date.parse(value)
  if (Number.isNaN(parsed)) return value

  return new Intl.DateTimeFormat('en-GB', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed)
}

export default function Observability() {
  const { data, isLoading, error, refetch } = useObservability()
  const platform = data?.platform
  const telemetry = data?.telemetry
  const readyPercent = platform?.overview.total_services
    ? Math.round((platform.overview.ready_services / platform.overview.total_services) * 100)
    : 0
  const blockedPercent = platform?.overview.total_services
    ? Math.round((platform.overview.blocked_services / platform.overview.total_services) * 100)
    : 0

  return (
    <div className="space-y-8 p-6 md:p-8">
      <section className="rounded-3xl border border-gray-200 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white shadow-xl dark:border-dark-700">
        <div className="relative overflow-hidden rounded-3xl">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(34,211,238,0.26),transparent_30%),radial-gradient(circle_at_bottom_left,rgba(14,165,233,0.18),transparent_24%)]" />
          <div className="relative grid gap-8 p-8 lg:grid-cols-[1.1fr_0.9fr]">
            <div className="space-y-5">
              <div className="inline-flex items-center rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-white/75">
                Observability control tower
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <StatusBadge
                  status={platform ? mapStatusToBadge(platform.status) : 'unknown'}
                  label={platform ? `Platform ${platform.status}` : 'Loading status'}
                />
                <span className="text-xs uppercase tracking-[0.18em] text-white/55">
                  {platform ? `${platform.environment} • ${platform.ai_backend}` : 'Backend snapshot pending'}
                </span>
              </div>
              <h1 className="max-w-3xl text-4xl font-semibold tracking-tight md:text-5xl">
                Prometheus metrics, health probes, and backend status in one operator view.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-white/75 md:text-lg">
                This page is fed from the live control plane. It surfaces endpoint health, telemetry
                counters, and scrape hints so operators can validate the platform before rollout.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  to="/"
                  className="inline-flex items-center justify-center rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-slate-950 transition-colors hover:bg-white/90"
                >
                  Back to dashboard
                </Link>
                <Link
                  to="/ai"
                  className="inline-flex items-center justify-center rounded-lg border border-white/20 bg-white/10 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-white/15"
                >
                  Ask AI for remediation
                </Link>
              </div>
            </div>

            <div className="space-y-4 rounded-3xl border border-white/10 bg-white/10 p-5 backdrop-blur">
              <div className="rounded-2xl bg-black/20 p-4">
                <p className="text-xs uppercase tracking-[0.2em] text-white/55">Metrics endpoint</p>
                <p className="mt-2 text-lg font-semibold">{data?.metrics_endpoint ?? '/metrics'}</p>
                <p className="mt-1 text-sm text-white/70">
                  Prometheus can scrape the endpoint directly using the service annotations.
                </p>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl bg-black/20 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-white/55">Ready services</p>
                  <p className="mt-2 text-2xl font-semibold">
                    {platform?.overview.ready_services ?? 0}/{platform?.overview.total_services ?? 0}
                  </p>
                </div>
                <div className="rounded-2xl bg-black/20 p-4">
                  <p className="text-xs uppercase tracking-[0.2em] text-white/55">Blocked services</p>
                  <p className="mt-2 text-2xl font-semibold">
                    {platform?.overview.blocked_services ?? 0}/{platform?.overview.total_services ?? 0}
                  </p>
                </div>
              </div>
              <ProgressBar value={readyPercent} label="Ready services" color="success" />
              <ProgressBar value={blockedPercent} label="Blocked services" color="warning" />
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <MetricDisplay
            value={telemetry?.http_requests_total ?? 0}
            label="HTTP requests"
            color="primary"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={telemetry?.ai_requests_total ?? 0}
            label="AI requests"
            trend={telemetry && telemetry.ai_failures_total > 0 ? -telemetry.ai_failures_total : 0}
            trendLabel="AI-assisted paths"
            color="success"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={telemetry?.deployment_requests_total ?? 0}
            label="Deployment actions"
            color="warning"
          />
        </Card>
        <Card>
          <MetricDisplay
            value={telemetry?.audit_events_total ?? 0}
            label="Audit events"
            color="danger"
          />
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <Card title="Endpoint health" subtitle="The surfaces operators and probes should watch first" isLoading={isLoading} error={error instanceof Error ? error.message : null} onRefresh={refetch}>
          <div className="space-y-3">
            {(data?.endpoints ?? []).map((endpoint) => (
              <div
                key={endpoint.path}
                className="flex flex-col gap-3 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40 md:flex-row md:items-center md:justify-between"
              >
                <div>
                  <p className="font-semibold text-gray-900 dark:text-white">{endpoint.name}</p>
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{endpoint.description}</p>
                  <p className="mt-1 text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                    {endpoint.path}
                  </p>
                </div>
                <StatusBadge status={mapStatusToBadge(endpoint.status)} label={endpoint.status} size="sm" />
              </div>
            ))}
          </div>
        </Card>

        <Card title="Telemetry signals" subtitle="Prometheus-style counters mirrored into the UI">
          <div className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Last request
                </p>
                <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                  {formatTime(telemetry?.last_request_at)}
                </p>
              </div>
              <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Last AI request
                </p>
                <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                  {formatTime(telemetry?.last_ai_request_at)}
                </p>
              </div>
              <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Last deployment
                </p>
                <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                  {formatTime(telemetry?.last_deployment_at)}
                </p>
              </div>
              <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Last audit event
                </p>
                <p className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
                  {formatTime(telemetry?.last_audit_at)}
                </p>
              </div>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">Scrape annotations</p>
              <ul className="mt-2 space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {(data?.prometheus_annotations ?? []).map((annotation) => (
                  <li key={annotation}>• {annotation}</li>
                ))}
              </ul>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-sm font-semibold text-gray-900 dark:text-white">Operational notes</p>
              <ul className="mt-2 space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {(data?.notes ?? []).map((note) => (
                  <li key={note}>• {note}</li>
                ))}
              </ul>
            </div>
          </div>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <Card title="Platform snapshot" subtitle="The live control-plane summary behind the UI">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Uptime</p>
              <p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{platform?.uptime ?? 'n/a'}</p>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {platform?.started_at ? `Started ${formatTime(platform.started_at)}` : 'Backend not available'}
              </p>
            </div>
            <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
              <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Audit trail</p>
              <p className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {platform?.audit.entries ?? 0}
              </p>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {platform
                  ? `${platform.audit.error_count} errors, ${platform.audit.denied_count} denied, ${platform.audit.success_count} successful`
                  : 'Waiting for live backend data'}
              </p>
            </div>
          </div>
        </Card>

        <Card title="Backend checks" subtitle="Configured readiness and operational guardrails">
          <div className="space-y-3">
            {(platform?.checks ?? []).map((check) => (
              <div
                key={check.name}
                className="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-700/40"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-white">{check.name}</p>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{check.message}</p>
                  </div>
                  <StatusBadge status={mapStatusToBadge(check.status)} label={check.status} size="sm" />
                </div>
              </div>
            ))}
          </div>
        </Card>
      </section>
    </div>
  )
}
