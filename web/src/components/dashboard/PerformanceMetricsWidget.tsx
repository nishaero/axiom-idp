import React, { useMemo } from 'react'
import { LineChart, BarChart, AreaChart } from '@/components/dashboard'
import { Card, MetricDisplay, StatusBadge } from '@/components/dashboard'
import { useDashboardStore, useRealTimeUpdates } from '@/hooks/useDashboardStore'

export default function PerformanceMetricsWidget() {
  const { performanceData, setPerformanceData } = useDashboardStore()
  const { serviceHealth, teamActivity } = useRealTimeUpdates()

  // Calculate derived metrics
  const metrics = useMemo(() => {
    const latest = performanceData[performanceData.length - 1]
    const avgResponseTime =
      performanceData.reduce((sum, p) => sum + p.responseTime, 0) / performanceData.length
    const avgThroughput =
      performanceData.reduce((sum, p) => sum + p.throughput, 0) / performanceData.length
    const maxConcurrent = Math.max(...performanceData.map((p) => p.concurrentUsers))

    return {
      currentResponseTime: latest.responseTime.toFixed(0),
      avgResponseTime: Math.round(avgResponseTime),
      currentThroughput: Math.round(avgThroughput).toLocaleString(),
      maxConcurrentUsers: maxConcurrent,
    }
  }, [performanceData])

  // Prepare chart data
  const responseTimeData = useMemo(
    () =>
      performanceData.map((p) => ({
        label: p.label,
        responseTime: p.responseTime,
      })),
    [performanceData]
  )

  const throughputData = useMemo(
    () =>
      performanceData.map((p) => ({
        label: p.label,
        throughput: p.throughput,
      })),
    [performanceData]
  )

  const errorRateData = useMemo(
    () =>
      performanceData.map((p) => ({
        label: p.label,
        errorRate: p.errorRate * 100,
      })),
    [performanceData]
  )

  return (
    <Card title="API Performance Metrics" onRefresh={() => setPerformanceData([...performanceData])}>
      <div className="grid grid-cols-4 gap-4 mb-6">
        <MetricDisplay
          value={`${metrics.currentResponseTime}ms`}
          label="Current Response Time"
          trend={-2.5}
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
          value={`~${metrics.currentThroughput}/min`}
          label="Throughput"
          trend={8.3}
          trendLabel="growth"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
              />
            </svg>
          }
          color="success"
        />
        <MetricDisplay
          value={`${(parseFloat(metrics.avgResponseTime) / 1000).toFixed(2)}s avg`}
          label="Average Response Time"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 012 2h2a2 2 0 012-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
              />
            </svg>
          }
          color="primary"
        />
        <MetricDisplay
          value={metrics.maxConcurrentUsers.toString()}
          label="Max Concurrent Users"
          trend={12.5}
          trendLabel="peak users"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
              />
            </svg>
          }
          color="primary"
        />
      </div>

      <div className="grid grid-cols-2 gap-6 mb-6">
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">Response Time</h4>
          <LineChart
            data={responseTimeData}
            dataKey="responseTime"
            label="Response Time (ms)"
            color="#0ea5e9"
            height={150}
          />
        </div>
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">Throughput</h4>
          <BarChart
            data={throughputData}
            dataKey="throughput"
            label="Requests/min"
            height={150}
          />
        </div>
      </div>

      <div className="space-y-2">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">Error Rate</h4>
        <AreaChart
          data={errorRateData}
          dataKey="errorRate"
          label="Error Rate (%)"
          color="#ef4444"
          fill="rgba(239, 68, 68, 0.2)"
          height={120}
        />
      </div>
    </Card>
  )
}
