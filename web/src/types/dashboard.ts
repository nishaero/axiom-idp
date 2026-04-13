// Dashboard Widget Types
export interface DashboardWidget {
  id: string
  title: string
  type: WidgetType
  position: {
    row: number
    col: number
    width: number
    height: number
  }
  refreshInterval?: number // milliseconds
  isVisible: boolean
}

export type WidgetType =
  | 'health'
  | 'costs'
  | 'security'
  | 'performance'
  | 'resource-utilization'
  | 'team-activity'
  | 'service-status'
  | 'api-metrics'

// Metrics Data Types
export interface MetricPoint {
  timestamp: number
  value: number
  label?: string
}

export interface HealthMetric {
  status: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
  uptime: number // percentage
  responseTime: number // ms
  errorRate: number // percentage
  lastCheck: number
}

export interface CostMetric {
  total: number
  breakdown: {
    category: string
    amount: number
    percentage: number
  }[]
  period: {
    start: string
    end: string
  }
  projectedTotal: number
}

export interface SecurityMetric {
  overallScore: number // 0-100
  vulnerabilities: {
    critical: number
    high: number
    medium: number
    low: number
  }
  compliance: {
    passed: number
    failed: number
    total: number
  }
  lastScan: number
}

export interface PerformanceMetric {
  responseTimes: MetricPoint[]
  throughput: MetricPoint[]
  errorRates: MetricPoint[]
  concurrentUsers: MetricPoint[]
}

export interface ResourceUtilization {
  cpu: number // percentage
  memory: number // percentage
  storage: number // percentage
  network: number // mbps
  hosts: {
    name: string
    cpu: number
    memory: number
    status: 'healthy' | 'degraded' | 'unhealthy'
  }[]
}

export interface TeamActivity {
  id: string
  user: string
  action: string
  target: string
  timestamp: number
  type: 'provision' | 'deploy' | 'update' | 'delete' | 'config_change'
}

export interface ServiceHealth {
  id: string
  name: string
  status: 'running' | 'stopped' | 'degraded' | 'error'
  health: HealthMetric
  uptime: number // percentage
  lastDeployment: number
  replicas: {
    desired: number
    current: number
    available: number
  }
}

// Widget Props
export interface WidgetProps<T = unknown> {
  data: T
  loading?: boolean
  error?: string | null
  onRefresh: () => void
  lastUpdated?: number
}

// Service Provisioning Types
export interface ServiceTemplate {
  id: string
  name: string
  description: string
  category: string
  icon: string
  parameters: TemplateParameter[]
  defaultValues: Record<string, unknown>
  estimatedCost: {
    min: number
    max: number
    currency: string
  }
}

export interface TemplateParameter {
  name: string
  type: 'text' | 'number' | 'select' | 'boolean' | 'textarea'
  label: string
  description?: string
  required: boolean
  options?: { value: string; label: string }[]
  validation?: {
    minLength?: number
    maxLength?: number
    pattern?: string
  }
}

export interface ServiceProvisioningRequest {
  templateId: string
  serviceName: string
  parameters: Record<string, unknown>
  environment: 'dev' | 'staging' | 'prod'
  approvers: string[]
  estimatedCost: number
}

export interface ProvisioningWorkflow {
  id: string
  serviceId: string
  status: 'pending' | 'in-progress' | 'approved' | 'completed' | 'rejected'
  created: number
  updated: number
  steps: WorkflowStep[]
  createdBy: string
  estimatedCost: number
}

export interface WorkflowStep {
  id: string
  name: string
  status: 'pending' | 'in-progress' | 'completed' | 'failed'
  order: number
  description?: string
  duration?: number // seconds
  required: boolean
}

// WebSocket Types
export interface WebSocketMessage {
  type: 'metric-update' | 'health-alert' | 'cost-update' | 'security-event' | 'heartbeat'
  timestamp: number
  data: unknown
}

export interface WebSocketStatus {
  connected: boolean
  reconnectAttempts: number
  lastMessageAt?: number
}
