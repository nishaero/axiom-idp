import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Card, ProgressBar, StatusBadge } from '@/components/dashboard'
import { demoCatalogServices } from '@/lib/demoContent'
import type { DemoCatalogService } from '@/lib/demoContent'
import {
  buildAssistantPrompts,
  getReleaseDecision,
  getWorkspaceSnapshot,
  sortServicesByOperationalPriority,
} from '@/lib/operatorInsights'

const categories = ['all', ...new Set(demoCatalogServices.map((service) => service.tier))]

function matchesSearch(service: DemoCatalogService, query: string) {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return true

  return [
    service.name,
    service.description,
    service.owner,
    service.team,
    service.tier,
    service.releaseState,
    ...service.tags,
    ...service.signals,
  ]
    .join(' ')
    .toLowerCase()
    .includes(normalized)
}

export default function Catalog() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string>('all')
  const [selectedServiceId, setSelectedServiceId] = useState(
    sortServicesByOperationalPriority(demoCatalogServices)[0]?.id ?? ''
  )

  const workspaceSnapshot = useMemo(() => getWorkspaceSnapshot(demoCatalogServices), [])
  const serviceDecisions = useMemo(
    () =>
      new Map(
        demoCatalogServices.map((service) => [
          service.id,
          getReleaseDecision(service),
        ])
      ),
    []
  )

  const filteredServices = useMemo(
    () =>
      sortServicesByOperationalPriority(demoCatalogServices).filter(
        (service) =>
          matchesSearch(service, searchQuery) &&
          (selectedCategory === 'all' || service.tier === selectedCategory)
      ),
    [searchQuery, selectedCategory]
  )

  const selectedService =
    filteredServices.find((service) => service.id === selectedServiceId) ??
    filteredServices[0] ??
    null
  const selectedDecision = selectedService ? serviceDecisions.get(selectedService.id) ?? getReleaseDecision(selectedService) : null

  useEffect(() => {
    if (filteredServices.length === 0) {
      setSelectedServiceId('')
      return
    }

    if (!filteredServices.some((service) => service.id === selectedServiceId)) {
      setSelectedServiceId(filteredServices[0].id)
    }
  }, [filteredServices, selectedServiceId])

  const readyServices = demoCatalogServices.filter((service) => service.releaseState === 'ready').length
  const ownerlessServices = demoCatalogServices.filter((service) => service.owner === 'Unassigned').length
  const criticalServices = demoCatalogServices.filter((service) => service.tier === 'critical').length
  const searchSummary = filteredServices.length === demoCatalogServices.length
    ? 'Showing the full workspace'
    : `Showing ${filteredServices.length} of ${demoCatalogServices.length} services`

  return (
    <div className="space-y-8 p-6 md:p-8">
      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
          <div className="space-y-4">
            <div className="inline-flex rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
              Demo registry snapshot
            </div>
            <div className="space-y-3">
              <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-4xl">
                Service catalog built for ownership, evidence, and release decisions.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-gray-600 dark:text-gray-300">
                This view surfaces service scorecards, BSI C5 signals, and AI-ready workflow
                metadata so teams can move from discovery to action without jumping between tools.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              {[
                'AI release-risk forecast',
                'Evidence pack export',
                'Ownership drift detection',
              ].map((item) => (
                <span
                  key={item}
                  className="inline-flex items-center rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-700/40 dark:text-gray-300"
                >
                  {item}
                </span>
              ))}
            </div>
          </div>

          <Card title="Catalog health" subtitle="Current snapshot across the demo registry">
            <div className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                  <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Services</p>
                  <p className="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
                    {demoCatalogServices.length}
                  </p>
                </div>
                <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                  <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Ready</p>
                  <p className="mt-2 text-2xl font-bold text-green-600 dark:text-green-400">{readyServices}</p>
                </div>
                <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                  <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Owner gaps</p>
                  <p className="mt-2 text-2xl font-bold text-amber-600 dark:text-amber-400">
                    {ownerlessServices}
                  </p>
                </div>
              </div>
              <div className="rounded-2xl border border-gray-200 p-4 dark:border-dark-700">
                <p className="text-sm font-medium text-gray-900 dark:text-white">Critical services</p>
                <p className="mt-2 text-3xl font-semibold text-primary-600 dark:text-primary-400">
                  {criticalServices}
                </p>
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  These services require the strictest release discipline and evidence coverage.
                </p>
              </div>
              <div className="rounded-2xl bg-primary-50 p-4 dark:bg-primary-950/30">
                <p className="text-xs uppercase tracking-[0.2em] text-primary-700 dark:text-primary-300">
                  Operator focus
                </p>
                <p className="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                  {workspaceSnapshot.topService.name}
                </p>
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">
                  {workspaceSnapshot.topDecision.title}: {workspaceSnapshot.topDecision.summary}
                </p>
              </div>
            </div>
          </Card>
        </div>
      </section>

      <section className="space-y-3">
        <div className="flex flex-wrap gap-2">
          {['release risk', 'owner gaps', 'evidence pack', 'rollback plan'].map((term) => (
            <button
              key={term}
              type="button"
              onClick={() => setSearchQuery(term)}
              className="rounded-full border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300"
            >
              {term}
            </button>
          ))}
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400">{searchSummary}</p>
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.1fr_auto]">
        <div className="relative">
          <input
            type="text"
            placeholder="Search services, owners, controls, or signals..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="w-full rounded-2xl border border-gray-300 bg-white px-4 py-3 pr-12 text-gray-900 shadow-sm outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 dark:border-dark-700 dark:bg-dark-800 dark:text-white"
          />
          <svg
            className="pointer-events-none absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="m21 21-6-6m2-5a7 7 0 1 1-14 0 7 7 0 0 1 14 0Z" />
          </svg>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {categories.map((category) => {
            const isActive = selectedCategory === category

            return (
              <button
                key={category}
                type="button"
                onClick={() => setSelectedCategory(category)}
                className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-primary-600 text-white shadow-sm'
                    : 'border border-gray-300 bg-white text-gray-700 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300'
                }`}
              >
                {category === 'all' ? 'All tiers' : category}
              </button>
            )
          })}
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <div className="space-y-4">
          {filteredServices.length > 0 ? (
            filteredServices.map((service) => {
              const isSelected = selectedService?.id === service.id
              const decision = getReleaseDecision(service)
              const decisionTone =
                decision.verdict === 'go' ? 'healthy' : decision.verdict === 'watch' ? 'degraded' : 'unhealthy'

              return (
                <button
                  key={service.id}
                  type="button"
                  onClick={() => setSelectedServiceId(service.id)}
                  className={`w-full rounded-3xl border p-5 text-left shadow-sm transition-all ${
                    isSelected
                      ? 'border-primary-500 bg-primary-50 shadow-md dark:border-primary-500 dark:bg-primary-950/30'
                      : 'border-gray-200 bg-white hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-800'
                  }`}
                  aria-pressed={isSelected}
                >
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{service.name}</h2>
                        <StatusBadge status={decisionTone} label={decision.title} size="sm" />
                      </div>
                      <p className="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
                        {decision.summary}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Health</p>
                      <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{service.healthScore}</p>
                    </div>
                  </div>

                  <div className="mt-4 grid gap-4 md:grid-cols-[1.5fr_0.8fr]">
                    <div className="space-y-3">
                      <div className="flex flex-wrap gap-2">
                        {service.tags.map((tag) => (
                          <span
                            key={tag}
                            className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                      <div className="text-sm text-gray-600 dark:text-gray-400">
                        Owner: <span className="font-medium text-gray-900 dark:text-white">{service.owner}</span> · Team:{' '}
                        <span className="font-medium text-gray-900 dark:text-white">{service.team}</span>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <ProgressBar
                        value={service.healthScore}
                        label="Scorecard"
                        color={service.releaseState === 'blocked' ? 'danger' : service.releaseState === 'watch' ? 'warning' : 'success'}
                        size="sm"
                      />
                      <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                        <span>Last deploy {service.lastDeployed}</span>
                        <span>{service.monthlyCost}/mo</span>
                      </div>
                    </div>
                  </div>
                </button>
              )
            })
          ) : (
            <Card title="No matching services" subtitle="Broaden the filter or clear the search term.">
              <button
                type="button"
                onClick={() => {
                  setSearchQuery('')
                  setSelectedCategory('all')
                }}
                className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-700"
              >
                Reset filters
              </button>
            </Card>
          )}
        </div>

        <Card
          title={selectedService?.name ?? 'Select a service'}
          subtitle={
            selectedService
              ? `${selectedService.owner} · ${selectedService.team} · ${selectedService.releaseState}`
              : 'Pick a service to inspect its release posture and evidence.'
          }
          className="h-fit"
          actions={
            selectedService ? (
              <Link
                to={`/ai?prompt=${encodeURIComponent(selectedDecision?.aiPrompt ?? buildAssistantPrompts(selectedService)[0])}&autostart=1`}
                className="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                Ask AI about this service
              </Link>
            ) : null
          }
        >
          {selectedService ? (
            <div className="space-y-5">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge
                  status={
                    selectedDecision?.verdict === 'go'
                      ? 'healthy'
                      : selectedDecision?.verdict === 'watch'
                        ? 'degraded'
                        : 'unhealthy'
                  }
                  label={selectedDecision?.title ?? selectedService.releaseState}
                />
                <StatusBadge
                  status={selectedService.riskLevel === 'high' ? 'unhealthy' : selectedService.riskLevel === 'medium' ? 'degraded' : 'healthy'}
                  label={`${selectedService.riskLevel} risk`}
                />
                <span className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">
                  {selectedService.tier}
                </span>
                <span className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">
                  {selectedService.monthlyCost}/mo
                </span>
              </div>

              <div className="rounded-2xl border border-gray-200 p-4 dark:border-dark-700">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Decision summary
                </p>
                <p className="mt-2 text-base leading-7 text-gray-900 dark:text-white">
                  {selectedDecision?.summary ?? selectedService.description}
                </p>
                <div className="mt-4 grid gap-3 md:grid-cols-2">
                  <div className="space-y-2 rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                    <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Why</p>
                    <ul className="space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                      {(selectedDecision?.reasons ?? []).slice(0, 4).map((reason) => (
                        <li key={reason}>• {reason}</li>
                      ))}
                    </ul>
                  </div>
                  <div className="space-y-2 rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                    <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                      Evidence to export
                    </p>
                    <ul className="space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                      {(selectedDecision?.evidence ?? selectedService.evidence).map((entry) => (
                        <li key={entry}>• {entry}</li>
                      ))}
                    </ul>
                  </div>
                </div>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                  <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">Release controls</p>
                  <div className="mt-3 space-y-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                    <p>
                      <span className="font-medium text-gray-900 dark:text-white">Signals:</span>{' '}
                      {selectedService.signals.join(', ')}
                    </p>
                    <p>
                      <span className="font-medium text-gray-900 dark:text-white">Health:</span>{' '}
                      {selectedService.healthScore}%
                    </p>
                    <p>
                      <span className="font-medium text-gray-900 dark:text-white">Last deploy:</span>{' '}
                      {selectedService.lastDeployed}
                    </p>
                  </div>
                </div>
                <div className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                  <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">AI actions</p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    {buildAssistantPrompts(selectedService).slice(0, 3).map((prompt) => (
                      <Link
                        key={prompt}
                        to={`/ai?prompt=${encodeURIComponent(prompt)}&autostart=1`}
                        className="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 transition-colors hover:border-primary-300 hover:bg-primary-100 dark:border-primary-900 dark:bg-primary-900/30 dark:text-primary-300 dark:hover:bg-primary-900/50"
                      >
                        {prompt}
                      </Link>
                    ))}
                  </div>
                </div>
              </div>

              <div className="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <p className="text-xs uppercase tracking-[0.2em] text-gray-500 dark:text-gray-400">
                  Recommended next step
                </p>
                <p className="mt-2 text-sm leading-6 text-gray-700 dark:text-gray-300">
                  {selectedDecision?.nextSteps[0] ?? 'Inspect the service and generate a fresh release summary.'}
                </p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-600 dark:text-gray-400">
              No service selected. Expand the filters or reset the search to continue.
            </p>
          )}
        </Card>
      </section>
    </div>
  )
}
