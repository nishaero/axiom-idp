interface StatusBadgeProps {
  status: 'healthy' | 'degraded' | 'unhealthy' | 'error' | 'running' | 'stopped' | 'pending' | 'completed' | 'unknown'
  label?: string
  size?: 'sm' | 'md' | 'lg'
  showIcon?: boolean
}

export default function StatusBadge({
  status,
  label,
  size = 'md',
  showIcon = true,
}: StatusBadgeProps) {
  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
    lg: 'px-3 py-1.5 text-base',
  }

  const statusColors: Record<string, string> = {
    healthy: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
    degraded: 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200',
    unhealthy: 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
    error: 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
    running: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
    stopped: 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200',
    pending: 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200',
    completed: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
    unknown: 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200',
  }

  const iconColors: Record<string, string> = {
    healthy: 'text-green-500',
    degraded: 'text-yellow-500',
    unhealthy: 'text-red-500',
    error: 'text-red-500',
    running: 'text-green-500',
    stopped: 'text-gray-500',
    pending: 'text-blue-500',
    completed: 'text-green-500',
    unknown: 'text-gray-500',
  }

  const getStatusIcon = () => {
    switch (status) {
      case 'healthy':
      case 'running':
      case 'completed':
        return (
          <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
              clipRule="evenodd"
            />
          </svg>
        )
      case 'degraded':
        return (
          <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
        )
      case 'unhealthy':
      case 'error':
        return (
          <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
              clipRule="evenodd"
            />
          </svg>
        )
      case 'stopped':
      case 'unknown':
        return (
          <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
              clipRule="evenodd"
            />
          </svg>
        )
      case 'pending':
        return (
          <svg className="h-4 w-4 animate-spin rounded-full" fill="none" viewBox="0 0 24 24">
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        )
      default:
        return null
    }
  }

  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full font-medium ${sizeClasses[size]} ${statusColors[status] || statusColors.unknown}`}>
      {showIcon && (
        <span className={iconColors[status] || iconColors.unknown}>
          {status === 'pending' ? null : getStatusIcon()}
        </span>
      )}
      <span>{label || status.charAt(0).toUpperCase() + status.slice(1)}</span>
    </span>
  )
}
