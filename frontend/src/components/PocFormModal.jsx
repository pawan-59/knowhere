import { useState } from 'react'
import { api } from '../lib/api'
import { Modal, ErrorBox } from './ui'

const STATUSES = ['in_progress', 'signed', 'freezer']
const STATUS_LABELS = { in_progress: 'In Progress', signed: 'Signed', freezer: 'Freezer' }

const PHASES = [
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

const empty = {
  company: '', shortCode: '', owner: '', primaryContact: '', status: 'in_progress',
  phase: PHASES[0], progress: 0, targetDate: '', notes: '',
}

function toFormState(o) {
  return o ? {
    company: o.company, shortCode: o.shortCode || '', owner: o.owner || '', primaryContact: o.primaryContact || '',
    status: o.status, phase: o.phase || PHASES[0], progress: o.progress,
    // The date input needs "YYYY-MM-DD"; the API returns/expects RFC3339.
    targetDate: o.targetDate ? o.targetDate.slice(0, 10) : '',
    notes: o.notes || '',
  } : empty
}

// PocFormModal is the shared add/edit form for a POC — used by both the
// Onboarding dashboard ("+ Add POC" / card edit) and the comprehensive list
// page (card edit), so the two stay in sync with a single implementation.
export function PocFormModal({ initial, onClose, onSaved }) {
  const [form, setForm] = useState(() => toFormState(initial))
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState(null)

  async function save(e) {
    e.preventDefault()
    setSaving(true); setErr(null)
    try {
      const saved = await api.saveOnboarding({
        ...form,
        progress: Number(form.progress) || 0,
        targetDate: form.targetDate ? new Date(form.targetDate).toISOString() : null,
      })
      onSaved(saved)
    } catch (e) { setErr(e) } finally { setSaving(false) }
  }

  return (
    <Modal title={initial ? `Edit: ${initial.company}` : 'Add a POC'} onClose={onClose}>
      <form onSubmit={save} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <Field label="Company *">
          <input required className={inp} value={form.company} onChange={(e) => setForm({ ...form, company: e.target.value })} />
        </Field>
        <Field label="Short code (card badge)">
          <input className={inp} placeholder="e.g. AMZ" value={form.shortCode} onChange={(e) => setForm({ ...form, shortCode: e.target.value.toUpperCase() })} />
        </Field>
        <Field label="Owner (CSM)">
          <input className={inp} value={form.owner} onChange={(e) => setForm({ ...form, owner: e.target.value })} />
        </Field>
        <Field label="Primary contact">
          <input className={inp} value={form.primaryContact} onChange={(e) => setForm({ ...form, primaryContact: e.target.value })} />
        </Field>
        <Field label="Status">
          <select className={inp} value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
            {STATUSES.map((s) => <option key={s} value={s}>{STATUS_LABELS[s]}</option>)}
          </select>
        </Field>
        <Field label="Phase">
          <select className={inp} value={form.phase} onChange={(e) => setForm({ ...form, phase: e.target.value })}>
            {PHASES.map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
        </Field>
        <Field label="Progress (%)">
          <input type="number" min="0" max="100" className={inp} value={form.progress} onChange={(e) => setForm({ ...form, progress: e.target.value })} />
        </Field>
        <Field label="Target date">
          <input type="date" className={inp} value={form.targetDate} onChange={(e) => setForm({ ...form, targetDate: e.target.value })} />
        </Field>
        <Field label="Notes" full>
          <input className={inp} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
        </Field>
        {err && <div className="sm:col-span-2"><ErrorBox error={err} /></div>}
        <div className="sm:col-span-2 flex items-center gap-3">
          <button disabled={saving} className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button type="button" onClick={onClose} className="text-sm text-slate-500 hover:text-slate-700">Cancel</button>
        </div>
      </form>
    </Modal>
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
