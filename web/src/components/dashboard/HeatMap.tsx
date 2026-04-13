import React from 'react'
import {
  ComposableMap,
  Geographies,
  Geography,
  Sphere,
  Graticule,
} from 'react-map-gl'
import {
  ResponsiveContainer,
  HeatMap as RechartsHeatMap,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts'

interface HeatMapData {
  x: number | string
  y: number | string
  value: number
}

interface HeatMapProps {
  data: HeatMapData[]
  xKey?: string
  yKey?: string
  valueKey?: string
  title?: string
  showTooltip?: boolean
  colorScale?: string[]
  height?: number | string
}

export default function HeatMap({
  data,
  xKey = 'x',
  yKey = 'y',
  valueKey = 'value',
  title,
  showTooltip = true,
  colorScale = ['#f7fcfd', '#e0f3f8', '#bae6f3', '#7ccce9', '#3db3df', '#1a8fcf', '#0a6ca9', '#044f84', '#02385e'],
  height = 300,
}: HeatMapProps) {
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const dataItem = payload[0].payload
      return (
        <div className="bg-white dark:bg-dark-800 p-3 rounded-lg shadow-lg border border-gray-200 dark:border-dark-700">
          <p className="text-sm font-medium text-gray-900 dark:text-white">
            {dataItem[xKey]} - {dataItem[yKey]}
          </p>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Value: <span className="font-semibold">{dataItem[valueKey]}</span>
          </p>
        </div>
      )
    }
    return null
  }

  const getColor = (value: number, max: number) => {
    const index = Math.min(
      Math.floor((value / max) * (colorScale.length - 1)),
      colorScale.length - 1
    )
    return colorScale[index]
  }

  return (
    <div className="w-full" style={{ height }}>
      {title && (
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {title}
        </h3>
      )}
      <ResponsiveContainer>
        <RechartsHeatMap
          data={data}
          xAxis={{
            dataKey: xKey,
            type: 'category',
            tick: { fontSize: 11 },
          }}
          yAxis={{
            dataKey: yKey,
            type: 'category',
            tick: { fontSize: 11 },
          }}
          shape="rect"
          tooltip={<CustomTooltip />}
        >
          {data.map((entry, index) => (
            <RechartsHeatMap.Cell
              key={`cell-${index}`}
              x={entry[xKey]}
              y={entry[yKey]}
              rx={4}
              ry={4}
              fill={entry[valueKey as keyof typeof entry]}
            />
          ))}
        </RechartsHeatMap>
      </ResponsiveContainer>
      {/* Custom cell rendering for color */}
      <div className="mt-2 text-center text-xs text-gray-600 dark:text-gray-400">
        Resource Utilization Heatmap
      </div>
    </div>
  )
}

// Simple grid heatmap without Recharts dependency
interface SimpleHeatMapProps {
  data: HeatMapData[]
  xKey?: string
  yKey?: string
  valueKey?: string
  colorScale?: string[]
  height?: number | string
}

export function SimpleHeatMap({
  data,
  xKey = 'x',
  yKey = 'y',
  valueKey = 'value',
  colorScale = ['#f7fcfd', '#e0f3f8', '#bae6f3', '#7ccce9', '#3db3df', '#1a8fcf', '#0a6ca9', '#044f84', '#02385e'],
  height = 300,
}: SimpleHeatMapProps) {
  // Get unique values
  const xValues = [...new Set(data.map((d) => d[xKey]))]
  const yValues = [...new Set(data.map((d) => d[yKey]))]

  // Find max value for color scaling
  const maxValue = Math.max(...data.map((d) => d[valueKey as keyof typeof d]))

  // Create a lookup map
  const valueMap = new Map<string, number>()
  data.forEach((d) => {
    const key = `${d[xKey]}-${d[yKey]}`
    valueMap.set(key, d[valueKey as keyof typeof d])
  })

  const getColor = (value: number) => {
    if (maxValue === 0) return colorScale[0]
    const index = Math.min(
      Math.floor((value / maxValue) * (colorScale.length - 1)),
      colorScale.length - 1
    )
    return colorScale[index]
  }

  const getXIndex = (x: any) => xValues.indexOf(x)
  const getYIndex = (y: any) => yValues.indexOf(y)

  const cellWidth = 50 // px
  const cellHeight = 30 // px

  return (
    <div className="w-full" style={{ height }}>
      <div className="flex">
        <div className="flex flex-col justify-center pr-2">
          {yValues.map((y) => (
            <div
              key={y}
              className="text-xs text-gray-700 dark:text-gray-300 py-1"
            >
              {y}
            </div>
          ))}
        </div>
        <div>
          <div className="flex mb-1">
            <div className="w-8"></div>
            {xValues.map((x) => (
              <div key={x} className="text-xs text-gray-700 dark:text-gray-300 flex-1 text-center">
                {x}
              </div>
            ))}
          </div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: `repeat(${xValues.length}, ${cellWidth}px)`,
              gap: '2px',
            }}
          >
            {yValues.map((y) =>
              xValues.map((x) => {
                const value = valueMap.get(`${x}-${y}`) || 0
                return (
                  <div
                    key={`${x}-${y}`}
                    className="flex items-center justify-center text-xs font-medium rounded-sm"
                    style={{
                      width: `${cellWidth}px`,
                      height: `${cellHeight}px`,
                      backgroundColor: getColor(value),
                      boxShadow: '0 1px 2px rgba(0,0,0,0.1)',
                    }}
                    title={`${x} - ${y}: ${value}`}
                  >
                    {value > 50 ? Math.round(value) : ''}
                  </div>
                )
              })
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
