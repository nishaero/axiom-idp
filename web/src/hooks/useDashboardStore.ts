import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { DashboardWidget, WidgetType, TeamActivity, ServiceHealth } from '@/types/dashboard'

// Mock data generators for demonstration
const generateMockHealthMetrics = (): ServiceHealth[] => [
  {
    id: 'svc-001',
    name: 'API Gateway',
    status: 'running',
    health: {
      status: 'healthy',
      uptime: 99.9,
      responseTime: 45,
      errorRate: 0.01,
      lastCheck: Date.now(),
    },
    uptime: 99.95,
    lastDeployment: Date.now() - 86400000,
    replicas: { desired: 3, current: 3, available: 3 },
  },
  {
    id: 'svc-002',
    name: 'Auth Service',
    status: 'running',
    health: {
      status: 'healthy',
      uptime: 99.8,
      responseTime: 32,
      errorRate: 0.02,
      lastCheck: Date.now(),
    },
    uptime: 99.75,
    lastDeployment: Date.now() - 172800000,
    replicas: { desired: 2, current: 2, available: 2 },
  },
  {
    id: 'svc-003',
    name: 'Database Cluster',
    status: 'running',
    health: {
      status: 'degraded',
      uptime: 98.5,
      responseTime: 120,
      errorRate: 0.5,
      lastCheck: Date.now(),
    },
    uptime: 98.2,
    lastDeployment: Date.now() - 259200000,
    replicas: { desired: 3, current: 3, available: 2 },
  },
  {
    id: 'svc-004',
    name: 'Cache Service',
    status: 'running',
    health: {
      status: 'healthy',
      uptime: 99.99,
      responseTime: 5,
      errorRate: 0.001,
      lastCheck: Date.now(),
    },
    uptime: 99.98,
    lastDeployment: Date.now() - 43200000,
    replicas: { desired: 2, current: 2, available: 2 },
  },
]

const generateMockTeamActivity = (): TeamActivity[] => [
  {
    id: 'act-001',
    user: 'Alice Chen',
    action: 'deployed',
    target: 'API Gateway v2.3.1',
    timestamp: Date.now() - 3600000,
    type: 'deploy',
  },
  {
    id: 'act-002',
    user: 'Bob Smith',
    action: 'provisioned',
    target: 'New staging environment',
    timestamp: Date.now() - 7200000,
    type: 'provision',
  },
  {
    id: 'act-003',
    user: 'Carol Davis',
    action: 'updated',
    target: 'Auth Service configuration',
    timestamp: Date.now() - 10800000,
    type: 'config_change',
  },
  {
    id: 'act-004',
    user: 'David Lee',
    action: 'deleted',
    target: 'Legacy API v1',
    timestamp: Date.now() - 14400000,
    type: 'delete',
  },
]

const generateMockPerformanceData = () => {
  const data = []
  const now = Date.now()
  for (let i = 12; i >= 0; i--) {
    const hour = new Date(now - i * 3600000)
    data.push({
      label: hour.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
      timestamp: hour.getTime(),
      responseTime: 30 + Math.random() * 30,
      throughput: 500 + Math.random() * 200,
      errorRate: Math.random() * 0.1,
      concurrentUsers: 100 + Math.floor(Math.random() * 100),
    })
  }
  return data
}

interface DashboardStore {
  // Widgets
  widgets: DashboardWidget[]
  setWidgets: (widgets: DashboardWidget[]) => void
  addWidget: (widget: DashboardWidget) => void
  removeWidget: (widgetId: string) => void
  updateWidgetPosition: (widgetId: string, position: { row: number; col: number; width: number; height: number }) => void
  toggleWidgetVisibility: (widgetId: string) => void

  // Metrics data
  serviceHealth: ServiceHealth[]
  setServiceHealth: (health: ServiceHealth[]) => void
  refreshServiceHealth: () => void

  teamActivity: TeamActivity[]
  setTeamActivity: (activity: TeamActivity[]) => void

  // Performance metrics
  performanceData: ReturnType<typeof generateMockPerformanceData>
  setPerformanceData: (data: ReturnType<typeof generateMockPerformanceData>) => void

  // UI state
  darkMode: boolean
  toggleDarkMode: () => void
  customView: string
  setCustomView: (view: string) => void
}

export const useDashboardStore = create<DashboardStore>()(
  persist(
    (set, get) => ({
      // Initial widgets
      widgets: [
        {
          id: 'health-summary',
          title: 'Service Health Summary',
          type: 'health',
          position: { row: 0, col: 0, width: 2, height: 1 },
          refreshInterval: 30000,
          isVisible: true,
        },
        {
          id: 'performance-graph',
          title: 'API Performance',
          type: 'performance',
          position: { row: 0, col: 2, width: 2, height: 2 },
          refreshInterval: 60000,
          isVisible: true,
        },
        {
          id: 'cost-breakdown',
          title: 'Cost Breakdown',
          type: 'costs',
          position: { row: 2, col: 0, width: 1, height: 2 },
          refreshInterval: 3600000,
          isVisible: true,
        },
        {
          id: 'security-posture',
          title: 'Security Posture',
          type: 'security',
          position: { row: 2, col: 1, width: 1, height: 1 },
          refreshInterval: 3600000,
          isVisible: true,
        },
        {
          id: 'resource-utilization',
          title: 'Resource Utilization',
          type: 'resource-utilization',
          position: { row: 2, col: 2, width: 2, height: 1 },
          refreshInterval: 60000,
          isVisible: true,
        },
        {
          id: 'team-activity',
          title: 'Team Activity Feed',
          type: 'team-activity',
          position: { row: 3, col: 0, width: 2, height: 1 },
          refreshInterval: 30000,
          isVisible: true,
        },
      ],

      setWidgets: (widgets) => set({ widgets }),

      addWidget: (widget) =>
        set((state) => ({
          widgets: [...state.widgets, widget],
        })),

      removeWidget: (widgetId) =>
        set((state) => ({
          widgets: state.widgets.filter((w) => w.id !== widgetId),
        })),

      updateWidgetPosition: (widgetId, position) =>
        set((state) => ({
          widgets: state.widgets.map((w) =>
            w.id === widgetId ? { ...w, position } : w
          ),
        })),

      toggleWidgetVisibility: (widgetId) =>
        set((state) => ({
          widgets: state.widgets.map((w) =>
            w.id === widgetId ? { ...w, isVisible: !w.isVisible } : w
          ),
        })),

      // Service health metrics
      serviceHealth: generateMockHealthMetrics(),
      setServiceHealth: (health) => set({ serviceHealth: health }),
      refreshServiceHealth: () =>
        set({ serviceHealth: generateMockHealthMetrics() }),

      // Team activity
      teamActivity: generateMockTeamActivity(),
      setTeamActivity: (activity) => set({ teamActivity: activity }),

      // Performance data
      performanceData: generateMockPerformanceData(),
      setPerformanceData: (data) => set({ performanceData: data }),

      // UI state
      darkMode: false,
      toggleDarkMode: () =>
        set((state) => {
          const newDarkMode = !state.darkMode
          document.documentElement.classList.toggle('dark', newDarkMode)
          return { darkMode: newDarkMode }
        }),
      customView: 'default',
      setCustomView: (view) => set({ customView: view }),
    }),
    {
      name: 'axiom-dashboard-storage',
      partialize: (state) => ({
        widgets: state.widgets,
        darkMode: state.darkMode,
        customView: state.customView,
      }),
    }
  )
)

// Custom hook for dashboard metrics
export function useDashboardMetrics() {
  const {
    serviceHealth,
    setServiceHealth,
    refreshServiceHealth,
    performanceData,
    setPerformanceData,
    teamActivity,
    setTeamActivity,
  } = useDashboardStore()

  // Calculate derived metrics
  const avgUptime = serviceHealth.reduce((sum, svc) => sum + svc.health.uptime, 0) / serviceHealth.length
  const avgResponseTime = serviceHealth.reduce((sum, svc) => sum + svc.health.responseTime, 0) / serviceHealth.length
  const avgErrorRate = serviceHealth.reduce((sum, svc) => sum + svc.health.errorRate, 0) / serviceHealth.length

  const healthyServices = serviceHealth.filter((svc) => svc.health.status === 'healthy').length
  const unhealthyServices = serviceHealth.filter((svc) => svc.health.status !== 'healthy').length

  const totalReplicas = serviceHealth.reduce((sum, svc) => sum + svc.replicas.desired, 0)
  const availableReplicas = serviceHealth.reduce((sum, svc) => sum + svc.replicas.available, 0)

  return {
    // Raw data
    serviceHealth,
    performanceData,
    teamActivity,

    // Derived metrics
    avgUptime: avgUptime.toFixed(2),
    avgResponseTime: Math.round(avgResponseTime),
    avgErrorRate: (avgErrorRate * 100).toFixed(2),
    healthyServices,
    unhealthyServices,
    totalReplicas,
    availableReplicas,

    // Actions
    setServiceHealth,
    refreshServiceHealth,
    setPerformanceData,
    setTeamActivity,
  }
}

// Custom hook for real-time updates simulation
export function useRealTimeUpdates() {
  const {
    serviceHealth,
    setServiceHealth,
    performanceData,
    setPerformanceData,
    teamActivity,
    setTeamActivity,
  } = useDashboardStore()

  // Simulate real-time updates
  useEffect(() => {
    const interval = setInterval(() => {
      // Update service health with slight variations
      setServiceHealth(
        serviceHealth.map((svc) => ({
          ...svc,
          health: {
            ...svc.health,
            responseTime: svc.health.responseTime + (Math.random() - 0.5) * 10,
            uptime: Math.min(100, svc.health.uptime + (Math.random() - 0.5) * 0.01),
          },
        }))
      )

      // Update performance data
      const newData = performanceData.map((point, index) => {
        if (index === performanceData.length - 1) return point
        return {
          ...point,
          responseTime: point.responseTime + (Math.random() - 0.5) * 5,
          throughput: point.throughput + (Math.random() - 0.5) * 20,
        }
      })
      setPerformanceData(newData)
    }, 5000)

    return () => clearInterval(interval)
  }, [serviceHealth, performanceData, setServiceHealth, setPerformanceData])

  return {
    serviceHealth,
    performanceData,
    teamActivity,
  }
}
