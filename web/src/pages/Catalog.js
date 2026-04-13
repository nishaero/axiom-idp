import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
import apiClient from '@/lib/api';
export default function Catalog() {
    const [services, setServices] = useState([]);
    const [loading, setLoading] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const handleSearch = async () => {
        setLoading(true);
        try {
            const response = await apiClient.get('/catalog/search', {
                params: { q: searchQuery },
            });
            setServices(response.data.results || []);
        }
        catch (error) {
            console.error('Search failed:', error);
        }
        finally {
            setLoading(false);
        }
    };
    return (_jsxs("div", { className: "p-8", children: [_jsx("h1", { className: "text-3xl font-bold mb-6 text-gray-900 dark:text-white", children: "Service Catalog" }), _jsxs("div", { className: "flex gap-4 mb-6", children: [_jsx("input", { type: "text", placeholder: "Search services...", value: searchQuery, onChange: (e) => setSearchQuery(e.target.value), onKeyDown: (e) => e.key === 'Enter' && handleSearch(), className: "flex-1 px-4 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white" }), _jsx("button", { onClick: handleSearch, disabled: loading, className: "px-6 py-2 bg-primary-600 dark:bg-primary-500 text-white rounded-lg hover:bg-primary-700 dark:hover:bg-primary-600 disabled:opacity-50 transition-colors", children: loading ? 'Searching...' : 'Search' })] }), _jsx("div", { className: "grid grid-cols-2 gap-6", children: services.map((service) => (_jsxs("div", { className: "bg-white dark:bg-dark-800 p-6 rounded-lg shadow hover:shadow-lg transition-shadow", children: [_jsx("h3", { className: "text-lg font-semibold mb-2 text-gray-900 dark:text-white", children: service.name }), _jsx("p", { className: "text-gray-600 dark:text-gray-400 mb-4", children: service.description }), _jsx("div", { className: "flex gap-2", children: service.tags &&
                                service.tags.map((tag) => (_jsx("span", { className: "px-2 py-1 bg-primary-100 dark:bg-primary-900 text-primary-800 dark:text-primary-200 text-xs rounded", children: tag }, tag))) })] }, service.id))) }), services.length === 0 && !loading && (_jsx("div", { className: "text-center py-12", children: _jsx("p", { className: "text-gray-600 dark:text-gray-400", children: searchQuery ? 'No services found' : 'Search to browse services' }) }))] }));
}
//# sourceMappingURL=Catalog.js.map