import React from 'react'
import {
  PieChart as RechartsPieChart,
  Pie,
  Cell,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'

interface PieChartProps {
  data: { name: string; value: number; color?: string }[]
  dataKey?: string
  labelKey?: string
  colors?: string[]
  showLabel?: boolean
  showPercentage?: boolean
  height?: number | string
  cx?: number
  cy?: number
  innerRadius?: number
  outerRadius?: number
}

export default function PieChart({
  data,
  dataKey = 'value',
  labelKey = 'name',
  colors = [
    '#0ea5e9',
    '#8b5cf6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#f97316',
    '#6366f1',
    '#ec4899',
  ],
  showLabel = true,
  showPercentage = true,
  height = 200,
  cx,
  cy,
  innerRadius = 0,
  outerRadius = 100,
}: PieChartProps) {
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const dataItem = payload[0].payload
      const value = dataItem[dataKey]
      const name = dataItem[labelKey]
      const total = data.reduce((sum, item) => sum + item[dataKey as keyof typeof item], 0)
      const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0

      return (
        <div className="bg-white dark:bg-dark-800 p-3 rounded-lg shadow-lg border border-gray-200 dark:border-dark-700">
          <p className="text-sm font-medium text-gray-900 dark:text-white mb-1">
            {name}
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Value: <span className="font-semibold">{value}</span>
          </p>
          {showPercentage && (
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Share: <span className="font-semibold">{percentage}%</span>
            </p>
          )}
        </div>
      )
    }
    return null
  }

  const CustomLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, percent }: any) => {
    if (!showLabel) return null

    const RADIAN = Math.PI / 180
    const x = cx + (innerRadius + (outerRadius - innerRadius) * 0.5) * Math.cos(-midAngle * RADIAN)
    const y = cy + (innerRadius + (outerRadius - innerRadius) * 0.5) * Math.sin(-midAngle * RADIAN)

    const total = data.reduce((sum, item) => sum + item[dataKey as keyof typeof item], 0)
    const value = data[Math.floor((midAngle + 180) / (360 / data.length))][dataKey]
    const percentage = total > 0 ? ((value / total) * 100).toFixed(0) : 0

    return (
      <text
        x={x}
        y={y}
        fill="#6b7280"
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={11}
      >
        {percentage}%
      </text>
    )
  }

  const renderCells = () => {
    return data.map((entry, index) => (
      <Cell key={`cell-${index}`} fill={entry.color || colors[index % colors.length]} />
    ))
  }

  return (
    <div className="w-full" style={{ height }}>
      <ResponsiveContainer>
        <RechartsPieChart margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
          <Pie
            data={data}
            cx={cx}
            cy={cy}
            innerRadius={innerRadius}
            outerRadius={outerRadius}
            dataKey={dataKey}
            nameKey={labelKey}
            label={showLabel ? CustomLabel : false}
            labelLine={false}
          >
            {renderCells()}
          </Pie>
          <Tooltip content={<CustomTooltip />} />
          {showLabel && (
            <Legend
              layout="horizontal"
              verticalAlign="bottom"
              align="center"
              wrapperStyle={{ fontSize: '12px' }}
            />
          )}
        </RechartsPieChart>
      </ResponsiveContainer>
    </div>
  )
}
