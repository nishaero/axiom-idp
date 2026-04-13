export default function Header() {
  return (
    <header className="h-16 bg-white dark:bg-dark-800 border-b border-gray-200 dark:border-dark-700 shadow-sm flex items-center justify-between px-8">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Dashboard</h2>
      <div className="flex items-center gap-4">
        <button className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-dark-700 rounded-lg transition-colors">
          Profile
        </button>
      </div>
    </header>
  )
}
