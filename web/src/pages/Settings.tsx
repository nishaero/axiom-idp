import { Card, ProgressBar, StatusBadge } from '@/components/dashboard'
import { Link } from 'react-router-dom'

const controlItems = [
  {
    label: 'Identity and access',
    status: 'completed' as const,
    description: 'SSO, MFA, and role templates should anchor every organization that adopts the platform.',
  },
  {
    label: 'Artifact provenance',
    status: 'completed' as const,
    description: 'Build and release metadata should flow into service scorecards and approval records.',
  },
  {
    label: 'Evidence retention',
    status: 'degraded' as const,
    description: 'Retention policy exists, but some services still need an owner-attested export.',
  },
  {
    label: 'Audit logging',
    status: 'completed' as const,
    description: 'Immutable traces should cover deployments, approvals, catalog changes, and AI actions.',
  },
]

const operatingDefaults = [
  {
    title: '1. Identity and ownership',
    description: 'Connect GitHub teams, service owners, and environment roles before rollout starts.',
  },
  {
    title: '2. Release governance',
    description: 'Require approvals, rollback plans, and evidence checks for critical or regulated services.',
  },
  {
    title: '3. AI routing',
    description: 'Ground AI responses in catalog, health, and evidence snapshots, then deep-link to the assistant.',
  },
]

const aiTouchpoints = [
  'Release decisions in the dashboard and catalog should be explainable and linked to the assistant.',
  'Service drilldowns should produce ready-to-send prompts for remediation or rollout planning.',
  'Evidence summaries should be reusable in audit, approval, and change-review workflows.',
  'AI should reduce toil by drafting context, not by making policy decisions on its own.',
]

export default function Settings() {
  return (
    <div className="space-y-8 p-6 md:p-8">
      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
          <div className="space-y-4">
            <div className="inline-flex rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
              Security & compliance
            </div>
            <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-4xl">
              Control access, evidence, AI routing, and registry trust from one place.
            </h1>
            <p className="max-w-2xl text-base leading-7 text-gray-600 dark:text-gray-300">
              The settings page keeps the platform honest about the controls a real deployment needs:
              identity, auditability, GitHub registry trust, BSI C5-aligned evidence, and grounded AI usage.
            </p>
            <div className="flex flex-wrap gap-3">
              <Link
                to="/ai"
                className="rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-700"
              >
                Ask AI about controls
              </Link>
              <Link
                to="/catalog"
                className="rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-semibold text-gray-700 transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-gray-300"
              >
                Review service scorecards
              </Link>
            </div>
          </div>

          <Card title="Compliance posture" subtitle="Current control coverage in the demo">
            <div className="space-y-4">
              <ProgressBar value={92} label="Identity and access" color="success" />
              <ProgressBar value={87} label="Artifact provenance" color="success" />
              <ProgressBar value={78} label="Evidence retention" color="warning" />
              <ProgressBar value={95} label="Audit traceability" color="success" />
            </div>
          </Card>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <Card title="Recommended operating model" subtitle="The first settings most organizations should lock in">
          <div className="space-y-4">
            {operatingDefaults.map((item) => (
              <div key={item.title} className="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/40">
                <p className="text-sm font-semibold text-gray-900 dark:text-white">{item.title}</p>
                <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{item.description}</p>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Where AI is used" subtitle="AI should reduce decision latency, not bypass controls">
          <ul className="space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
            {aiTouchpoints.map((item) => (
              <li key={item}>• {item}</li>
            ))}
          </ul>
        </Card>
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        {controlItems.map((item) => (
          <Card key={item.label} title={item.label} subtitle={item.description}>
            <div className="flex items-center justify-between">
              <StatusBadge status={item.status} />
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {item.status === 'completed' ? 'Configured' : 'Needs attention'}
              </span>
            </div>
          </Card>
        ))}
      </section>

      <Card title="Implementation notes" subtitle="Where this frontend expects the platform to connect">
        <ul className="grid gap-3 text-sm leading-6 text-gray-600 dark:text-gray-400 md:grid-cols-2">
          <li>GitHub registry metadata should feed service ownership and release provenance.</li>
          <li>BSI C5 evidence should be exportable from deployments, approvals, and audit logs.</li>
          <li>AI assistant responses should be grounded in current catalog and health snapshots.</li>
          <li>Security settings should map to roles, environment policies, and approval gates.</li>
        </ul>
      </Card>
    </div>
  )
}
