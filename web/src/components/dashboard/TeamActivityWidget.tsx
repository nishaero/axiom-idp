import React from 'react'
import { Card, StatusBadge } from '@/components/dashboard'
import { useDashboardStore, useRealTimeUpdates } from '@/hooks/useDashboardStore'
import type { TeamActivity } from '@/types/dashboard'

interface ActivityItemProps {
  activity: TeamActivity
}

function ActivityItem({ activity }: ActivityItemProps) {
  const getTypeIcon = () => {
    switch (activity.type) {
      case 'provision':
        return (
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
            />
          </svg>
        )
      case 'deploy':
        return (
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
            />
          </svg>
        )
      case 'config_change':
        return (
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
            />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        )
      case 'delete':
        return (
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        )
      case 'update':
        return (
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            />
          </svg>
        )
      default:
        return null
    }
  }

  const getTypeBadge = () => {
    switch (activity.type) {
      case 'provision':
        return 'pending'
      case 'deploy':
        return 'completed'
      case 'config_change':
        return 'degraded'
      case 'delete':
        return 'stopped'
      case 'update':
        return 'running'
      default:
        return 'unknown'
    }
  }

  const formatTime = (timestamp: number) => {
    const now = Date.now()
    const diff = now - timestamp

    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return `${Math.floor(diff / 86400000)}d ago`
  }

  return (
    <div className="flex items-start gap-3 p-3 hover:bg-gray-50 dark:hover:bg-dark-700/50 rounded-lg transition-colors">
      <div className="flex-shrink-0">
        <div className="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center text-primary-600 dark:text-primary-400">
          {getTypeIcon()}
        </div>
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium text-gray-900 dark:text-white">{activity.user}</span>
          <StatusBadge status={getTypeBadge()} size="sm" />
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          {activity.action} <span className="font-medium text-gray-700 dark:text-gray-300">{activity.target}</span>
        </p>
        <div className="flex items-center gap-3 mt-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{formatTime(activity.timestamp)}</span>
        </div>
      </div>
    </div>
  )
}

interface TeamActivityWidgetProps {
  limit?: number
}

export default function TeamActivityWidget({ limit = 10 }: TeamActivityWidgetProps) {
  const { teamActivity, setTeamActivity } = useDashboardStore()
  const { serviceHealth } = useRealTimeUpdates()

  // Add some new random activities for demo
  const actions = ['deployed', 'provisioned', 'updated', 'deleted', 'configured']
  const targets = [
    'API Gateway',
    'Auth Service',
    'Database Cluster',
    'Staging Environment',
    'Load Balancer',
    'Cache Service',
    'Monitoring Dashboard',
  ]

  const mockNewActivity = (): TeamActivity => ({
    id: `act-${Date.now()}`,
    user: [
      'Alice Chen',
      'Bob Smith',
      'Carol Davis',
      'David Lee',
      'Eva Martinez',
      'Frank Wilson',
    ][Math.floor(Math.random() * 6)],
    action: actions[Math.floor(Math.random() * actions.length)],
    target: targets[Math.floor(Math.random() * targets.length)],
    timestamp: Date.now() - Math.random() * 86400000,
    type: ['deploy', 'provision', 'config_change', 'delete', 'update'][Math.floor(Math.random() * 5)] as any,
  })

  const addNewActivity = () => {
    setTeamActivity([mockNewActivity(), ...teamActivity.slice(0, limit - 1)])
  }

  return (
    <Card
      title="Team Activity Feed"
      actions={
        <button
          onClick={addNewActivity}
          className="p-2 hover:bg-gray-100 dark:hover:bg-dark-700 rounded-lg transition-colors"
          aria-label="Refresh activity"
        >
          <svg className="h-5 w-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
        </button>
      }
    >
      <div className="space-y-1 max-h-64 overflow-y-auto">
        {teamActivity.length === 0 ? (
          <div className="text-center py-8">
            <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.789 9.789 0 01-4.355-.99l-1.798-.998"
              />
            </svg>
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">No recent activity</p>
          </div>
        ) : (
          teamActivity.slice(0, limit).map((activity) => (
            <ActivityItem key={activity.id} activity={activity} />
          ))
        )}
      </div>

      {teamActivity.length > 0 && teamActivity.length >= limit && (
        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-dark-700 text-center">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Showing {Math.min(limit, teamActivity.length)} of {teamActivity.length} activities
          </p>
        </div>
      )}

      {/* Activity type legend */}
      <div className="mt-4 pt-4 border-t border-gray-200 dark:border-dark-700">
        <div className="flex flex-wrap gap-4 justify-center text-xs text-gray-600 dark:text-gray-400">
          <div className="flex items-center gap-1">
            <span className="text-gray-500 dark:text-gray-400">Activity Types:</span>
          </div>
          <div className="flex items-center gap-1">
            <StatusBadge status="completed" size="sm" />
            <span>Deployed</span>
          </div>
          <div className="flex items-center gap-1">
            <StatusBadge status="pending" size="sm" />
            <span>Provisioned</span>
          </div>
          <div className="flex items-center gap-1">
            <StatusBadge status="degraded" size="sm" />
            <span>Config Changed</span>
          </div>
          <div className="flex items-center gap-1">
            <StatusBadge status="stopped" size="sm" />
            <span>Deleted</span>
          </div>
        </div>
      </div>
    </Card>
  )
}
