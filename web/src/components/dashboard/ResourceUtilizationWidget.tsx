import React, { useMemo } from 'react'
import { Card, MetricDisplay, ProgressBar, HeatMap, SimpleHeatMap } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'

interface ResourceHost {
  name: string
  cpu: number
  memory: number
  storage: number
  status: 'healthy' | 'degraded' | 'unhealthy'
}

export default function ResourceUtilizationWidget() {
  const { refreshServiceHealth } = useDashboardStore()

  // Mock resource utilization data
  const hosts = useMemo<ResourceHost[]>(
    () => [
      { name: 'prod-server-01', cpu: 72, memory: 65, storage: 45, status: 'healthy' },
      { name: 'prod-server-02', cpu: 58, memory: 71, storage: 52, status: 'healthy' },
      { name: 'prod-server-03', cpu: 85, memory: 78, storage: 61, status: 'degraded' },
      { name: 'staging-server-01', cpu: 42, memory: 38, storage: 30, status: 'healthy' },
      { name: 'staging-server-02', cpu: 35, memory: 40, storage: 28, status: 'healthy' },
      { name: 'dev-server-01', cpu: 28, memory: 32, storage: 25, status: 'healthy' },
    ],
    []
  )

  const metrics = useMemo(() => {
    const avgCpu = hosts.reduce((sum, h) => sum + h.cpu, 0) / hosts.length
    const avgMemory = hosts.reduce((sum, h) => sum + h.memory, 0) / hosts.length
    const avgStorage = hosts.reduce((sum, h) => sum + h.storage, 0) / hosts.length
    const totalCpuCapacity = hosts.length * 100
    const totalMemoryCapacity = hosts.length * 100

    return {
      avgCpu: avgCpu.toFixed(1),
      avgMemory: avgMemory.toFixed(1),
      avgStorage: avgStorage.toFixed(1),
      totalCpuCapacity,
      totalMemoryCapacity,
      hosts,
      highUtilizationHosts: hosts.filter((h) => h.cpu > 80 || h.memory > 80).length,
    }
  }, [hosts])

  // Prepare heatmap data for time-based utilization
  const heatMapData = useMemo(() => {
    const timeSlots = ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00']
    const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

    const data = []
    timeSlots.forEach((time, timeIndex) => {
      days.forEach((day, dayIndex) => {
        const baseValue = Math.random() * 60 + 20
        const timeMultiplier = timeIndex >= 2 && timeIndex <= 4 ? 1.5 : 0.8
        const dayMultiplier = dayIndex >= 1 && dayIndex <= 4 ? 1.3 : 0.9

        data.push({
          x: day,
          y: time,
          value: Math.min(100, baseValue * timeMultiplier * dayMultiplier),
        })
      })
    })
    return data
  }, [])

  const formatTime = (timestamp: number) => {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <Card title="Resource Utilization">
      <div className="grid grid-cols-3 gap-4 mb-6">
        <MetricDisplay
          value={`${metrics.avgCpu}%`}
          label="CPU Usage"
          trend={metrics.avgCpu > 70 ? -5.2 : 3.1}
          trendLabel={metrics.avgCpu > 70 ? 'reducing' : 'stable'}
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
              />
            </svg>
          }
          color={parseFloat(metrics.avgCpu) > 80 ? 'danger' : parseFloat(metrics.avgCpu) > 60 ? 'warning' : 'success'}
        />
        <MetricDisplay
          value={`${metrics.avgMemory}%`}
          label="Memory Usage"
          trend={parseFloat(metrics.avgMemory) > 80 ? -8.3 : 1.5}
          trendLabel={parseFloat(metrics.avgMemory) > 80 ? 'stabilizing' : 'stable'}
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
              />
            </svg>
          }
          color={parseFloat(metrics.avgMemory) > 80 ? 'danger' : parseFloat(metrics.avgMemory) > 60 ? 'warning' : 'success'}
        />
        <MetricDisplay
          value={`${metrics.avgStorage}%`}
          label="Storage Usage"
          trend={2.3}
          trendLabel="growth"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
              />
            </svg>
          }
          color={parseFloat(metrics.avgStorage) > 80 ? 'danger' : parseFloat(metrics.avgStorage) > 60 ? 'warning' : 'success'}
        />
      </div>

      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">Host Utilization</h4>
        <div className="space-y-3">
          {hosts.map((host) => (
            <div
              key={host.name}
              className="flex items-center gap-4 p-3 bg-gray-50 dark:bg-dark-700/50 rounded-lg"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <StatusBadge
                    status={host.status}
                    size="sm"
                    showIcon={true}
                  />
                  <span className="font-medium text-gray-900 dark:text-white truncate">
                    {host.name}
                  </span>
                </div>
                <div className="grid grid-cols-3 gap-4 text-xs text-gray-500 dark:text-gray-400">
                  <div>CPU: {host.cpu}%</div>
                  <div>Memory: {host.memory}%</div>
                  <div>Storage: {host.storage}%</div>
                </div>
              </div>
              <div className="w-48 space-y-2">
                <ProgressBar
                  value={host.cpu}
                  color={host.cpu > 80 ? 'danger' : host.cpu > 60 ? 'warning' : 'success'}
                  size="sm"
                  showValue={false}
                />
                <ProgressBar
                  value={host.memory}
                  color={host.memory > 80 ? 'danger' : host.memory > 60 ? 'warning' : 'success'}
                  size="sm"
                  showValue={false}
                />
                <ProgressBar
                  value={host.storage}
                  color={host.storage > 80 ? 'danger' : host.storage > 60 ? 'warning' : 'success'}
                  size="sm"
                  showValue={false}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-gray-200 dark:border-dark-700 pt-4">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">
          Weekly Utilization Heatmap
        </h4>
        <SimpleHeatMap
          data={heatMapData}
          colorScale={['#f7fcfd', '#e0f3f8', '#bae6f3', '#7ccce9', '#3db3df', '#1a8fcf', '#0a6ca9', '#044f84', '#02385e']}
          height={200}
        />
        <div className="mt-4 flex items-center justify-center gap-6">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-gradient-to-r from-blue-50 to-blue-900"></div>
            <span className="text-xs text-gray-500 dark:text-gray-400">Low</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-gradient-to-r from-blue-400 to-blue-800"></div>
            <span className="text-xs text-gray-500 dark:text-gray-400">Medium</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-gradient-to-r from-blue-300 to-blue-600"></div>
            <span className="text-xs text-gray-500 dark:text-gray-400">High</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded bg-gradient-to-r from-blue-200 to-blue-500"></div>
            <span className="text-xs text-gray-500 dark:text-gray-400">Critical</span>
          </div>
        </div>
      </div>

      {metrics.highUtilizationHosts > 0 && (
        <div className="mt-6 p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
          <div className="flex items-start gap-3">
            <svg className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
            <div>
              <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                High Utilization Alert
              </p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
                {metrics.highUtilizationHosts} host{metrics.highUtilizationHosts > 1 ? 's' : ''} are using over 80% CPU or memory. Consider scaling or load balancing.
              </p>
            </div>
          </div>
        </div>
      )}
    </Card>
  )
}
