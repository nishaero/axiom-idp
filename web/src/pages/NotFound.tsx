export default function NotFound() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">404</h1>
        <p className="text-gray-600 dark:text-gray-300 mb-8">Page not found</p>
        <a
          href="/"
          className="px-6 py-2 bg-primary-600 dark:bg-primary-500 text-white rounded-lg hover:bg-primary-700 dark:hover:bg-primary-600 transition-colors"
        >
          Back to Dashboard
        </a>
      </div>
    </div>
  )
}
