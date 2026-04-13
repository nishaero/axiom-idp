import { useState } from 'react'
import apiClient from '@/lib/api'

export default function Catalog() {
  const [services, setServices] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  const handleSearch = async () => {
    setLoading(true)
    try {
      const response = await apiClient.get<{ results: any[] }>('/catalog/search', {
        params: { q: searchQuery },
      })
      setServices((response.data as { results?: any[] }).results || [])
    } catch (error) {
      console.error('Search failed:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="p-8">
      <h1 className="text-3xl font-bold mb-6 text-gray-900 dark:text-white">
        Service Catalog
      </h1>

      <div className="flex gap-4 mb-6">
        <input
          type="text"
          placeholder="Search services..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          className="flex-1 px-4 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white"
        />
        <button
          onClick={handleSearch}
          disabled={loading}
          className="px-6 py-2 bg-primary-600 dark:bg-primary-500 text-white rounded-lg hover:bg-primary-700 dark:hover:bg-primary-600 disabled:opacity-50 transition-colors"
        >
          {loading ? 'Searching...' : 'Search'}
        </button>
      </div>

      <div className="grid grid-cols-2 gap-6">
        {services.map((service) => (
          <div
            key={service.id}
            className="bg-white dark:bg-dark-800 p-6 rounded-lg shadow hover:shadow-lg transition-shadow"
          >
            <h3 className="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
              {service.name}
            </h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              {service.description}
            </p>
            <div className="flex gap-2">
              {service.tags &&
                service.tags.map((tag: string) => (
                  <span
                    key={tag}
                    className="px-2 py-1 bg-primary-100 dark:bg-primary-900 text-primary-800 dark:text-primary-200 text-xs rounded"
                  >
                    {tag}
                  </span>
                ))}
            </div>
          </div>
        ))}
      </div>

      {services.length === 0 && !loading && (
        <div className="text-center py-12">
          <p className="text-gray-600 dark:text-gray-400">
            {searchQuery ? 'No services found' : 'Search to browse services'}
          </p>
        </div>
      )}
    </div>
  )
}
