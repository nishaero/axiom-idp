interface ProgressBarProps {
  value: number
  max?: number
  label?: string
  color?: 'primary' | 'success' | 'warning' | 'danger'
  showValue?: boolean
  size?: 'sm' | 'md' | 'lg'
  animate?: boolean
}

export default function ProgressBar({
  value,
  max = 100,
  label,
  color = 'primary',
  showValue = true,
  size = 'md',
  animate = true,
}: ProgressBarProps) {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100)

  const colorClasses: Record<string, string> = {
    primary: 'bg-primary-600 dark:bg-primary-500',
    success: 'bg-green-600 dark:bg-green-500',
    warning: 'bg-yellow-600 dark:bg-yellow-500',
    danger: 'bg-red-600 dark:bg-red-500',
  }

  const sizeClasses: Record<string, string> = {
    sm: 'h-1.5',
    md: 'h-2.5',
    lg: 'h-4',
  }

  return (
    <div className="w-full">
      {label && (
        <div className="flex items-center justify-between mb-1">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {label}
          </span>
          {showValue && (
            <span className="text-sm font-semibold text-gray-900 dark:text-white">
              {percentage.toFixed(0)}%
            </span>
          )}
        </div>
      )}
      <div
        className={`w-full bg-gray-200 dark:bg-dark-700 rounded-full overflow-hidden ${sizeClasses[size]}`}
      >
        <div
          className={`${colorClasses[color]} rounded-full transition-all duration-500 ${
            animate ? 'animate-pulse' : ''
          }`}
          style={{ width: `${percentage}%` }}
        />
      </div>
      {label && showValue && !animate && (
        <div className="flex items-center justify-between mt-1">
          <span className="text-sm text-gray-600 dark:text-gray-400">
            {value} / {max}
          </span>
        </div>
      )}
    </div>
  )
}

interface ProgressRingProps {
  value: number
  max?: number
  size?: 'sm' | 'md' | 'lg'
  color?: string
  showLabel?: boolean
  label?: string
}

export function ProgressRing({
  value,
  max = 100,
  size = 'md',
  color = '#0ea5e9',
  showLabel = true,
  label,
}: ProgressRingProps) {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100)
  const radius = size === 'sm' ? 18 : size === 'lg' ? 35 : 27
  const strokeWidth = size === 'sm' ? 4 : size === 'lg' ? 6 : 5
  const circumference = 2 * Math.PI * radius

  const sizeClasses = {
    sm: 'h-12 w-12',
    md: 'h-20 w-20',
    lg: 'h-28 w-28',
  }

  return (
    <div className="flex flex-col items-center">
      <svg
        className={`${sizeClasses[size]} transform -rotate-90`}
        viewBox={`0 0 ${radius * 2 + strokeWidth} ${radius * 2 + strokeWidth}`}
      >
        <circle
          className="text-gray-200 dark:text-dark-700"
          strokeWidth={strokeWidth}
          stroke="currentColor"
          fill="transparent"
          r={radius}
          cx={radius + strokeWidth / 2}
          cy={radius + strokeWidth / 2}
        />
        <circle
          className="transition-all duration-500 ease-out"
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={circumference - (circumference * percentage) / 100}
          strokeLinecap="round"
          stroke={color}
          fill="transparent"
          r={radius}
          cx={radius + strokeWidth / 2}
          cy={radius + strokeWidth / 2}
        />
      </svg>
      {showLabel && (
        <div className="mt-2 text-center">
          <div className="text-xl font-bold text-gray-900 dark:text-white">
            {percentage.toFixed(0)}%
          </div>
          {label && (
            <div className="text-sm text-gray-600 dark:text-gray-400">{label}</div>
          )}
        </div>
      )}
    </div>
  )
}
