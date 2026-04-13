import React, { useState } from 'react'
import { Card, StatusBadge, ProgressBar, MetricDisplay } from '@/components/dashboard'
import type { ServiceProvisioningRequest, ProvisioningWorkflow, ServiceTemplate } from '@/types/dashboard'

interface ServiceProvisioningWorkflowProps {
  templates: ServiceTemplate[]
  onProvision: (request: ServiceProvisioningRequest) => Promise<string>
}

interface WorkflowStatus {
  status: 'idle' | 'in-progress' | 'success' | 'error'
  progress: number
  message: string
}

export default function ServiceProvisioningWorkflow({ templates, onProvision }: ServiceProvisioningWorkflowProps) {
  const [selectedTemplate, setSelectedTemplate] = useState<ServiceTemplate | null>(null)
  const [formData, setFormData] = useState<Record<string, unknown>>({
    serviceName: '',
    environment: 'dev',
  })
  const [workflowStatus, setWorkflowStatus] = useState<WorkflowStatus>({
    status: 'idle',
    progress: 0,
    message: '',
  })
  const [selectedEnvironments, setSelectedEnvironments] = useState<string[]>([])

  const handleTemplateSelect = (template: ServiceTemplate) => {
    setSelectedTemplate(template)
    setFormData({
      serviceName: '',
      environment: 'dev',
      ...template.defaultValues,
    })
    setWorkflowStatus({ status: 'idle', progress: 0, message: '' })
  }

  const handleParameterChange = (name: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [name]: value }))
  }

  const handleEnvironmentToggle = (env: string) => {
    setSelectedEnvironments((prev) =>
      prev.includes(env) ? prev.filter((e) => e !== env) : [...prev, env]
    )
  }

  const validateForm = (): boolean => {
    if (!selectedTemplate) return false
    if (!formData.serviceName) return false

    for (const param of selectedTemplate.parameters) {
      if (param.required && !formData[param.name]) {
        return false
      }
    }
    return true
  }

  const getAvailableEnvironments = () => {
    if (!selectedTemplate) return []
    const envs: string[] = ['dev', 'staging', 'prod']
    return envs.filter((env) =>
      selectedTemplate.environmentSettings?.allowedEnvironments?.includes(env) ?? true
    )
  }

  const handleProvision = async () => {
    if (!selectedTemplate || !validateForm()) return

    setWorkflowStatus({ status: 'in-progress', progress: 0, message: 'Starting provisioning workflow...' })

    try {
      // Simulate provisioning workflow stages
      const stages = [
        { progress: 20, message: 'Validating configuration...' },
        { progress: 40, message: 'Checking dependencies...' },
        { progress: 60, message: 'Deploying infrastructure...' },
        { progress: 80, message: 'Running health checks...' },
        { progress: 100, message: 'Provisioning complete!' },
      ]

      for (const stage of stages) {
        await new Promise((resolve) => setTimeout(resolve, 800))
        setWorkflowStatus({
          status: 'in-progress',
          progress: stage.progress,
          message: stage.message,
        })
      }

      const requestId = await onProvision({
        templateId: selectedTemplate.id,
        serviceName: formData.serviceName as string,
        parameters: formData,
        environment: formData.environment as any,
        approvers: [],
        estimatedCost: selectedTemplate.estimatedCost.min,
      })

      setWorkflowStatus({
        status: 'success',
        progress: 100,
        message: `Provisioning successful! Request ID: ${requestId}`,
      })
    } catch (error) {
      setWorkflowStatus({
        status: 'error',
        progress: 0,
        message: 'Provisioning failed. Please try again.',
      })
    }
  }

  const availableEnvs = getAvailableEnvironments()

  return (
    <div className="p-8 max-w-5xl mx-auto">
      <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-6">
        Service Provisioning Workflow
      </h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Template Selection & Configuration */}
        <div className="space-y-6">
          {/* Template Selection */}
          <Card title="Select Service Template">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {templates.map((template) => (
                <div
                  key={template.id}
                  className={`p-4 border-2 rounded-lg cursor-pointer transition-colors ${
                    selectedTemplate?.id === template.id
                      ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                      : 'border-gray-200 dark:border-dark-700 hover:border-primary-500'
                  }`}
                  onClick={() => handleTemplateSelect(template)}
                >
                  <h3 className="font-semibold text-gray-900 dark:text-white mb-2">
                    {template.name}
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                    {template.description}
                  </p>
                  <div className="space-y-1">
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-gray-500 dark:text-gray-400">Parameters:</span>
                      <span className="font-medium text-gray-900 dark:text-white">
                        {template.parameters.length}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-gray-500 dark:text-gray-400">Est. Cost:</span>
                      <span className="font-medium text-gray-900 dark:text-white">
                        ${template.estimatedCost.min}-{template.estimatedCost.max}/mo
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-gray-500 dark:text-gray-400">Deploy Time:</span>
                      <span className="font-medium text-gray-900 dark:text-white">
                        {template.estimatedDeploymentTime}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Configuration Form */}
          {selectedTemplate && (
            <Card title="Configure Service">
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Service Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.serviceName}
                    onChange={(e) => handleParameterChange('serviceName', e.target.value)}
                    placeholder="e.g., my-awesome-service"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Primary Environment
                  </label>
                  <select
                    value={formData.environment}
                    onChange={(e) => handleParameterChange('environment', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                  >
                    {availableEnvs.map((env) => (
                      <option key={env} value={env} className="capitalize">
                        {env}
                      </option>
                    ))}
                  </select>
                </div>

                {selectedTemplate.parameters
                  .filter((p) => p.name !== 'serviceName' && p.name !== 'environment')
                  .map((param) => (
                    <div key={param.name} className="space-y-1">
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {param.label}
                        {param.required && <span className="text-red-500 ml-1">*</span>}
                      </label>

                      {param.type === 'select' && (
                        <select
                          value={formData[param.name] || ''}
                          onChange={(e) => handleParameterChange(param.name, e.target.value)}
                          className="w-full px-3 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                        >
                          <option value="">Select {param.label}</option>
                          {param.options?.map((option) => (
                            <option key={option} value={option}>
                              {option}
                            </option>
                          ))}
                        </select>
                      )}

                      {param.type === 'toggle' && (
                        <button
                          onClick={() =>
                            handleParameterChange(param.name, !(formData[param.name] as boolean))
                          }
                          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                            formData[param.name] ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-700'
                          }`}
                        >
                          <span
                            className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                              formData[param.name] ? 'translate-x-6' : 'translate-x-1'
                            }`}
                          />
                        </button>
                      )}

                      {param.type === 'number' && (
                        <input
                          type="number"
                          value={formData[param.name] || ''}
                          onChange={(e) =>
                            handleParameterChange(param.name, Number(e.target.value))
                          }
                          min={param.min}
                          max={param.max}
                          step={param.step || 1}
                          className="w-full px-3 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                      )}

                      {param.description && (
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {param.description}
                        </p>
                      )}
                    </div>
                  ))}
              </div>
            </Card>
          )}
        </div>

        {/* Right Column: Workflow Status & Summary */}
        <div className="space-y-6">
          <Card title="Provisioning Workflow">
            {selectedTemplate ? (
              <div className="space-y-4">
                <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg mb-4">
                  <div className="flex items-start gap-3">
                    <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <div>
                      <h4 className="text-sm font-medium text-blue-900 dark:text-blue-200">
                        Workflow Stages
                      </h4>
                      <ul className="text-xs text-blue-700 dark:text-blue-300 mt-2 space-y-1">
                        <li>✓ Validate Configuration</li>
                        <li>✓ Check Dependencies</li>
                        <li>✓ Deploy Infrastructure</li>
                        <li>✓ Health Checks</li>
                        <li>✓ Finalize Deployment</li>
                      </ul>
                    </div>
                  </div>
                </div>

                {workflowStatus.status === 'in-progress' && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-gray-600 dark:text-gray-400">
                        {workflowStatus.message}
                      </span>
                      <span className="text-sm font-medium text-primary-600 dark:text-primary-400">
                        {workflowStatus.progress}%
                      </span>
                    </div>
                    <ProgressBar value={workflowStatus.progress} color="primary" size="sm" showValue={false} />
                  </div>
                )}

                {workflowStatus.status === 'success' && (
                  <div className="bg-green-50 dark:bg-green-900/20 p-4 rounded-lg">
                    <div className="flex items-start gap-3">
                      <svg className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      <div>
                        <h4 className="text-sm font-medium text-green-900 dark:text-green-200">
                          Success!
                        </h4>
                        <p className="text-xs text-green-700 dark:text-green-300 mt-1">
                          {workflowStatus.message}
                        </p>
                      </div>
                    </div>
                  </div>
                )}

                {workflowStatus.status === 'error' && (
                  <div className="bg-red-50 dark:bg-red-900/20 p-4 rounded-lg">
                    <div className="flex items-start gap-3">
                      <svg className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <div>
                        <h4 className="text-sm font-medium text-red-900 dark:text-red-200">
                          Error
                        </h4>
                        <p className="text-xs text-red-700 dark:text-red-300 mt-1">
                          {workflowStatus.message}
                        </p>
                      </div>
                    </div>
                  </div>
                )}

                <button
                  onClick={handleProvision}
                  disabled={!validateForm() || workflowStatus.status === 'in-progress'}
                  className={`w-full py-3 rounded-lg font-medium transition-colors ${
                    !validateForm() || workflowStatus.status === 'in-progress'
                      ? 'bg-gray-300 dark:bg-dark-700 text-gray-500 cursor-not-allowed'
                      : 'bg-primary-500 text-white hover:bg-primary-600'
                  }`}
                >
                  {workflowStatus.status === 'in-progress' ? (
                    <span className="flex items-center justify-center gap-2">
                      <svg className="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                      </svg>
                      Processing...
                    </span>
                  ) : (
                    'Start Provisioning'
                  )}
                </button>
              </div>
            ) : (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                Select a template to begin provisioning
              </div>
            )}
          </Card>

          <Card title="Cost Summary">
            {selectedTemplate ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600 dark:text-gray-400">Estimated Monthly Cost</span>
                  <span className="text-lg font-bold text-gray-900 dark:text-white">
                    ${selectedTemplate.estimatedCost.min}-${selectedTemplate.estimatedCost.max}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600 dark:text-gray-400">Deployment Time</span>
                  <span className="font-medium text-gray-900 dark:text-white">
                    {selectedTemplate.estimatedDeploymentTime}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600 dark:text-gray-400">Required Parameters</span>
                  <span className="font-medium text-gray-900 dark:text-white">
                    {selectedTemplate.parameters.filter((p) => p.required).length}
                  </span>
                </div>
                <div className="border-t border-gray-200 dark:border-dark-700 pt-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-600 dark:text-gray-400">Compliance Status</span>
                    <StatusBadge status="completed" size="sm" />
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                No template selected
              </div>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
