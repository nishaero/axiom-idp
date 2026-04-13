import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState, useRef, useEffect } from 'react';
import apiClient from '@/lib/api';
export default function AIAssistant() {
    const [messages, setMessages] = useState([]);
    const [input, setInput] = useState('');
    const [loading, setLoading] = useState(false);
    const messagesEndRef = useRef(null);
    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };
    useEffect(() => {
        scrollToBottom();
    }, [messages]);
    const handleSendMessage = async () => {
        if (!input.trim())
            return;
        const userMessage = {
            id: Date.now().toString(),
            role: 'user',
            content: input,
            timestamp: new Date(),
        };
        setMessages((prev) => [...prev, userMessage]);
        setInput('');
        setLoading(true);
        try {
            const response = await apiClient.post('/ai/query', {
                query: input,
                context_limit: 2000,
            });
            const assistantMessage = {
                id: (Date.now() + 1).toString(),
                role: 'assistant',
                content: response.data.response || 'No response received',
                timestamp: new Date(),
            };
            setMessages((prev) => [...prev, assistantMessage]);
        }
        catch (error) {
            console.error('Failed to get AI response:', error);
            const errorMessage = {
                id: (Date.now() + 1).toString(),
                role: 'assistant',
                content: 'Failed to process query',
                timestamp: new Date(),
            };
            setMessages((prev) => [...prev, errorMessage]);
        }
        finally {
            setLoading(false);
        }
    };
    return (_jsxs("div", { className: "h-full flex flex-col p-8", children: [_jsx("h1", { className: "text-3xl font-bold mb-6 text-gray-900 dark:text-white", children: "AI Assistant" }), _jsx("div", { className: "flex-1 overflow-y-auto mb-6 bg-white dark:bg-dark-800 rounded-lg p-4 space-y-4", children: messages.length === 0 ? (_jsx("div", { className: "flex items-center justify-center h-full", children: _jsx("p", { className: "text-gray-600 dark:text-gray-400", children: "Start a conversation with the AI assistant" }) })) : (_jsxs(_Fragment, { children: [messages.map((message) => (_jsx("div", { className: `flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`, children: _jsxs("div", { className: `max-w-md px-4 py-2 rounded-lg ${message.role === 'user'
                                    ? 'bg-primary-600 text-white'
                                    : 'bg-gray-200 dark:bg-dark-700 text-gray-900 dark:text-white'}`, children: [_jsx("p", { children: message.content }), _jsx("span", { className: "text-xs opacity-70", children: message.timestamp.toLocaleTimeString() })] }) }, message.id))), _jsx("div", { ref: messagesEndRef })] })) }), _jsxs("div", { className: "flex gap-4", children: [_jsx("input", { type: "text", placeholder: "Ask me anything...", value: input, onChange: (e) => setInput(e.target.value), onKeyDown: (e) => e.key === 'Enter' && !loading && handleSendMessage(), disabled: loading, className: "flex-1 px-4 py-2 border border-gray-300 dark:border-dark-700 rounded-lg bg-white dark:bg-dark-800 text-gray-900 dark:text-white disabled:opacity-50" }), _jsx("button", { onClick: handleSendMessage, disabled: loading || !input.trim(), className: "px-6 py-2 bg-primary-600 dark:bg-primary-500 text-white rounded-lg hover:bg-primary-700 dark:hover:bg-primary-600 disabled:opacity-50 transition-colors", children: loading ? 'Thinking...' : 'Send' })] })] }));
}
//# sourceMappingURL=AIAssistant.js.map