import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../lib/api'
import { Card, Badge, ErrorBox, Spinner, useAsync, fmtDate, fmtRelative } from '../components/ui'

const CONTACT_TYPES = ['call', 'email', 'chat']
const CONTACT_TYPE_LABELS = { call: 'Call', email: 'Email', chat: 'Chat' }

const todayStr = () => new Date().toISOString().slice(0, 10)
const emptyLogForm = { contactDate: todayStr(), contactType: 'call', reachedBy: '', contactPerson: '', description: '' }

// Full-page activity report for one POC — details plus the complete
// call/update log history, with a structured form to add new entries.
export default function OnboardingReport() {
  const { code } = useParams()
  const poc = useAsync(() => api.onboarding(code), [code])
  const logs = useAsync(() => api.onboardingLogs(code), [code])
  const [logForm, setLogForm] = useState(emptyLogForm)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState(null)

  // Default "who reached" / "with whom" from the POC's owner/primary contact once loaded.
  useEffect(() => {
    if (!poc.data) return
    setLogForm((f) => ({
      ...f,
      reachedBy: f.reachedBy || poc.data.owner || '',
      contactPerson: f.contactPerson || poc.data.primaryContact || '',
    }))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [poc.data])

  async function addLog(e) {
    e.preventDefault()
    if (!logForm.description.trim()) return
    setSaving(true); setErr(null)
    try {
      await api.addOnboardingLog(code, {
        contactDate: logForm.contactDate ? new Date(logForm.contactDate).toISOString() : null,
        contactType: logForm.contactType,
        reachedBy: logForm.reachedBy,
        contactPerson: logForm.contactPerson,
        description: logForm.description.trim(),
      })
      setLogForm((f) => ({ ...f, description: '' }))
      logs.reload()
    } catch (e) { setErr(e) } finally { setSaving(false) }
  }

  if (poc.loading) return <Spinner />
  if (poc.error) return <ErrorBox error={poc.error} />

  const p = poc.data

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{p.company} — Activity Log</h1>
          <p className="text-sm text-slate-500">Full report and call/update history for this POC.</p>
        </div>
        <Link to="/onboarding/list" className="text-sm font-medium text-blue-600 hover:text-blue-700">‹ Back to All POCs</Link>
      </header>

      <Card title="Details">
        <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-3">
          <Info label="Status"><Badge>{p.status}</Badge></Info>
          <Info label="Phase">{p.phase || '—'}</Info>
          <Info label="Progress">{p.progress}%</Info>
          <Info label="Owner (CSM)">{p.owner || '—'}</Info>
          <Info label="Primary contact">{p.primaryContact || '—'}</Info>
          <Info label="Target date">{fmtDate(p.targetDate)}</Info>
          {p.notes && <Info label="Notes" full>{p.notes}</Info>}
        </div>
      </Card>

      <Card title="Add a log entry">
        <form onSubmit={addLog} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Date">
            <input type="date" className={inp} value={logForm.contactDate} onChange={(e) => setLogForm({ ...logForm, contactDate: e.target.value })} />
          </Field>
          <Field label="Type of contact">
            <select className={inp} value={logForm.contactType} onChange={(e) => setLogForm({ ...logForm, contactType: e.target.value })}>
              {CONTACT_TYPES.map((t) => <option key={t} value={t}>{CONTACT_TYPE_LABELS[t]}</option>)}
            </select>
          </Field>
          <Field label="Who reached">
            <input className={inp} placeholder="e.g. Alex CSM" value={logForm.reachedBy} onChange={(e) => setLogForm({ ...logForm, reachedBy: e.target.value })} />
          </Field>
          <Field label="With whom">
            <input className={inp} placeholder="e.g. Jane Doe" value={logForm.contactPerson} onChange={(e) => setLogForm({ ...logForm, contactPerson: e.target.value })} />
          </Field>
          <Field label="Description" full>
            <textarea
              className={inp}
              rows={3}
              placeholder="What was discussed…"
              value={logForm.description}
              onChange={(e) => setLogForm({ ...logForm, description: e.target.value })}
            />
          </Field>
          {err && <div className="sm:col-span-2"><ErrorBox error={err} /></div>}
          <div className="sm:col-span-2">
            <button
              disabled={saving || !logForm.description.trim()}
              className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"
            >
              {saving ? 'Logging…' : 'Add Log'}
            </button>
          </div>
        </form>
      </Card>

      <Card title="History">
        {logs.loading ? <Spinner /> : logs.error ? <ErrorBox error={logs.error} /> : (
          <div className="space-y-3">
            {(logs.data?.logs || []).map((l) => (
              <div key={l.id} className="rounded-lg border border-slate-100 px-3 py-2 text-sm">
                <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                  <Badge>{l.contactType}</Badge>
                  <span>{fmtDate(l.contactDate)}</span>
                  {(l.reachedBy || l.contactPerson) && (
                    <span>{l.reachedBy || '—'} → {l.contactPerson || '—'}</span>
                  )}
                </div>
                <div className="mt-1.5 whitespace-pre-wrap text-slate-700">{l.description}</div>
                <div className="mt-1 text-xs text-slate-400">Logged {fmtRelative(l.createdAt)}</div>
              </div>
            ))}
            {logs.data?.logs?.length === 0 && (
              <div className="py-6 text-center text-sm text-slate-400">No log entries yet — add the first one above.</div>
            )}
          </div>
        )}
      </Card>
    </div>
  )
}

function Info({ label, children, full }) {
  return (
    <div className={full ? 'col-span-full' : ''}>
      <div className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</div>
      <div className="mt-0.5 text-slate-800">{children}</div>
    </div>
  )
}

const inp = 'w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none focus:border-slate-400'
function Field({ label, children, full }) {
  return (
    <label className={`block ${full ? 'sm:col-span-2' : ''}`}>
      <span className="mb-1 block text-xs font-medium text-slate-500">{label}</span>
      {children}
    </label>
  )
}
