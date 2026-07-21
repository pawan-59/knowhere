import { useState } from 'react'
import { api } from '../lib/api'
import { Card, Stat, Badge, Progress, Spinner, ErrorBox, useAsync, fmtDate } from '../components/ui'

// Onboarding pipeline stages, in order. The two "Blocked On …" stages require a
// reason (see BLOCKED_STAGES / the conditional field below).
const STAGES = [
  'Discovery Call',
  'Demo Call',
  'Kickoff',
  'Success Criteria',
  'Infra Provisioning',
  'Devtron Stack Onboarding',
  'Sanity app deployed (with all needed modules)',
  'Configured Essentials Credentials',
  'Deployed 1st App',
  'Handed Over',
  'Blocked On Devtron',
  'Blocked On Client',
]
const BLOCKED_STAGES = ['Blocked On Devtron', 'Blocked On Client']
const isBlocked = (stage) => BLOCKED_STAGES.includes(stage)

const STATUSES = ['on_track', 'at_risk', 'blocked', 'completed']

const empty = { customer: '', owner: '', stage: 'Discovery Call', status: 'on_track', progress: 0, targetDate: '', notes: '', blockedReason: '' }

export default function Onboarding() {
  const summary = useAsync(api.onboardingSummary, [])
  const list = useAsync(() => api.onboardings(), [])
  const [form, setForm] = useState(empty)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState(null)

  const reload = () => { summary.reload(); list.reload() }

  async function save(e) {
    e.preventDefault()
    setSaving(true); setErr(null)
    try {
      const payload = {
        ...form,
        progress: Number(form.progress) || 0,
        // The date input yields "YYYY-MM-DD"; the API expects an RFC3339
        // timestamp (Go time.Time). Convert, or send null when empty.
        targetDate: form.targetDate ? new Date(form.targetDate).toISOString() : null,
        // Only a "Blocked On …" stage carries a blocking reason; clear it otherwise.
        blockedReason: isBlocked(form.stage) ? form.blockedReason : null,
      }
      await api.saveOnboarding(payload)
      setForm(empty)
      reload()
    } catch (e) { setErr(e) } finally { setSaving(false) }
  }

  async function remove(id) {
    if (!confirm('Delete this onboarding record?')) return
    await api.deleteOnboarding(id)
    reload()
  }

  function edit(o) {
    setForm({
      customer: o.customer, owner: o.owner || '', stage: o.stage, status: o.status,
      progress: o.progress, targetDate: o.targetDate ? o.targetDate.slice(0, 10) : '', notes: o.notes || '',
      blockedReason: o.blockedReason || '',
    })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-slate-900">Onboarding Status</h1>
        <p className="text-sm text-slate-500">Track every customer from discovery call to handover.</p>
      </header>

      {summary.data && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Stat label="Active" value={summary.data.total} />
          <Stat label="Avg progress" value={`${summary.data.avgProgress}%`} />
          <Stat label="At risk" value={summary.data.atRisk} tone="warn" />
          <Stat label="Blocked" value={summary.data.blocked} tone="bad" />
        </div>
      )}

      <Card title={form.customer ? `Edit / add: ${form.customer}` : 'Add / update onboarding'}>
        <form onSubmit={save} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Customer *"><input required className={inp} value={form.customer} onChange={(e) => setForm({ ...form, customer: e.target.value })} /></Field>
          <Field label="Owner (CSM)"><input className={inp} value={form.owner} onChange={(e) => setForm({ ...form, owner: e.target.value })} /></Field>
          <Field label="Stage"><select className={inp} value={form.stage} onChange={(e) => setForm({ ...form, stage: e.target.value })}>{STAGES.map((s) => <option key={s}>{s}</option>)}</select></Field>
          <Field label="Status"><select className={inp} value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>{STATUSES.map((s) => <option key={s}>{s}</option>)}</select></Field>
          {isBlocked(form.stage) && (
            <Field label="Blocked — where exactly? *" full>
              <input
                required className={inp}
                placeholder="e.g. waiting on client firewall change, or Devtron support ticket #123"
                value={form.blockedReason}
                onChange={(e) => setForm({ ...form, blockedReason: e.target.value })}
              />
            </Field>
          )}
          <Field label="Progress (%)"><input type="number" min="0" max="100" className={inp} value={form.progress} onChange={(e) => setForm({ ...form, progress: e.target.value })} /></Field>
          <Field label="Target date"><input type="date" className={inp} value={form.targetDate} onChange={(e) => setForm({ ...form, targetDate: e.target.value })} /></Field>
          <Field label="Notes" full><input className={inp} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="sm:col-span-2 flex items-center gap-3">
            <button disabled={saving} className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50">{saving ? 'Saving…' : 'Save'}</button>
            {form.customer && <button type="button" onClick={() => setForm(empty)} className="text-sm text-slate-500 hover:text-slate-700">Clear</button>}
          </div>
          {err && <div className="sm:col-span-2"><ErrorBox error={err} /></div>}
        </form>
      </Card>

      <Card title="Customers">
        {list.loading ? <Spinner /> : list.error ? <ErrorBox error={list.error} /> : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs uppercase text-slate-400">
                  <th className="py-2 pr-3">Customer</th><th className="py-2 pr-3">Owner</th>
                  <th className="py-2 pr-3">Stage</th><th className="py-2 pr-3">Status</th>
                  <th className="py-2 pr-3 w-40">Progress</th><th className="py-2 pr-3">Target</th><th className="py-2"></th>
                </tr>
              </thead>
              <tbody>
                {(list.data.onboardings || []).map((o) => (
                  <tr key={o.id} className="border-b border-slate-50 hover:bg-slate-50">
                    <td className="py-2 pr-3 font-medium text-slate-800">{o.customer}</td>
                    <td className="py-2 pr-3 text-slate-600">{o.owner || '—'}</td>
                    <td className="py-2 pr-3">
                      <Badge>{o.stage}</Badge>
                      {o.blockedReason && <div className="mt-0.5 max-w-xs text-xs text-rose-600">⚠ {o.blockedReason}</div>}
                    </td>
                    <td className="py-2 pr-3"><Badge>{o.status}</Badge></td>
                    <td className="py-2 pr-3"><Progress value={o.progress} /></td>
                    <td className="py-2 pr-3 text-slate-500">{fmtDate(o.targetDate)}</td>
                    <td className="py-2 text-right whitespace-nowrap">
                      <button onClick={() => edit(o)} className="text-xs text-slate-500 hover:text-slate-800">Edit</button>
                      <button onClick={() => remove(o.id)} className="ml-3 text-xs text-rose-500 hover:text-rose-700">Delete</button>
                    </td>
                  </tr>
                ))}
                {list.data.onboardings?.length === 0 && (
                  <tr><td colSpan="7" className="py-6 text-center text-slate-400">No records yet — add one above.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>
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
