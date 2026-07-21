import { useState } from 'react'
import { api } from '../lib/api'
import { Card, Stat, Badge, Spinner, ErrorBox, useAsync, fmtDate } from '../components/ui'

const STATUSES = ['active', 'trial', 'expired', 'suspended']

const empty = { customer: '', installation: '', edition: 'enterprise', seats: 0, seatsUsed: 0, status: 'active', expiresAt: '', devtronVersion: '', notes: '' }

export default function License() {
  const summary = useAsync(api.licenseSummary, [])
  const list = useAsync(() => api.licenses(), [])
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
        seats: Number(form.seats) || 0,
        seatsUsed: Number(form.seatsUsed) || 0,
        expiresAt: form.expiresAt ? new Date(form.expiresAt).toISOString() : null,
      }
      await api.saveLicense(payload)
      setForm(empty)
      reload()
    } catch (e) { setErr(e) } finally { setSaving(false) }
  }

  async function remove(id) {
    if (!confirm('Delete this license record?')) return
    await api.deleteLicense(id)
    reload()
  }

  function edit(l) {
    setForm({
      customer: l.customer, installation: l.installation, edition: l.edition,
      seats: l.seats, seatsUsed: l.seatsUsed, status: l.status,
      expiresAt: l.expiresAt ? l.expiresAt.slice(0, 10) : '',
      devtronVersion: l.devtronVersion || '', notes: l.notes || '',
    })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function expiryTone(days) {
    if (days == null) return 'default'
    if (days < 0) return 'bad'
    if (days < 30) return 'warn'
    return 'good'
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-slate-900">License Monitoring</h1>
        <p className="text-sm text-slate-500">Devtron installation licenses, seats, and expiry.</p>
      </header>

      {summary.data && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
          <Stat label="Total" value={summary.data.total} />
          <Stat label="Expiring <30d" value={summary.data.expiringSoon} tone="warn" />
          <Stat label="Expired" value={summary.data.expired} tone="bad" />
          <Stat label="Seats" value={summary.data.totalSeats} />
          <Stat label="Seats used" value={summary.data.totalSeatsUsed} hint={`${summary.data.totalSeats - summary.data.totalSeatsUsed} free`} />
        </div>
      )}

      <Card title={form.customer ? `Edit / add: ${form.customer}` : 'Add / update license'}>
        <form onSubmit={save} className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <Field label="Customer *"><input required className={inp} value={form.customer} onChange={(e) => setForm({ ...form, customer: e.target.value })} /></Field>
          <Field label="Installation *"><input required className={inp} value={form.installation} onChange={(e) => setForm({ ...form, installation: e.target.value })} /></Field>
          <Field label="Edition"><input className={inp} value={form.edition} onChange={(e) => setForm({ ...form, edition: e.target.value })} /></Field>
          <Field label="Seats"><input type="number" min="0" className={inp} value={form.seats} onChange={(e) => setForm({ ...form, seats: e.target.value })} /></Field>
          <Field label="Seats used"><input type="number" min="0" className={inp} value={form.seatsUsed} onChange={(e) => setForm({ ...form, seatsUsed: e.target.value })} /></Field>
          <Field label="Status"><select className={inp} value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>{STATUSES.map((s) => <option key={s}>{s}</option>)}</select></Field>
          <Field label="Expires at"><input type="date" className={inp} value={form.expiresAt} onChange={(e) => setForm({ ...form, expiresAt: e.target.value })} /></Field>
          <Field label="Devtron version"><input className={inp} value={form.devtronVersion} onChange={(e) => setForm({ ...form, devtronVersion: e.target.value })} /></Field>
          <Field label="Notes"><input className={inp} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="sm:col-span-3 flex items-center gap-3">
            <button disabled={saving} className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50">{saving ? 'Saving…' : 'Save'}</button>
            {form.customer && <button type="button" onClick={() => setForm(empty)} className="text-sm text-slate-500 hover:text-slate-700">Clear</button>}
          </div>
          {err && <div className="sm:col-span-3"><ErrorBox error={err} /></div>}
        </form>
      </Card>

      <Card title="Licenses">
        {list.loading ? <Spinner /> : list.error ? <ErrorBox error={list.error} /> : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs uppercase text-slate-400">
                  <th className="py-2 pr-3">Customer</th><th className="py-2 pr-3">Installation</th>
                  <th className="py-2 pr-3">Status</th><th className="py-2 pr-3">Seats</th>
                  <th className="py-2 pr-3">Version</th><th className="py-2 pr-3">Expires</th><th className="py-2"></th>
                </tr>
              </thead>
              <tbody>
                {(list.data.licenses || []).map((l) => (
                  <tr key={l.id} className="border-b border-slate-50 hover:bg-slate-50">
                    <td className="py-2 pr-3 font-medium text-slate-800">{l.customer}</td>
                    <td className="py-2 pr-3 text-slate-600">{l.installation}</td>
                    <td className="py-2 pr-3"><Badge>{l.status}</Badge></td>
                    <td className="py-2 pr-3 text-slate-600">{l.seatsUsed}/{l.seats}</td>
                    <td className="py-2 pr-3 font-mono text-xs text-slate-500">{l.devtronVersion || '—'}</td>
                    <td className={`py-2 pr-3 ${tone(expiryTone(l.daysToExpiry))}`}>
                      {fmtDate(l.expiresAt)}{l.daysToExpiry != null && <span className="ml-1 text-xs">({l.daysToExpiry}d)</span>}
                    </td>
                    <td className="py-2 text-right whitespace-nowrap">
                      <button onClick={() => edit(l)} className="text-xs text-slate-500 hover:text-slate-800">Edit</button>
                      <button onClick={() => remove(l.id)} className="ml-3 text-xs text-rose-500 hover:text-rose-700">Delete</button>
                    </td>
                  </tr>
                ))}
                {list.data.licenses?.length === 0 && (
                  <tr><td colSpan="7" className="py-6 text-center text-slate-400">No licenses yet — add one above.</td></tr>
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
function tone(t) {
  return { good: 'text-emerald-600', warn: 'text-amber-600', bad: 'text-rose-600 font-medium', default: 'text-slate-500' }[t]
}
function Field({ label, children }) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-slate-500">{label}</span>
      {children}
    </label>
  )
}
