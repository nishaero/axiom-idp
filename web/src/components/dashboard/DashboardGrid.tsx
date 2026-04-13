import React, { useMemo } from 'react'
import { useDashboardStore } from '@/hooks/useDashboardStore'
import ServiceHealthWidget from './ServiceHealthWidget'
import PerformanceMetricsWidget from './PerformanceMetricsWidget'
import CostBreakdownWidget from './CostBreakdownWidget'
import SecurityPostureWidget from './SecurityPostureWidget'
import ResourceUtilizationWidget from './ResourceUtilizationWidget'
import TeamActivityWidget from './TeamActivityWidget'

interface WidgetContentProps {
  widgetId: string
}

function getWidgetContent({ widgetId }: WidgetContentProps) {
  switch (widgetId) {
    case 'health-summary':
      return <ServiceHealthWidget />
    case 'performance-graph':
      return <PerformanceMetricsWidget />
    case 'cost-breakdown':
      return <CostBreakdownWidget />
    case 'security-posture':
      return <SecurityPostureWidget />
    case 'resource-utilization':
      return <ResourceUtilizationWidget />
    case 'team-activity':
      return <TeamActivityWidget />
    default:
      return <div className="text-center py-8 text-gray-600 dark:text-gray-400">Widget not found</div>
  }
}

interface WidgetCellProps {
  widget: ReturnType<typeof useDashboardStore>['widgets'][number]
  children: React.ReactNode
}

function WidgetCell({ widget, children }: WidgetCellProps) {
  const isDragging = false

  return (
    <div
      className={`relative ${widget.position.width === 2 ? 'md:col-span-2' : 'md:col-span-1'} ${
        widget.position.height === 2 ? 'md:row-span-2' : ''
      }`}
      style={{
        gridRowStart: widget.position.row + 1,
        gridColumnStart: widget.position.col + 1,
      }}
    >
      <div
        className={`h-full ${
          isDragging ? 'ring-2 ring-primary-500 rounded-lg' : ''
        }`}
      >
        {children}
      </div>
    </div>
  )
}

interface DashboardGridProps {
  customView?: string
}

export default function DashboardGrid({ customView = 'default' }: { customView?: string }) {
  const { widgets } = useDashboardStore()

  const visibleWidgets = useMemo(
    () => widgets.filter((w) => w.isVisible),
    [widgets]
  )

  // Custom view configurations
  const getGridLayout = (view: string) => {
    switch (view) {
      case 'overview':
        return ['health-summary', 'performance-graph', 'cost-breakdown', 'security-posture']
      case 'infrastructure':
        return ['health-summary', 'resource-utilization', 'team-activity']
      case 'financial':
        return ['cost-breakdown', 'security-posture']
      default:
        return ['health-summary', 'performance-graph', 'cost-breakdown', 'security-posture', 'resource-utilization', 'team-activity']
    }
  }

  const activeWidgets = getGridLayout(customView).filter((id) =>
    visibleWidgets.some((w) => w.id === id)
  )

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Real-time metrics and insights
          </p>
        </div>
        <div className="flex items-center gap-4">
          {/* Connection Status */}
          <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-100 dark:bg-dark-700 rounded-lg">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            <span className="text-sm text-gray-600 dark:text-gray-400">Live</span>
          </div>
          {/* View Selector */}
          <div className="flex items-center gap-2">
            <select
              className="px-3 py-2 bg-white dark:bg-dark-800 border border-gray-300 dark:border-dark-700 rounded-lg text-sm text-gray-700 dark:text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary-500"
              defaultValue="default"
            >
              <option value="default">Default View</option>
              <option value="overview">Overview</option>
              <option value="infrastructure">Infrastructure</option>
              <option value="financial">Financial</option>
            </select>
          </div>
        </div>
      </div>

      {/* Quick Stats Bar */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-gradient-to-r from-green-500 to-green-600 rounded-lg p-4 text-white shadow-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-90">Services Running</p>
              <p className="text-2xl font-bold mt-1">12</p>
            </div>
            <svg className="h-8 w-8 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
        </div>
        <div className="bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg p-4 text-white shadow-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-90">Avg Response Time</p>
              <p className="text-2xl font-bold mt-1">45ms</p>
            </div>
            <svg className="h-8 w-8 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
        </div>
        <div className="bg-gradient-to-r from-purple-500 to-purple-600 rounded-lg p-4 text-white shadow-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-90">Active Teams</p>
              <p className="text-2xl font-bold mt-1">8</p>
            </div>
            <svg className="h-8 w-8 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
              />
            </svg>
          </div>
        </div>
        <div className="bg-gradient-to-r from-orange-500 to-orange-600 rounded-lg p-4 text-white shadow-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm opacity-90">Cost Today</p>
              <p className="text-2xl font-bold mt-1">$342</p>
            </div>
            <svg className="h-8 w-8 opacity-75" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
        </div>
      </div>

      {/* Draggable Widget Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6 auto-rows-[minmax(180px,auto)]">
        {activeWidgets.map((widgetId) => {
          const widget = visibleWidgets.find((w) => w.id === widgetId)
          if (!widget) return null

          return (
            <WidgetCell key={widgetId} widget={widget}>
              <div className="h-full">
                {getWidgetContent({ widgetId })}
              </div>
            </WidgetCell>
          )
        })}
      </div>

      {/* Add Widget Button */}
      <button className="w-full py-4 border-2 border-dashed border-gray-300 dark:border-dark-700 rounded-lg flex items-center justify-center gap-2 text-gray-500 dark:text-gray-400 hover:border-primary-500 hover:text-primary-500 transition-colors">
        <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        <span>Add Custom Widget</span>
      </button>
    </div>
  )
}
