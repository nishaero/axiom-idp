import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Header from './Header';
export default function Layout() {
    return (_jsxs("div", { className: "flex h-screen bg-gray-50 dark:bg-dark-900", children: [_jsx(Sidebar, {}), _jsxs("div", { className: "flex-1 flex flex-col overflow-hidden", children: [_jsx(Header, {}), _jsx("main", { className: "flex-1 overflow-auto", children: _jsx(Outlet, {}) })] })] }));
}
//# sourceMappingURL=Layout.js.map