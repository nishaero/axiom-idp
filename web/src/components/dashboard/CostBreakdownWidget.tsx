import React, { useMemo } from 'react'
import { PieChart, Card, MetricDisplay, ProgressBar } from '@/components/dashboard'
import { useDashboardStore } from '@/hooks/useDashboardStore'

interface CostBreakdownData {
  category: string
  amount: number
  color: string
}

export default function CostBreakdownWidget() {
  const { refreshServiceHealth } = useDashboardStore()

  // Mock cost data
  const costData = useMemo<CostBreakdownData[]>(
    () => [
      { category: 'Compute', amount: 2450, color: '#0ea5e9' },
      { category: 'Storage', amount: 890, color: '#8b5cf6' },
      { category: 'Network', amount: 420, color: '#10b981' },
      { category: 'Database', amount: 1200, color: '#f59e0b' },
      { category: 'Security', amount: 350, color: '#ef4444' },
      { category: 'Monitoring', amount: 180, color: '#6366f1' },
    ],
    []
  )

  const metrics = useMemo(() => {
    const total = costData.reduce((sum, item) => sum + item.amount, 0)
    const monthlyGrowth = 8.5
    const yearlyProjection = total * 12 + (total * 12 * 0.085)

    return {
      totalMonthly: total,
      monthlyGrowth,
      yearlyProjection,
      largestCategory: costData.reduce((max, item) => (item.amount > max.amount ? item : max)),
      savings: 1240,
      savingsPercentage: 12.5,
    }
  }, [costData])

  const costDataWithPercentage = useMemo(() => {
    const total = metrics.totalMonthly
    return costData.map((item) => ({
      name: item.category,
      value: item.amount,
      color: item.color,
      percentage: ((item.amount / total) * 100).toFixed(1),
    }))
  }, [costData, metrics.totalMonthly])

  const formatCurrency = (amount: number) =>
    new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount)

  return (
    <Card title="Cost Breakdown">
      <div className="grid grid-cols-2 gap-4 mb-6">
        <MetricDisplay
          value={formatCurrency(metrics.totalMonthly)}
          label="Monthly Spend"
          trend={metrics.monthlyGrowth}
          trendLabel="vs last month"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          }
          color="primary"
        />
        <MetricDisplay
          value={formatCurrency(metrics.yearlyProjection)}
          label="Projected Annual"
          trend={metrics.monthlyGrowth * 12}
          trendLabel="projected growth"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"
              />
            </svg>
          }
          color="warning"
        />
        <MetricDisplay
          value={formatCurrency(metrics.savings)}
          label="Estimated Savings"
          trend={metrics.savingsPercentage}
          trendLabel="optimization"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          }
          color="success"
        />
        <MetricDisplay
          value={metrics.largestCategory.category}
          label="Largest Expense"
          icon={
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
              />
            </svg>
          }
          color="danger"
        />
      </div>

      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">
          Cost by Category
        </h4>
        <div className="grid grid-cols-2 gap-4">
          {costData.map((item) => (
            <div key={item.category} className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">{item.category}</span>
                <span className="text-sm font-semibold text-gray-900 dark:text-white">
                  {formatCurrency(item.amount)}
                </span>
              </div>
              <ProgressBar
                value={item.amount}
                max={Math.max(...costData.map((c) => c.amount))}
                color="primary"
                size="sm"
                showValue={false}
              />
              <div className="flex items-center justify-end">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: item.color }}
                />
                <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                  {costDataWithPercentage.find((c) => c.category === item.category)?.percentage}%
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="flex justify-center">
        <div className="w-48 h-48">
          <PieChart
            data={costDataWithPercentage}
            showLabel={true}
            showPercentage={true}
            height={192}
          />
        </div>
      </div>

      <div className="mt-6 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
        <div className="flex items-start gap-3">
          <svg className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
              clipRule="evenodd"
            />
          </svg>
          <div>
            <p className="text-sm font-medium text-green-800 dark:text-green-200">
              Cost Optimization Tips
            </p>
            <p className="text-xs text-green-700 dark:text-green-300 mt-1">
              Consider right-sizing your compute resources. Estimated additional savings: {formatCurrency(metrics.savings * 0.5)}
            </p>
          </div>
        </div>
      </div>
    </Card>
  )
}
