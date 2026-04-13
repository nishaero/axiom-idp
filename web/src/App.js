import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import Layout from '@/components/Layout';
import Dashboard from '@/pages/Dashboard';
import Catalog from '@/pages/Catalog';
import AIAssistant from '@/pages/AIAssistant';
import NotFound from '@/pages/NotFound';
const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 1000 * 60 * 5,
            gcTime: 1000 * 60 * 10,
        },
    },
});
function App() {
    return (_jsx(QueryClientProvider, { client: queryClient, children: _jsx(BrowserRouter, { children: _jsx(Routes, { children: _jsxs(Route, { element: _jsx(Layout, {}), children: [_jsx(Route, { path: "/", element: _jsx(Dashboard, {}) }), _jsx(Route, { path: "/catalog", element: _jsx(Catalog, {}) }), _jsx(Route, { path: "/ai", element: _jsx(AIAssistant, {}) }), _jsx(Route, { path: "*", element: _jsx(NotFound, {}) })] }) }) }) }));
}
export default App;
//# sourceMappingURL=App.js.map