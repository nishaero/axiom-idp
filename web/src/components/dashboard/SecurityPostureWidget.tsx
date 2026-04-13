import React, { useMemo } from 'react'
import { Card, MetricDisplay, StatusBadge, ProgressBar, ProgressRing } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'

interface VulnerabilityData {
  level: 'critical' | 'high' | 'medium' | 'low'
  count: number
  label: string
  color: string
}

export default function SecurityPostureWidget() {
  const { refreshServiceHealth } = useDashboardStore()

  // Mock security data
  const securityMetrics = useMemo(
    () => ({
      overallScore: 87,
      vulnerabilities: {
        critical: 2,
        high: 8,
        medium: 24,
        low: 47,
      },
      compliance: {
        passed: 32,
        failed: 3,
        total: 35,
      },
      lastScan: Date.now() - 3600000, // 1 hour ago
    }),
    []
  )

  const vulnerabilityData: VulnerabilityData[] = useMemo(
    () => [
      { level: 'critical', count: securityMetrics.vulnerabilities.critical, label: 'Critical', color: '#ef4444' },
      { level: 'high', count: securityMetrics.vulnerabilities.high, label: 'High', color: '#f97316' },
      { level: 'medium', count: securityMetrics.vulnerabilities.medium, label: 'Medium', color: '#f59e0b' },
      { level: 'low', count: securityMetrics.vulnerabilities.low, label: 'Low', color: '#10b981' },
    ],
    [securityMetrics]
  )

  const complianceRate = (securityMetrics.compliance.passed / securityMetrics.compliance.total) * 100

  const formatTimeAgo = (timestamp: number) => {
    const seconds = Math.floor((Date.now() - timestamp) / 1000)
    if (seconds < 60) return 'Just now'
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}m ago`
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    return `${days}d ago`
  }

  return (
    <Card title="Security Posture">
      <div className="grid grid-cols-2 gap-4 mb-6">
        <MetricDisplay
          value={`${securityMetrics.overallScore}/100`}
          label="Security Score"
          trend={3.2}
          trendLabel="improvement"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
              />
            </svg>
          }
          color={securityMetrics.overallScore >= 80 ? 'success' : securityMetrics.overallScore >= 60 ? 'warning' : 'danger'}
        />
        <MetricDisplay
          value={`${securityMetrics.compliance.passed}/${securityMetrics.compliance.total}`}
          label="Compliance Pass Rate"
          trend={(1 - complianceRate / 100) * 100}
          trendLabel={`${(100 - complianceRate).toFixed(0)} failures`}
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
          color={complianceRate >= 90 ? 'success' : complianceRate >= 70 ? 'warning' : 'danger'}
        />
      </div>

      <div className="flex justify-center mb-6">
        <ProgressRing
          value={securityMetrics.overallScore}
          max={100}
          size="lg"
          color={
            securityMetrics.overallScore >= 80 ? '#10b981' : securityMetrics.overallScore >= 60 ? '#f59e0b' : '#ef4444'
          }
          showLabel={true}
          label="Security Score"
        />
      </div>

      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">Vulnerabilities by Severity</h4>
        <div className="space-y-4">
          {vulnerabilityData.map((item) => (
            <div key={item.level} className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <StatusBadge status={item.level === 'critical' ? 'error' : item.level === 'high' ? 'warning' : item.level === 'medium' ? 'degraded' : 'healthy'} size="sm" />
                  <span className="text-sm text-gray-600 dark:text-gray-400">{item.label} Severity</span>
                </div>
                <span className="text-lg font-semibold text-gray-900 dark:text-white">
                  {item.count}
                </span>
              </div>
              <ProgressBar
                value={item.count}
                max={Math.max(...vulnerabilityData.map((v) => v.count))}
                color={item.color}
                size="sm"
                showValue={false}
              />
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-gray-200 dark:border-dark-700 pt-4">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Compliance Status</span>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-green-500 rounded-full" />
            <span className="text-xs text-green-600 dark:text-green-400">{securityMetrics.compliance.passed} Passed</span>
            <div className="w-3 h-3 bg-red-500 rounded-full" />
            <span className="text-xs text-red-600 dark:text-red-400">{securityMetrics.compliance.failed} Failed</span>
          </div>
        </div>
        <div className="flex justify-center">
          <div className="w-48 h-2 bg-gray-200 dark:bg-dark-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-green-500 rounded-full transition-all duration-500"
              style={{ width: `${complianceRate}%` }}
            />
          </div>
        </div>
        <div className="mt-2 text-center text-sm text-gray-600 dark:text-gray-400">
          {complianceRate.toFixed(1)}% compliance rate
        </div>
      </div>

      <div className="mt-6 pt-4 border-t border-gray-200 dark:border-dark-700 flex items-center justify-between text-sm">
        <span className="text-gray-600 dark:text-gray-400">Last security scan:</span>
        <span className="text-sm font-medium text-gray-900 dark:text-white">
          {formatTimeAgo(securityMetrics.lastScan)}
        </span>
      </div>
    </Card>
  )
}
