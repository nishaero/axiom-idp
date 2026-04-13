import React from 'react'

interface CardProps {
  children: React.ReactNode
  className?: string
  title?: string
  subtitle?: string
  actions?: React.ReactNode
  isLoading?: boolean
  error?: string | null
  onRefresh?: () => void
}

export default function Card({
  children,
  className = '',
  title,
  subtitle,
  actions,
  isLoading = false,
  error,
  onRefresh,
}: CardProps) {
  return (
    <div className={`bg-white dark:bg-dark-800 rounded-lg shadow border border-gray-200 dark:border-dark-700 ${className}`}>
      {(title || actions) && (
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-dark-700">
          <div>
            {title && (
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                {title}
              </h3>
            )}
            {subtitle && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                {subtitle}
              </p>
            )}
          </div>
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}

      {isLoading ? (
        <div className="p-6 flex items-center justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          <span className="ml-3 text-gray-600 dark:text-gray-400">Loading...</span>
        </div>
      ) : error ? (
        <div className="p-6">
          <div className="flex items-start gap-3 text-red-600 dark:text-red-400">
            <svg className="h-5 w-5 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
            <div className="flex-1">
              <p className="font-medium">Error loading data</p>
              <p className="text-sm mt-1">{error}</p>
            </div>
            {onRefresh && (
              <button
                onClick={onRefresh}
                className="p-1 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                aria-label="Retry"
              >
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                  />
                </svg>
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="p-4">{children}</div>
      )}
    </div>
  )
}

interface MetricDisplayProps {
  value: string | number
  label: string
  trend?: number
  trendLabel?: string
  icon?: React.ReactNode
  color?: 'primary' | 'success' | 'warning' | 'danger'
  className?: string
}

export function MetricDisplay({
  value,
  label,
  trend,
  trendLabel,
  icon,
  color = 'primary',
  className = '',
}: MetricDisplayProps) {
  const colorClasses = {
    primary: 'text-primary-600 dark:text-primary-400',
    success: 'text-green-600 dark:text-green-400',
    warning: 'text-yellow-600 dark:text-yellow-400',
    danger: 'text-red-600 dark:text-red-400',
  }

  const iconColorClasses = {
    primary: 'bg-primary-100 dark:bg-primary-900 text-primary-600 dark:text-primary-400',
    success: 'bg-green-100 dark:bg-green-900 text-green-600 dark:text-green-400',
    warning: 'bg-yellow-100 dark:bg-yellow-900 text-yellow-600 dark:text-yellow-400',
    danger: 'bg-red-100 dark:bg-red-900 text-red-600 dark:text-red-400',
  }

  return (
    <div className={`flex items-start gap-3 ${className}`}>
      {icon && <div className={`p-2 rounded-lg ${iconColorClasses[color]}`}>{icon}</div>}
      <div className="flex-1 min-w-0">
        <div className={`text-2xl font-bold ${colorClasses[color]}`}>
          {value}
        </div>
        <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">{label}</div>
        {trend !== undefined && (
          <div className="flex items-center gap-1 mt-2">
            <span
              className={`text-sm font-medium ${
                trend > 0 ? 'text-green-600 dark:text-green-400' : trend < 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-400'
              }`}
            >
              {trend > 0 ? '↑' : trend < 0 ? '↓'} {Math.abs(trend)}%
            </span>
            {trendLabel && (
              <span className="text-xs text-gray-500 dark:text-gray-500">{trendLabel}</span>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
