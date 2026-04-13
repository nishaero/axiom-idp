import React, { useMemo } from 'react'
import { Card, MetricDisplay, StatusBadge, ProgressBar } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'

interface ServiceHealthWidgetProps {
  compact?: boolean
}

export default function ServiceHealthWidget({ compact = false }: ServiceHealthWidgetProps) {
  const { serviceHealth, refreshServiceHealth, lastUpdated, setServiceHealth } = useDashboardStore()

  const metrics = useMemo(() => {
    const avgUptime = serviceHealth.reduce((sum, svc) => sum + svc.health.uptime, 0) / serviceHealth.length
    const avgResponseTime = serviceHealth.reduce((sum, svc) => sum + svc.health.responseTime, 0) / serviceHealth.length
    const avgErrorRate = serviceHealth.reduce((sum, svc) => sum + svc.health.errorRate, 0) / serviceHealth.length
    const healthyServices = serviceHealth.filter((svc) => svc.health.status === 'healthy').length
    const totalReplicas = serviceHealth.reduce((sum, svc) => sum + svc.replicas.desired, 0)
    const availableReplicas = serviceHealth.reduce((sum, svc) => sum + svc.replicas.available, 0)

    return {
      avgUptime: avgUptime.toFixed(2),
      avgResponseTime: Math.round(avgResponseTime),
      avgErrorRate: (avgErrorRate * 100).toFixed(2),
      healthyServices,
      unhealthyServices: serviceHealth.length - healthyServices,
      totalReplicas,
      availableReplicas,
    }
  }, [serviceHealth])

  const statusColor =
    metrics.unhealthyServices === 0
      ? 'success'
      : metrics.unhealthyServices < serviceHealth.length / 2
      ? 'warning'
      : 'danger'

  return (
    <Card
      title="Service Health Summary"
      actions={
        <button
          onClick={() => refreshServiceHealth()}
          className="p-2 hover:bg-gray-100 dark:hover:bg-dark-700 rounded-lg transition-colors"
          aria-label="Refresh health data"
        >
          <svg className="h-5 w-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
        </button>
      }
      isLoading={false}
    >
      <div className={compact ? '' : 'grid grid-cols-3 gap-4 mb-6'}>
        <MetricDisplay
          value={`${metrics.avgUptime}%`}
          label="Average Uptime"
          trend={0.15}
          trendLabel="vs last week"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          }
          color="success"
        />
        <MetricDisplay
          value={`${metrics.avgResponseTime}ms`}
          label="Avg Response Time"
          trend={-5.2}
          trendLabel="improvement"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          }
          color="primary"
        />
        <MetricDisplay
          value={`${metrics.avgErrorRate}%`}
          label="Error Rate"
          trend={-0.03}
          trendLabel="improvement"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
          }
          color={parseFloat(metrics.avgErrorRate) > 1 ? 'danger' : 'success'}
        />
      </div>

      <div className={compact ? 'grid grid-cols-2 gap-4' : 'mb-6'}>
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600 dark:text-gray-400">Healthy Services</span>
          <span className="text-lg font-semibold text-green-600 dark:text-green-400">
            {metrics.healthyServices} / {serviceHealth.length}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-600 dark:text-gray-400">Replicas Available</span>
          <span className="text-lg font-semibold text-primary-600 dark:text-primary-400">
            {metrics.availableReplicas} / {metrics.totalReplicas}
          </span>
        </div>
      </div>

      {!compact && (
        <div className="space-y-3">
          {serviceHealth.map((service) => (
            <div
              key={service.id}
              className="flex items-center justify-between p-3 bg-gray-50 dark:bg-dark-700/50 rounded-lg"
            >
              <div className="flex items-center gap-3 flex-1">
                <StatusBadge
                  status={service.health.status}
                  size="sm"
                  showIcon={true}
                />
                <div className="min-w-0">
                  <div className="font-medium text-gray-900 dark:text-white truncate">
                    {service.name}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {service.health.responseTime.toFixed(0)}ms avg response time
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 ml-4">
                <div className="text-right">
                  <div className="text-sm font-medium text-gray-900 dark:text-white">
                    {service.health.uptime.toFixed(1)}%
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">uptime</div>
                </div>
                <div className="w-24">
                  <ProgressBar
                    value={service.health.uptime}
                    max={100}
                    color={service.health.status === 'healthy' ? 'success' : service.health.status === 'degraded' ? 'warning' : 'danger'}
                    size="sm"
                    showValue={false}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  )
}
