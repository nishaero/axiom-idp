import React, { useState } from 'react'
import { Card, StatusBadge, ProgressBar, MetricDisplay } from '@/components/dashboard'
import type { ApprovalWorkflowData, ApprovalAction } from '@/types/dashboard'

interface ApprovalsWorkflowProps {
  workflowData?: ApprovalWorkflowData
  onApprove?: (requestId: string, comment?: string) => Promise<void>
  onReject?: (requestId: string, comment: string) => Promise<void>
}

interface ApprovalActionButtonsProps {
  request: ApprovalWorkflowData['requests'][number]
  onApprove: (comment?: string) => void
  onReject: (comment: string) => void
}

function ApprovalActionButtons({ request, onApprove, onReject }: ApprovalActionButtonsProps) {
  const [showComment, setShowComment] = useState(false)
  const [comment, setComment] = useState('')

  const isActionable = request.status === 'pending'
  const isOwner = true // Could be derived from auth context

  const handleApprove = () => {
    onApprove(comment)
    setShowComment(false)
    setComment('')
  }

  const handleReject = () => {
    if (comment.trim()) {
      onReject(comment)
    }
    setShowComment(false)
    setComment('')
  }

  return (
    <div className="flex items-center gap-2">
      {isActionable && isOwner && (
        <>
          <button
            onClick={() => setShowComment(true)}
            className="px-3 py-1.5 bg-green-500 text-white rounded-lg text-sm font-medium hover:bg-green-600 transition-colors flex items-center gap-1"
          >
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            Approve
          </button>
          <button
            onClick={() => setShowComment(true)}
            className="px-3 py-1.5 bg-red-500 text-white rounded-lg text-sm font-medium hover:bg-red-600 transition-colors flex items-center gap-1"
          >
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
            Reject
          </button>
        </>
      )}
      {request.status === 'approved' && (
        <StatusBadge status="completed" size="sm" showIcon={true}>Approved</StatusBadge>
      )}
      {request.status === 'rejected' && (
        <StatusBadge status="error" size="sm" showIcon={true}>Rejected</StatusBadge>
      )}
    </div>
  )
}

function CommentInput({
  comment,
  setComment,
  onConfirm,
  actionType,
}: {
  comment: string
  setComment: (value: string) => void
  onConfirm: () => void
  actionType: 'approve' | 'reject'
}) {
  return (
    <div className="space-y-3">
      <textarea
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        placeholder={actionType === 'reject' ? 'Please provide a reason for rejection...' : 'Optional comment...'}
        rows={3}
        className="w-full px-3 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 resize-none"
      />
      <div className="flex items-center justify-end gap-2">
        <button
          onClick={() => onConfirm()}
          className="px-4 py-2 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onConfirm}
          className={`px-4 py-2 rounded-lg text-white font-medium transition-colors ${
            actionType === 'approve' ? 'bg-green-500 hover:bg-green-600' : 'bg-red-500 hover:bg-red-600'
          }`}
        >
          {actionType === 'approve' ? 'Confirm Approval' : 'Submit Rejection'}
        </button>
      </div>
    </div>
  )
}

export default function ApprovalsWorkflow({
  workflowData,
  onApprove,
  onReject,
}: ApprovalsWorkflowProps) {
  const [activeTab, setActiveTab] = useState<'all' | 'pending' | 'approved' | 'rejected'>('all')
  const [selectedRequest, setSelectedRequest] = useState<string | null>(null)
  const [isProcessing, setIsProcessing] = useState<string | null>(null)

  const workflow = workflowData || {
    id: 'WF-001',
    name: 'Production Deployment Approval',
    description: 'Production deployment approval workflow',
    createdBy: 'System',
    createdAt: Date.now() - 86400000,
    status: 'active',
    requests: [
      {
        id: 'REQ-001',
        serviceName: 'API Gateway',
        requestType: 'deployment',
        version: 'v2.3.1',
        requestedBy: 'Alice Chen',
        requestedAt: Date.now() - 3600000,
        status: 'pending',
        priority: 'high',
        environment: 'production',
        costImpact: 150,
        approvers: [
          { id: 'usr-1', name: 'Bob Smith', role: 'Engineering Lead', status: 'pending' },
          { id: 'usr-2', name: 'Carol Davis', role: 'Security Officer', status: 'approved' },
        ],
        details: {
          changes: ['Fixed authentication timeout issue', 'Updated API rate limits', 'Optimized database queries'],
          riskLevel: 'medium',
          rollbackPlan: 'Automatic rollback on health check failure',
          testingCompleted: true,
        },
        comments: [
          {
            id: 'cmt-1',
            author: 'Carol Davis',
            text: 'Security review completed. Approved from security perspective.',
            timestamp: Date.now() - 1800000,
          },
        ],
      },
      {
        id: 'REQ-002',
        serviceName: 'Auth Service',
        requestType: 'config_change',
        version: 'v1.8.0',
        requestedBy: 'David Lee',
        requestedAt: Date.now() - 7200000,
        status: 'approved',
        priority: 'medium',
        environment: 'production',
        costImpact: 50,
        approvers: [
          { id: 'usr-1', name: 'Bob Smith', role: 'Engineering Lead', status: 'approved' },
          { id: 'usr-2', name: 'Carol Davis', role: 'Security Officer', status: 'approved' },
        ],
        details: {
          changes: ['Updated JWT token expiry settings', 'Enhanced session management'],
          riskLevel: 'low',
          rollbackPlan: 'Configuration reverted to previous version',
          testingCompleted: true,
        },
        comments: [
          {
            id: 'cmt-1',
            author: 'Bob Smith',
            text: 'Approved. Changes look good.',
            timestamp: Date.now() - 5400000,
          },
          {
            id: 'cmt-2',
            author: 'Carol Davis',
            text: 'Security approved.',
            timestamp: Date.now() - 4800000,
          },
        ],
      },
      {
        id: 'REQ-003',
        serviceName: 'Database Cluster',
        requestType: 'infrastructure',
        version: 'v3.0.0',
        requestedBy: 'Eva Martinez',
        requestedAt: Date.now() - 172800000,
        status: 'rejected',
        priority: 'high',
        environment: 'production',
        costImpact: 500,
        approvers: [
          { id: 'usr-1', name: 'Bob Smith', role: 'Engineering Lead', status: 'pending' },
          { id: 'usr-2', name: 'Carol Davis', role: 'Security Officer', status: 'rejected' },
        ],
        details: {
          changes: ['Upgrade PostgreSQL version', 'Increase storage capacity', 'Add read replicas'],
          riskLevel: 'high',
          rollbackPlan: 'Manual rollback required',
          testingCompleted: false,
        },
        comments: [
          {
            id: 'cmt-1',
            author: 'Carol Davis',
            text: 'Rejected due to incomplete testing. Please complete integration tests first.',
            timestamp: Date.now() - 155520000,
          },
        ],
      },
    ],
  }

  const filteredRequests = workflow.requests.filter((request) => {
    if (activeTab === 'all') return true
    if (activeTab === 'pending') return request.status === 'pending'
    if (activeTab === 'approved') return request.status === 'approved'
    if (activeTab === 'rejected') return request.status === 'rejected'
    return true
  })

  const handleApprove = async (requestId: string, comment?: string) => {
    setIsProcessing(requestId)
    try {
      await onApprove?.(requestId, comment)
    } finally {
      setIsProcessing(null)
    }
  }

  const handleReject = async (requestId: string, comment: string) => {
    setIsProcessing(requestId)
    try {
      await onReject?.(requestId, comment)
    } finally {
      setIsProcessing(null)
    }
  }

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'bg-red-500'
      case 'medium':
        return 'bg-yellow-500'
      case 'low':
        return 'bg-green-500'
      default:
        return 'bg-gray-500'
    }
  }

  const getRiskColor = (riskLevel: string) => {
    switch (riskLevel) {
      case 'high':
        return 'text-red-600 dark:text-red-400'
      case 'medium':
        return 'text-yellow-600 dark:text-yellow-400'
      case 'low':
        return 'text-green-600 dark:text-green-400'
      default:
        return 'text-gray-600 dark:text-gray-400'
    }
  }

  const pendingCount = workflow.requests.filter((r) => r.status === 'pending').length
  const approvalRate = Math.round(
    ((workflow.requests.filter((r) => r.status === 'approved').length) / workflow.requests.length) * 100
  )

  const formatDate = (timestamp: number) => {
    const date = new Date(timestamp)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const formatTimeAgo = (timestamp: number) => {
    const now = Date.now()
    const diff = now - timestamp

    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return `${Math.floor(diff / 86400000)}d ago`
  }

  const renderApproverList = (approvers: ApprovalWorkflowData['requests'][number]['approvers']) => (
    <div className="space-y-2 mt-3">
      {approvers.map((approver) => (
        <div key={approver.id} className="flex items-center justify-between p-2 bg-gray-50 dark:bg-dark-700/50 rounded-lg">
          <div>
            <div className="text-sm font-medium text-gray-900 dark:text-white">{approver.name}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{approver.role}</div>
          </div>
          <StatusBadge
            status={approver.status === 'approved' ? 'completed' : approver.status === 'rejected' ? 'error' : 'pending'}
            size="sm"
          />
        </div>
      ))}
    </div>
  )

  const renderCommentThread = (comments: ApprovalWorkflowData['requests'][number]['comments']) => (
    <div className="space-y-3 mt-4">
      {comments.map((comment) => (
        <div key={comment.id} className="flex items-start gap-3">
          <div className="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center text-primary-600 dark:text-primary-400 text-sm font-medium">
            {comment.author.charAt(0)}
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-900 dark:text-white">{comment.author}</span>
              <span className="text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(comment.timestamp)}</span>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{comment.text}</p>
          </div>
        </div>
      ))}
    </div>
  )

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{workflow.name}</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">{workflow.description}</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="px-4 py-2 bg-gray-100 dark:bg-dark-700 rounded-lg">
            <span className="text-sm text-gray-600 dark:text-gray-400">Approval Rate</span>
            <span className="text-lg font-semibold text-primary-600 dark:text-primary-400 ml-2">{approvalRate}%</span>
          </div>
          <div className="px-4 py-2 bg-gray-100 dark:bg-dark-700 rounded-lg">
            <span className="text-sm text-gray-600 dark:text-gray-400">Pending</span>
            <span className="text-lg font-semibold text-yellow-600 dark:text-yellow-400 ml-2">{pendingCount}</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column: Requests List */}
        <div className="lg:col-span-2 space-y-6">
          <Card title="Approval Requests">
            {/* Tab Navigation */}
            <div className="flex items-center gap-2 mb-4 border-b border-gray-200 dark:border-dark-700">
              {(['all', 'pending', 'approved', 'rejected'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${
                    activeTab === tab
                      ? 'bg-white dark:bg-dark-800 text-primary-600 dark:text-primary-400 border-b-2 border-primary-500'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
                  }`}
                >
                  {tab.charAt(0).toUpperCase() + tab.slice(1)}
                  {tab === 'pending' && pendingCount > 0 && (
                    <span className="ml-2 px-2 py-0.5 bg-red-500 text-white rounded-full text-xs">
                      {pendingCount}
                    </span>
                  )}
                </button>
              ))}
            </div>

            {/* Requests List */}
            <div className="space-y-3">
              {filteredRequests.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                  No requests found
                </div>
              ) : (
                filteredRequests.map((request) => (
                  <div
                    key={request.id}
                    className={`p-4 border-2 rounded-lg ${
                      selectedRequest === request.id
                        ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                        : 'border-gray-200 dark:border-dark-700'
                    }`}
                    onClick={() => setSelectedRequest(request.id === selectedRequest ? null : request.id)}
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-start gap-3">
                        <StatusBadge
                          status={request.status === 'pending' ? 'warning' : request.status === 'approved' ? 'completed' : 'error'}
                          size="sm"
                        />
                        <div>
                          <h3 className="font-semibold text-gray-900 dark:text-white">{request.serviceName}</h3>
                          <div className="flex items-center gap-2 mt-1">
                            <span className="text-xs text-gray-500 dark:text-gray-400">Version {request.version}</span>
                            <span className="text-xs text-gray-500 dark:text-gray-400">•</span>
                            <span className="text-xs text-gray-500 dark:text-gray-400 capitalize">{request.requestType}</span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${getPriorityColor(request.priority)}`}>
                          {request.priority}
                        </span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4 text-sm mb-3">
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Requestor:</span>
                        <span className="ml-2 text-gray-900 dark:text-white">{request.requestedBy}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Requested:</span>
                        <span className="ml-2 text-gray-900 dark:text-white">{formatDate(request.requestedAt)}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Environment:</span>
                        <span className="ml-2 text-gray-900 dark:text-white capitalize">{request.environment}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 dark:text-gray-400">Cost Impact:</span>
                        <span className="ml-2 text-gray-900 dark:text-white">${request.costImpact}</span>
                      </div>
                    </div>

                    {request.status === 'pending' && (
                      <div className="mt-3 pl-3 border-l-4 border-yellow-500">
                        <div className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                          {request.details.riskLevel === 'high' && (
                            <span className="text-red-600 dark:text-red-400 font-medium">High Risk:</span>
                          )}
                          {request.details.riskLevel === 'medium' && (
                            <span className="text-yellow-600 dark:text-yellow-400 font-medium">Medium Risk:</span>
                          )}
                          <span className={getRiskColor(request.details.riskLevel)}>
                            {request.details.riskLevel.charAt(0).toUpperCase() + request.details.riskLevel.slice(1)} risk
                          </span>
                        </div>
                        <div className="space-y-1 text-xs text-gray-500 dark:text-gray-400">
                          <div>• {request.details.testingCompleted ? '✓' : '✗'} Testing completed</div>
                          <div>• {request.details.rollbackPlan}</div>
                        </div>
                      </div>
                    )}

                    <ApprovalActionButtons
                      request={request}
                      onApprove={(comment) => handleApprove(request.id, comment)}
                      onReject={(comment) => handleReject(request.id, comment)}
                    />

                    {selectedRequest === request.id && (
                      <div className="mt-4 pt-4 border-t border-gray-200 dark:border-dark-700">
                        {renderApproverList(request.approvers)}
                        {renderCommentThread(request.comments)}
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </Card>
        </div>

        {/* Right Column: Workflow Summary */}
        <div className="space-y-6">
          <Card title="Workflow Summary">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">Workflow Status</span>
                <StatusBadge status={workflow.status === 'active' ? 'running' : 'stopped'} size="sm" />
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">Total Requests</span>
                <span className="font-semibold text-gray-900 dark:text-white">{workflow.requests.length}</span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">Pending Approval</span>
                <span className="font-semibold text-yellow-600 dark:text-yellow-400">{pendingCount}</span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">Approval Rate</span>
                <span className="font-semibold text-green-600 dark:text-green-400">{approvalRate}%</span>
              </div>

              <div className="border-t border-gray-200 dark:border-dark-700 pt-4">
                <div className="flex items-center gap-2 mb-3">
                  <svg className="h-5 w-5 text-primary-600 dark:text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Quick Stats</span>
                </div>

                <div className="grid grid-cols-2 gap-2">
                  <div className="text-center p-2 bg-green-50 dark:bg-green-900/20 rounded-lg">
                    <div className="text-lg font-bold text-green-600 dark:text-green-400">
                      {workflow.requests.filter((r) => r.status === 'approved').length}
                    </div>
                    <div className="text-xs text-gray-600 dark:text-gray-400">Approved</div>
                  </div>
                  <div className="text-center p-2 bg-red-50 dark:bg-red-900/20 rounded-lg">
                    <div className="text-lg font-bold text-red-600 dark:text-red-400">
                      {workflow.requests.filter((r) => r.status === 'rejected').length}
                    </div>
                    <div className="text-xs text-gray-600 dark:text-gray-400">Rejected</div>
                  </div>
                  <div className="text-center p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <div className="text-lg font-bold text-blue-600 dark:text-blue-400">
                      {workflow.requests.filter((r) => r.status === 'pending').length}
                    </div>
                    <div className="text-xs text-gray-600 dark:text-gray-400">Pending</div>
                  </div>
                  <div className="text-center p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                    <div className="text-lg font-bold text-purple-600 dark:text-purple-400">
                      {workflow.requests.filter((r) => r.priority === 'high').length}
                    </div>
                    <div className="text-xs text-gray-600 dark:text-gray-400">High Priority</div>
                  </div>
                </div>
              </div>
            </div>
          </Card>

          <Card title="Required Approvers">
            <div className="space-y-3">
              {workflow.requests.reduce<Record<string, string[]>[]>((acc, req) => {
                req.approvers.forEach((approver) => {
                  const key = approver.id
                  if (!acc[key]) acc[key] = []
                  acc[key].push({ name: approver.name, role: approver.role, id: approver.id })
                })
                return acc
              }, {})}
            </div>
          </Card>

          <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
            <div className="flex items-start gap-3">
              <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-200">Workflow Rules</h4>
                <ul className="text-xs text-blue-700 dark:text-blue-300 mt-2 space-y-1">
                  <li>• All high-risk changes require 2 approvers</li>
                  <li>• Security officer must approve production changes</li>
                  <li>• Auto-approve if all required approvers agree</li>
                  <li>• Reject after 48 hours of inactivity</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
