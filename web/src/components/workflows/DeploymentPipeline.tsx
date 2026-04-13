import React, { useState } from 'react'
import { Card, StatusBadge, ProgressBar } from '@/components/dashboard'
import type { PipelineStage, PipelineState } from '@/types/dashboard'

interface DeploymentPipelineProps {
  pipelineName: string
  stages: PipelineStage[]
  onStageComplete?: (stageId: string) => void
  onStageFail?: (stageId: string) => void
}

export default function DeploymentPipeline({
  pipelineName,
  stages,
  onStageComplete,
  onStageFail,
}: DeploymentPipelineProps) {
  const [selectedStage, setSelectedStage] = useState<string | null>(null)
  const [isAutoRollback, setIsAutoRollback] = useState(true)

  const stageIcons: Record<string, JSX.Element> = {
    build: (
      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
    ),
    test: (
      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    security: (
      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
    deploy: (
      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
      </svg>
    ),
    verify: (
      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
    ),
  }

  const getStatusColor = (state: PipelineState) => {
    switch (state) {
      case 'completed':
        return '#10b981'
      case 'in-progress':
        return '#0ea5e9'
      case 'failed':
        return '#ef4444'
      case 'pending':
        return '#9ca3af'
      default:
        return '#9ca3af'
    }
  }

  const isStageComplete = (stageId: string): boolean => {
    return stages.some((s) => s.id === stageId && s.state === 'completed')
  }

  const isStageRunning = (stageId: string): boolean => {
    return stages.some((s) => s.id === stageId && s.state === 'in-progress')
  }

  const isStageFailed = (stageId: string): boolean => {
    return stages.some((s) => s.id === stageId && s.state === 'failed')
  }

  const getPreviousStage = (stageId: string): string | null => {
    const index = stages.findIndex((s) => s.id === stageId)
    if (index > 0) return stages[index - 1].id
    return null
  }

  const getNextStage = (stageId: string): string | null => {
    const index = stages.findIndex((s) => s.id === stageId)
    if (index < stages.length - 1) return stages[index + 1].id
    return null
  }

  const canExecuteStage = (stageId: string): boolean => {
    const prevStageId = getPreviousStage(stageId)
    if (!prevStageId) return true // First stage
    return isStageComplete(prevStageId) && !isStageFailed(prevStageId)
  }

  const handleExecuteStage = async (stageId: string) => {
    if (!canExecuteStage(stageId)) return

    // Update all subsequent stages to pending
    const updatedStages = stages.map((stage) => {
      const currentIndex = stages.findIndex((s) => s.id === stageId)
      const stageIndex = stages.findIndex((s) => s.id === stage.id)
      if (stageIndex > currentIndex) {
        return { ...stage, state: 'pending' }
      }
      return stage
    })

    // Update current stage to in-progress
    const updatedCurrentStage = updatedStages.map((stage) => {
      if (stage.id === stageId) {
        return { ...stage, state: 'in-progress' }
      }
      return stage
    })

    // Simulate execution
    await new Promise((resolve) => setTimeout(resolve, 2000))

    // Mark as completed
    const finalStages = updatedCurrentStage.map((stage) => {
      if (stage.id === stageId) {
        return { ...stage, state: 'completed' }
      }
      return stage
    })

    onStageComplete?.(stageId)
  }

  const handleManualFail = (stageId: string) => {
    const failedStages = stages.map((stage) => {
      if (stage.id === stageId) {
        return { ...stage, state: 'failed' }
      }
      return stage
    })
    onStageFail?.(stageId)
  }

  const handleRollback = () => {
    const rolledBackStages = stages.map((stage) => ({
      ...stage,
      state: 'pending' as PipelineState,
    }))
    console.log('Rollback initiated', rolledBackStages)
  }

  const getOverallStatus = (): 'completed' | 'in-progress' | 'failed' | 'pending' => {
    if (stages.some((s) => s.state === 'failed')) return 'failed'
    if (stages.some((s) => s.state === 'in-progress')) return 'in-progress'
    if (stages.every((s) => s.state === 'completed')) return 'completed'
    return 'pending'
  }

  const overallStatus = getOverallStatus()

  const totalStages = stages.length
  const completedStages = stages.filter((s) => s.state === 'completed').length
  const progressPercentage = (completedStages / totalStages) * 100

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{pipelineName}</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Continuous Deployment Pipeline</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setIsAutoRollback(!isAutoRollback)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                isAutoRollback ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-700'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  isAutoRollback ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
            <span className="text-sm text-gray-700 dark:text-gray-300">Auto Rollback</span>
          </div>
          {overallStatus === 'failed' && (
            <button
              onClick={handleRollback}
              className="px-4 py-2 bg-red-500 text-white rounded-lg font-medium hover:bg-red-600 transition-colors"
            >
              Rollback
            </button>
          )}
        </div>
      </div>

      {/* Progress Bar */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Pipeline Progress</span>
          <span className="text-sm font-semibold text-gray-900 dark:text-white">
            {completedStages}/{totalStages} stages complete
          </span>
        </div>
        <div className="flex justify-center">
          <div className="w-full max-w-3xl h-3 bg-gray-200 dark:bg-dark-700 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all duration-500 ${
                overallStatus === 'failed' ? 'bg-red-500' : overallStatus === 'completed' ? 'bg-green-500' : 'bg-primary-500'
              }`}
              style={{ width: `${progressPercentage}%` }}
            />
          </div>
        </div>
        <div className="flex justify-between mt-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{overallStatus === 'completed' ? '✓ Pipeline Complete' : overallStatus === 'failed' ? '✗ Pipeline Failed' : 'In Progress'}</span>
          <span>{Math.round(progressPercentage)}% complete</span>
        </div>
      </div>

      {/* Pipeline Stages */}
      <div className="relative mb-8">
        {/* Connecting Lines */}
        <div className="absolute left-12 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-dark-700" />

        <div className="space-y-6">
          {stages.map((stage, index) => {
            const isComplete = isStageComplete(stage.id)
            const isRunning = isStageRunning(stage.id)
            const isFailed = isStageFailed(stage.id)
            const isAvailable = canExecuteStage(stage.id)
            const prevStageId = getPreviousStage(stage.id)
            const nextStageId = getNextStage(stage.id)

            return (
              <div
                key={stage.id}
                className="flex items-start gap-6 relative"
              >
                {/* Stage Icon */}
                <div className="flex-shrink-0">
                  <div
                    className={`relative w-12 h-12 rounded-full flex items-center justify-center transition-colors ${
                      isFailed
                        ? 'bg-red-100 dark:bg-red-900'
                        : isRunning
                        ? 'bg-blue-100 dark:bg-blue-900 animate-pulse'
                        : isComplete
                        ? 'bg-green-100 dark:bg-green-900'
                        : 'bg-gray-100 dark:bg-dark-700'
                    }`}
                  >
                    {isComplete ? (
                      <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <span
                        className={`text-lg font-bold ${
                          isFailed
                            ? 'text-red-600 dark:text-red-400'
                            : isRunning
                            ? 'text-blue-600 dark:text-blue-400'
                            : isComplete
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-gray-600 dark:text-gray-400'
                        }`}
                      >
                        {index + 1}
                      </span>
                    )}
                    {isRunning && (
                      <div className="absolute inset-0 rounded-full border-2 border-blue-500 animate-ping opacity-75" />
                    )}
                  </div>
                </div>

                {/* Stage Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-3">
                      <h3 className="font-semibold text-gray-900 dark:text-white">{stage.name}</h3>
                      <StatusBadge
                        status={isFailed ? 'error' : isRunning ? 'running' : isComplete ? 'completed' : 'pending'}
                        size="sm"
                      />
                    </div>
                    <div className="flex items-center gap-2">
                      {isAvailable && !isRunning && !isFailed && !isComplete && (
                        <button
                          onClick={() => handleExecuteStage(stage.id)}
                          className="px-3 py-1 text-sm bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
                        >
                          Execute
                        </button>
                      )}
                      {isFailed && !isRunning && (
                        <button
                          onClick={() => handleManualFail(stage.id)}
                          className="px-3 py-1 text-sm bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors"
                        >
                          Retry
                        </button>
                      )}
                    </div>
                  </div>

                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{stage.description}</p>

                  {stage.duration && (
                    <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400 mb-3">
                      <div className="flex items-center gap-1">
                        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <span>{stage.duration}</span>
                      </div>
                      {isRunning && (
                        <div className="flex items-center gap-1">
                          <svg className="h-4 w-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                          <span>Running...</span>
                        </div>
                      )}
                    </div>
                  )}

                  {stage.statusMessage && (
                    <div className="text-sm text-gray-600 dark:text-gray-400">{stage.statusMessage}</div>
                  )}

                  {/* Stage-specific metrics */}
                  {stage.metrics && (
                    <div className="mt-3 grid grid-cols-3 gap-3">
                      {stage.metrics.map((metric, idx) => (
                        <div key={idx} className="text-center p-2 bg-gray-50 dark:bg-dark-700/50 rounded-lg">
                          <div className="text-xs text-gray-500 dark:text-gray-400">{metric.label}</div>
                          <div className={`text-lg font-semibold ${metric.isWarning ? 'text-yellow-600 dark:text-yellow-400' : 'text-gray-900 dark:text-white'}`}>
                            {metric.value}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Sub-stages if any */}
                  {stage.subStages && stage.subStages.length > 0 && (
                    <div className="mt-4 pl-4 border-l-2 border-gray-200 dark:border-dark-700 space-y-2">
                      {stage.subStages.map((subStage, sIdx) => (
                        <div key={sIdx} className="flex items-center gap-2 text-xs">
                          <StatusBadge
                            status={subStage.completed ? 'completed' : subStage.failed ? 'error' : 'pending'}
                            size="sm"
                          />
                          <span className="text-gray-600 dark:text-gray-400">{subStage.name}</span>
                          {subStage.duration && (
                            <span className="text-gray-500 dark:text-gray-400">({subStage.duration})</span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Pipeline Summary */}
      <Card title="Pipeline Summary">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
            <div className="text-2xl font-bold text-green-600 dark:text-green-400">{completedStages}</div>
            <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">Completed</div>
          </div>
          <div className="text-center p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
              {stages.filter((s) => s.state === 'in-progress').length}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">In Progress</div>
          </div>
          <div className="text-center p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
            <div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
              {stages.filter((s) => s.state === 'pending').length}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">Pending</div>
          </div>
          <div className="text-center p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
            <div className="text-2xl font-bold text-red-600 dark:text-red-400">
              {stages.filter((s) => s.state === 'failed').length}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">Failed</div>
          </div>
        </div>
      </Card>
    </div>
  )
}
