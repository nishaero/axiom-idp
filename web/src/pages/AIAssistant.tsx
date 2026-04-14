import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Card, StatusBadge } from '@/components/dashboard'
import apiClient from '@/lib/api'
import {
  aiPromptSuggestions,
  buildAssistantFallbackResponse,
  demoCatalogServices,
} from '@/lib/demoContent'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
  source?: 'live' | 'local'
}

const initialMessage: Message = {
  id: 'assistant-welcome',
  role: 'assistant',
  content:
    'I can help with release risk, ownership gaps, evidence packs, and rollout guidance. Ask for a service-by-service assessment or a BSI C5 summary.',
  timestamp: new Date(),
  source: 'local',
}

export default function AIAssistant() {
  const [messages, setMessages] = useState<Message[]>([initialMessage])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const contextSummary = useMemo(() => {
    const readyServices = demoCatalogServices.filter((service) => service.releaseState === 'ready').length
    const ownerlessServices = demoCatalogServices.filter((service) => service.owner === 'Unassigned').length

    return {
      serviceCount: demoCatalogServices.length,
      readyServices,
      ownerlessServices,
    }
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ behavior: 'smooth' })
  }, [messages])

  const handleSendMessage = async (overrideInput?: string) => {
    const prompt = (overrideInput ?? input).trim()
    if (!prompt || loading) return

    const userMessage: Message = {
      id: `${Date.now()}-user`,
      role: 'user',
      content: prompt,
      timestamp: new Date(),
    }

    setMessages((previous) => [...previous, userMessage])
    setInput('')
    setLoading(true)

    try {
      const response = await apiClient.post('/ai/query', {
        query: prompt,
        context_limit: 2000,
        context: {
          services: contextSummary.serviceCount,
          readyServices: contextSummary.readyServices,
          ownerlessServices: contextSummary.ownerlessServices,
        },
      })

      const assistantText =
        ((response.data as { response?: string; answer?: string }).response ??
          (response.data as { response?: string; answer?: string }).answer ??
          '').trim() || buildAssistantFallbackResponse(prompt)

      const assistantMessage: Message = {
        id: `${Date.now()}-assistant`,
        role: 'assistant',
        content: /coming soon/i.test(assistantText)
          ? buildAssistantFallbackResponse(prompt, demoCatalogServices)
          : assistantText,
        timestamp: new Date(),
        source: 'live',
      }

      setMessages((previous) => [...previous, assistantMessage])
    } catch (error) {
      console.error('Failed to get AI response:', error)
      setMessages((previous) => [
        ...previous,
        {
          id: `${Date.now()}-assistant`,
          role: 'assistant',
          content: buildAssistantFallbackResponse(prompt, demoCatalogServices),
          timestamp: new Date(),
          source: 'local',
        },
      ])
    } finally {
      setLoading(false)
      textareaRef.current?.focus()
    }
  }

  return (
    <div className="grid gap-6 p-6 md:p-8 xl:grid-cols-[1.15fr_0.85fr]">
      <section className="flex min-h-[72vh] flex-col rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div className="flex items-center justify-between border-b border-gray-200 px-6 py-5 dark:border-dark-700">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">AI Assistant</h1>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Ask for release guidance, compliance evidence, or ownership remediation.
            </p>
          </div>
          <StatusBadge status="running" label="Offline-ready" />
        </div>

        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          {messages.map((message) => (
            <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-2xl rounded-2xl px-4 py-3 shadow-sm ${
                  message.role === 'user'
                    ? 'bg-primary-600 text-white'
                    : 'border border-gray-200 bg-gray-50 text-gray-900 dark:border-dark-700 dark:bg-dark-700 dark:text-white'
                }`}
              >
                <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.2em] opacity-70">
                  <span>{message.role === 'user' ? 'You' : 'Axiom'}</span>
                  <span>·</span>
                  <span>{message.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                  {message.source && (
                    <>
                      <span>·</span>
                      <span>{message.source === 'live' ? 'Live' : 'Local'}</span>
                    </>
                  )}
                </div>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{message.content}</p>
              </div>
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>

        <div className="border-t border-gray-200 p-5 dark:border-dark-700">
          <div className="flex flex-wrap gap-2">
            {aiPromptSuggestions.map((suggestion) => (
              <button
                key={suggestion}
                type="button"
                onClick={() => void handleSendMessage(suggestion)}
                className="rounded-full border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300"
              >
                {suggestion}
              </button>
            ))}
          </div>
          <div className="mt-4 flex flex-col gap-3 sm:flex-row">
            <textarea
              ref={textareaRef}
              rows={3}
              placeholder="Ask me to assess risk, generate evidence, or explain control gaps..."
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  void handleSendMessage()
                }
              }}
              disabled={loading}
              className="min-h-[84px] flex-1 rounded-2xl border border-gray-300 bg-white px-4 py-3 text-sm text-gray-900 shadow-sm outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-700 dark:bg-dark-900 dark:text-white"
            />
            <button
              type="button"
              onClick={() => void handleSendMessage()}
              disabled={loading || !input.trim()}
              className="inline-flex items-center justify-center rounded-2xl bg-primary-600 px-5 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? 'Thinking...' : 'Send'}
            </button>
          </div>
        </div>
      </section>

      <aside className="space-y-4">
        <Card title="Assistant context" subtitle="What the assistant can reason over">
          <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
            <div className="flex items-center justify-between">
              <span>Services indexed</span>
              <span className="font-semibold text-gray-900 dark:text-white">{contextSummary.serviceCount}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Ready for release</span>
              <span className="font-semibold text-green-600 dark:text-green-400">{contextSummary.readyServices}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Owner gaps</span>
              <span className="font-semibold text-amber-600 dark:text-amber-400">{contextSummary.ownerlessServices}</span>
            </div>
          </div>
        </Card>

        <Card title="Market gap focus" subtitle="Features the UI now highlights by default">
          <ul className="space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
            <li>AI release-risk forecasting using live service health and ownership signals.</li>
            <li>BSI C5-ready evidence pack summaries for audit and approval workflows.</li>
            <li>Ownership drift detection for services that are otherwise invisible in static catalogs.</li>
          </ul>
        </Card>

        <Card
          title="Connected views"
          subtitle="Jump between the control plane screens"
          actions={
            <Link
              to="/catalog"
              className="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Open catalog
            </Link>
          }
        >
          <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
            <p>
              Use the catalog to inspect release posture, then come back here for a policy-aware
              narrative and next-step recommendations.
            </p>
            <Link
              to="/settings"
              className="inline-flex rounded-lg border border-gray-300 px-3 py-2 font-medium text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-gray-300"
            >
              Review compliance settings
            </Link>
          </div>
        </Card>
      </aside>
    </div>
  )
}
