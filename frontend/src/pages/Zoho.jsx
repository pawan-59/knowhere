import { api } from '../lib/api'
import { Card, Stat, Badge, Spinner, ErrorBox, useAsync, fmtDate } from '../components/ui'

export default function Zoho() {
  const summary = useAsync(api.zohoSummary, [])
  const tickets = useAsync(() => api.zohoTickets('?limit=50'), [])

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-slate-900">Zoho Desk — Ticket Monitoring</h1>
        <p className="text-sm text-slate-500">Live from Zoho Desk (India DC).</p>
      </header>

      {summary.error && <ErrorBox error={summary.error} />}
      {summary.loading ? (
        <Spinner />
      ) : summary.data ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Stat label="Total" value={summary.data.total} />
          <Stat label="Open" value={summary.data.open} tone="warn" />
          <Stat label="On Hold" value={summary.data.onHold} />
          <Stat label="Escalated" value={summary.data.escalated} tone="bad" />
        </div>
      ) : null}

      <Card title="Recent tickets">
        {tickets.error ? (
          <ErrorBox error={tickets.error} />
        ) : tickets.loading ? (
          <Spinner />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs uppercase text-slate-400">
                  <th className="py-2 pr-3">#</th>
                  <th className="py-2 pr-3">Subject</th>
                  <th className="py-2 pr-3">Status</th>
                  <th className="py-2 pr-3">Priority</th>
                  <th className="py-2 pr-3">Contact</th>
                  <th className="py-2">Updated</th>
                </tr>
              </thead>
              <tbody>
                {(tickets.data.tickets || []).map((t) => (
                  <tr key={t.id} className="border-b border-slate-50 hover:bg-slate-50">
                    <td className="py-2 pr-3 font-mono text-xs text-slate-500">{t.ticketNumber}</td>
                    <td className="py-2 pr-3 font-medium text-slate-800">{t.subject}</td>
                    <td className="py-2 pr-3"><Badge>{t.status}</Badge></td>
                    <td className="py-2 pr-3 text-slate-600">{t.priority || '—'}</td>
                    <td className="py-2 pr-3 text-slate-600">{t.contact || '—'}</td>
                    <td className="py-2 text-slate-500">{fmtDate(t.modifiedTime)}</td>
                  </tr>
                ))}
                {tickets.data.tickets?.length === 0 && (
                  <tr><td colSpan="6" className="py-6 text-center text-slate-400">No tickets found.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
