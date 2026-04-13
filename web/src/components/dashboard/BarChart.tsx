import React from 'react'
import {
  BarChart as RechartsBarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Cell,
} from 'recharts'

interface BarChartProps {
  data: Record<string, unknown>[]
  dataKey: string
  label?: string
  colors?: string[]
  unit?: string
  showGrid?: boolean
  height?: number | string
  orientation?: 'vertical' | 'horizontal'
}

export default function BarChart({
  data,
  dataKey,
  label,
  colors = ['#0ea5e9', '#8b5cf6', '#10b981', '#f59e0b', '#ef4444'],
  unit,
  showGrid = true,
  height = 200,
  orientation = 'vertical',
}: BarChartProps) {
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

  const renderCells = () => {
    return data.map((entry, index) => (
      <Cell key={`cell-${index}`} fill={colors[index % colors.length]} />
    ))
  }

  return (
    <div className="w-full" style={{ height }}>
      <ResponsiveContainer>
        <RechartsBarChart
          data={data}
          layout={orientation === 'horizontal' ? 'horizontal' : 'vertical'}
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
            dataKey={orientation === 'horizontal' ? 'name' : 'label'}
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
          <Bar
            dataKey={dataKey}
            name={label || dataKey}
            radius={orientation === 'horizontal' ? [0, 4, 4, 0] : [4, 4, 0, 0]}
          >
            {renderCells()}
          </Bar>
        </RechartsBarChart>
      </ResponsiveContainer>
    </div>
  )
}
