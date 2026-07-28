import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { Card, Stat, Spinner, useAsync } from '../components/ui'

// A dashboard tile that loads one module's summary and links to its detail page.
function ModuleTile({ title, to, load, render, accent }) {
  const { loading, data, error } = useAsync(load, [])
  return (
    <Link to={to} className="group block">
      <div className="h-full rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition group-hover:border-slate-300 group-hover:shadow-md">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold text-slate-800">{title}</h3>
          <span className={`h-2.5 w-2.5 rounded-full ${accent}`} />
        </div>
        <div className="mt-4 min-h-[92px]">
          {loading ? (
            <Spinner />
          ) : error ? (
            <div className="text-sm text-amber-600">
              {error.status === 503 ? 'Not configured yet' : 'Unavailable'}
            </div>
          ) : (
            render(data)
          )}
        </div>
        <div className="mt-3 text-xs font-medium text-slate-400 group-hover:text-slate-600">View details →</div>
      </div>
    </Link>
  )
}

export default function Overview() {
  return (
    <div>
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Overview</h1>
        <p className="text-sm text-slate-500">Everything Devtron, in one place.</p>
      </header>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <ModuleTile
          title="Zoho Desk — Tickets"
          to="/zoho"
          accent="bg-blue-500"
          load={api.zohoSummary}
          render={(d) => (
            <div className="grid grid-cols-3 gap-2">
              <Stat label="Total" value={d.total} />
              <Stat label="Open" value={d.open} tone="warn" />
              <Stat label="Escalated" value={d.escalated} tone="bad" />
            </div>
          )}
        />

        <ModuleTile
          title="Devtron — Releases"
          to="/devtron"
          accent="bg-emerald-500"
          load={api.devtronSummary}
          render={(d) => (
            <div className="grid grid-cols-3 gap-2">
              <Stat label="Deployments" value={d.totalDeployments} />
              <Stat label="Healthy" value={d.healthy} tone="good" />
              <Stat label="Degraded" value={d.degraded} tone="bad" />
            </div>
          )}
        />

        <ModuleTile
          title="Onboarding"
          to="/onboarding"
          accent="bg-indigo-500"
          load={api.onboardingSummary}
          render={(d) => (
            <div className="grid grid-cols-3 gap-2">
              <Stat label="Total POCs" value={d.total} />
              <Stat label="In progress" value={d.inProgress} tone="warn" />
              <Stat label="Signed" value={d.signed} tone="good" />
            </div>
          )}
        />

        <ModuleTile
          title="Licenses"
          to="/licenses"
          accent="bg-amber-500"
          load={api.licenseSummary}
          render={(d) => (
            <div className="grid grid-cols-3 gap-2">
              <Stat label="Total" value={d.total} />
              <Stat label="Expiring" value={d.expiringSoon} tone="warn" />
              <Stat label="Expired" value={d.expired} tone="bad" />
            </div>
          )}
        />
      </div>
    </div>
  )
}
