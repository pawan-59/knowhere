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
  // Onboarding pipeline stages
  'Blocked On Devtron': 'bg-rose-100 text-rose-700',
  'Blocked On Client': 'bg-rose-100 text-rose-700',
  'Handed Over': 'bg-emerald-100 text-emerald-700',
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
