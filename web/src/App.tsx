import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClientProvider, QueryClient } from '@tanstack/react-query'
import Layout from '@/components/Layout'
import Dashboard from '@/pages/Dashboard'
import Observability from '@/pages/Observability'
import Catalog from '@/pages/Catalog'
import AIAssistant from '@/pages/AIAssistant'
import Settings from '@/pages/Settings'
import NotFound from '@/pages/NotFound'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,
      gcTime: 1000 * 60 * 10,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/observability" element={<Observability />} />
            <Route path="/catalog" element={<Catalog />} />
            <Route path="/ai" element={<AIAssistant />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
