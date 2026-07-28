// Small reusable UI primitives used across all dashboard modules.
import { useEffect, useState } from 'react'

export function Card({ title, action, children, className = '' }) {
  return (
    <div className={`rounded-xl border border-slate-200 bg-white shadow-sm ${className}`}>
      {(title || action) && (
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3">
          <h3 className="text-sm font-semibold text-slate-700">{title}</h3>
          {action}
        </div>
      )}
      <div className="p-5">{children}</div>
    </div>
  )
}

export function Modal({ title, onClose, children }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4" onClick={onClose}>
      <div className="w-full max-w-lg rounded-xl bg-white shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3">
          <h3 className="text-sm font-semibold text-slate-700">{title}</h3>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600">✕</button>
        </div>
        <div className="max-h-[80vh] overflow-y-auto p-5">{children}</div>
      </div>
    </div>
  )
}

// SearchInput is a small text input with a search icon, used to filter POC lists client-side.
export function SearchInput({ value, onChange, placeholder = 'Search…', className = '' }) {
  return (
    <div className={`relative ${className}`}>
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">🔍</span>
      <input
        className="w-full rounded-lg border border-slate-200 py-1.5 pl-9 pr-3 text-sm outline-none focus:border-slate-400"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}

export function Stat({ label, value, hint, tone = 'default' }) {
  const tones = {
    default: 'text-slate-900',
    good: 'text-emerald-600',
    warn: 'text-amber-600',
    bad: 'text-rose-600',
  }
  return (
    <div className="rounded-lg bg-slate-50 px-4 py-3">
      <div className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</div>
      <div className={`mt-1 text-2xl font-bold ${tones[tone]}`}>{value ?? '—'}</div>
      {hint && <div className="mt-0.5 text-xs text-slate-400">{hint}</div>}
    </div>
  )
}

const badgeTones = {
  active: 'bg-emerald-100 text-emerald-700',
  live: 'bg-emerald-100 text-emerald-700',
  on_track: 'bg-emerald-100 text-emerald-700',
  completed: 'bg-emerald-100 text-emerald-700',
  Healthy: 'bg-emerald-100 text-emerald-700',
  Open: 'bg-blue-100 text-blue-700',
  trial: 'bg-blue-100 text-blue-700',
  at_risk: 'bg-amber-100 text-amber-700',
  'On Hold': 'bg-amber-100 text-amber-700',
  Progressing: 'bg-amber-100 text-amber-700',
  expired: 'bg-rose-100 text-rose-700',
  blocked: 'bg-rose-100 text-rose-700',
  Escalated: 'bg-rose-100 text-rose-700',
  suspended: 'bg-rose-100 text-rose-700',
  Degraded: 'bg-rose-100 text-rose-700',
  // Onboarding / POC status
  in_progress: 'bg-amber-100 text-amber-700',
  signed: 'bg-emerald-100 text-emerald-700',
  freezer: 'bg-rose-100 text-rose-700',
  // Activity log contact type
  call: 'bg-blue-100 text-blue-700',
  email: 'bg-purple-100 text-purple-700',
  chat: 'bg-teal-100 text-teal-700',
}

// POC card progress-bar color, keyed by status — mirrors the badge tones above.
export const POC_BAR_COLOR = {
  in_progress: 'bg-amber-500',
  signed: 'bg-emerald-500',
  freezer: 'bg-rose-500',
}

export function Badge({ children }) {
  const tone = badgeTones[children] || 'bg-slate-100 text-slate-600'
  return (
    <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${tone}`}>
      {String(children).replace(/_/g, ' ')}
    </span>
  )
}

export function Progress({ value = 0 }) {
  const v = Math.max(0, Math.min(100, value))
  const color = v >= 80 ? 'bg-emerald-500' : v >= 40 ? 'bg-amber-500' : 'bg-rose-500'
  return (
    <div className="flex items-center gap-2">
      <div className="h-2 w-full rounded-full bg-slate-100">
        <div className={`h-2 rounded-full ${color}`} style={{ width: `${v}%` }} />
      </div>
      <span className="w-9 text-right text-xs text-slate-500">{v}%</span>
    </div>
  )
}

export function Spinner() {
  return (
    <div className="flex items-center justify-center py-10 text-slate-400">
      <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-500" />
    </div>
  )
}

export function ErrorBox({ error }) {
  if (!error) return null
  const notConfigured = error.status === 503
  return (
    <div className={`rounded-lg border px-4 py-3 text-sm ${notConfigured ? 'border-amber-200 bg-amber-50 text-amber-800' : 'border-rose-200 bg-rose-50 text-rose-700'}`}>
      <div className="font-medium">{error.message}</div>
      {error.details && <div className="mt-1 text-xs opacity-80">{error.details}</div>}
    </div>
  )
}

// useAsync runs an async fn on mount (and when deps change) with loading/error state.
export function useAsync(fn, deps = []) {
  const [state, setState] = useState({ loading: true, data: null, error: null })
  const [nonce, setNonce] = useState(0)
  useEffect(() => {
    let alive = true
    setState((s) => ({ ...s, loading: true }))
    fn()
      .then((data) => alive && setState({ loading: false, data, error: null }))
      .catch((error) => alive && setState({ loading: false, data: null, error }))
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])
  return { ...state, reload: () => setNonce((n) => n + 1) }
}

export function fmtDate(s) {
  if (!s) return '—'
  const d = new Date(s)
  return isNaN(d) ? '—' : d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// fmtRelative renders a short "time ago" label (e.g. "2h ago"), falling back
// to fmtDate once it's more than a month old.
export function fmtRelative(s) {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d)) return '—'
  const mins = Math.round((Date.now() - d.getTime()) / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days < 30) return `${days}d ago`
  return fmtDate(s)
}

// PocCard renders one onboarding/POC record — used on both the Onboarding
// dashboard (active POCs) and the comprehensive list page. Clicking the card
// (outside the "..." menu) opens its full activity report via onView.
export function PocCard({ o, onView, onEdit, onDelete, onAddLog }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const progress = Math.max(0, Math.min(100, o.progress))
  const barColor = POC_BAR_COLOR[o.status] || 'bg-blue-600'

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onView}
      onKeyDown={(e) => { if (e.key === 'Enter') onView() }}
      className="cursor-pointer rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-slate-300 hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-bold text-slate-600">
            {o.shortCode}
          </div>
          <div className="min-w-0">
            <div className="truncate font-semibold text-slate-900">{o.company}</div>
            <div className="text-xs text-slate-400">Updated {fmtRelative(o.updatedAt)}</div>
          </div>
        </div>
        <Badge>{o.status}</Badge>
      </div>

      <div className="mt-5 flex items-center justify-between text-sm">
        <span className="truncate text-slate-600">{o.phase || '—'}</span>
        <span className="ml-2 shrink-0 font-medium text-slate-700">{progress}%</span>
      </div>
      <div className="mt-1.5 h-2 w-full rounded-full bg-slate-100">
        <div className={`h-2 rounded-full ${barColor}`} style={{ width: `${progress}%` }} />
      </div>

      <div className="mt-5 flex items-center justify-between border-t border-slate-100 pt-3 text-xs">
        <span className="truncate text-slate-500">
          Primary Contact: <span className="text-slate-700">{o.primaryContact || '—'}</span>
        </span>
        <div
          className="relative shrink-0"
          onClick={(e) => e.stopPropagation()}
          onBlur={(e) => { if (!e.currentTarget.contains(e.relatedTarget)) setMenuOpen(false) }}
        >
          <button onClick={() => setMenuOpen((v) => !v)} className="px-2 text-slate-400 hover:text-slate-700">•••</button>
          {menuOpen && (
            <div className="absolute right-0 z-10 mt-1 w-32 rounded-lg border border-slate-200 bg-white py-1 shadow-lg">
              <button onClick={() => { setMenuOpen(false); onAddLog() }} className="block w-full px-3 py-1.5 text-left text-slate-600 hover:bg-slate-50">Add Log</button>
              <button onClick={() => { setMenuOpen(false); onEdit() }} className="block w-full px-3 py-1.5 text-left text-slate-600 hover:bg-slate-50">Edit</button>
              <button onClick={() => { setMenuOpen(false); onDelete() }} className="block w-full px-3 py-1.5 text-left text-rose-600 hover:bg-rose-50">Delete</button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
