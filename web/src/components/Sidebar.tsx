import { NavLink } from 'react-router-dom'

export default function Sidebar() {
  const navItems = [
    { label: 'Dashboard', path: '/' },
    { label: 'Catalog', path: '/catalog' },
    { label: 'AI Assistant', path: '/ai' },
    { label: 'Settings', path: '/settings' },
  ]

  return (
    <aside className="w-64 bg-white dark:bg-dark-800 border-r border-gray-200 dark:border-dark-700 shadow-sm">
      <div className="h-16 flex items-center px-6 border-b border-gray-200 dark:border-dark-700">
        <h1 className="text-xl font-bold text-primary-600 dark:text-primary-400">Axiom</h1>
      </div>
      <nav className="mt-6 space-y-2 px-3">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              `block px-4 py-2 rounded-lg transition-colors ${
                isActive
                  ? 'bg-primary-50 dark:bg-primary-900 text-primary-600 dark:text-primary-400 font-medium'
                  : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-dark-700'
              }`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
