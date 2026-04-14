import { useLocation } from 'react-router-dom'

const routeTitles: Record<string, { title: string; subtitle: string }> = {
  '/': { title: 'Dashboard', subtitle: 'Operational overview and release readiness' },
  '/catalog': { title: 'Service Catalog', subtitle: 'Discover, score, and route services' },
  '/ai': { title: 'AI Assistant', subtitle: 'Ask for risk, evidence, and rollout guidance' },
  '/settings': { title: 'Security & Compliance', subtitle: 'Identity, registry, and control settings' },
}

export default function Header() {
  const { pathname } = useLocation()
  const page = routeTitles[pathname] ?? {
    title: 'Axiom IDP',
    subtitle: 'AI-native internal developer platform',
  }

  return (
    <header className="h-16 bg-white dark:bg-dark-800 border-b border-gray-200 dark:border-dark-700 shadow-sm flex items-center justify-between px-8">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{page.title}</h2>
        <p className="text-xs text-gray-500 dark:text-gray-400">{page.subtitle}</p>
      </div>
      <div className="flex items-center gap-4">
        <span className="hidden sm:inline-flex items-center rounded-full border border-green-200 bg-green-50 px-3 py-1 text-xs font-medium text-green-700 dark:border-green-900 dark:bg-green-950/40 dark:text-green-300">
          GitHub registry synced
        </span>
        <button className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-dark-700 rounded-lg transition-colors">
          Profile
        </button>
      </div>
    </header>
  )
}
