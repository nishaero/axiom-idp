import React from 'react'
import {
  AreaChart as RechartsAreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'

interface AreaChartProps {
  data: Record<string, unknown>[]
  dataKey: string
  label?: string
  color?: string
  fill?: string
  unit?: string
  showGrid?: boolean
  height?: number | string
}

export default function AreaChart({
  data,
  dataKey,
  label,
  color = '#0ea5e9',
  fill = 'rgba(14, 165, 233, 0.2)',
  unit,
  showGrid = true,
  height = 200,
}: AreaChartProps) {
  const CustomTooltip = ({ active, payload, label: tooltipLabel }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white dark:bg-dark-800 p-3 rounded-lg shadow-lg border border-gray-200 dark:border-dark-700">
          <p className="text-sm font-medium text-gray-900 dark:text-white mb-1">
            {tooltipLabel}
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {label || dataKey}: <span className="font-semibold">{payload[0].value}</span>
            {unit && <span className="text-xs text-gray-500"> {unit}</span>}
          </p>
        </div>
      )
    }
    return null
  }

  return (
    <div className="w-full" style={{ height }}>
      <ResponsiveContainer>
        <RechartsAreaChart
          data={data}
          margin={{ top: 10, right: 10, left: 0, bottom: 0 }}
        >
          {showGrid && (
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="#e5e7eb"
              className="dark:stroke-dark-700"
            />
          )}
          <XAxis
            dataKey="label"
            tick={{ fontSize: 12, fill: '#6b7280' }}
            tickLine={false}
            axisLine={{ stroke: '#e5e7eb', className: 'dark:stroke-dark-700' }}
          />
          <YAxis
            tick={{ fontSize: 12, fill: '#6b7280' }}
            tickLine={false}
            axisLine={{ stroke: '#e5e7eb', className: 'dark:stroke-dark-700' }}
            tickFormatter={(value) => unit ? `${value}${unit}` : String(value)}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend wrapperStyle={{ fontSize: '12px' }} />
          <Area
            type="monotone"
            dataKey={dataKey}
            name={label || dataKey}
            stroke={color}
            strokeWidth={2}
            fill={fill}
            fillOpacity={1}
            activeDot={{ r: 6, fill: color, strokeWidth: 2 }}
            animationDuration={500}
          />
        </RechartsAreaChart>
      </ResponsiveContainer>
    </div>
  )
}
