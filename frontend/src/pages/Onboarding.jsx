import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { Spinner, ErrorBox, useAsync, PocCard, SearchInput } from '../components/ui'
import { PocFormModal } from '../components/PocFormModal'

// Tailwind classes per stat card tone — Total=blue (Devtron blue), In Progress=yellow,
// Signed=green, Freezer=red.
const TONES = {
  blue: { bg: 'bg-blue-50', border: 'border-blue-200', icon: 'bg-blue-600 text-white', value: 'text-blue-700' },
  yellow: { bg: 'bg-amber-50', border: 'border-amber-200', icon: 'bg-amber-500 text-white', value: 'text-amber-700' },
  green: { bg: 'bg-emerald-50', border: 'border-emerald-200', icon: 'bg-emerald-600 text-white', value: 'text-emerald-700' },
  red: { bg: 'bg-rose-50', border: 'border-rose-200', icon: 'bg-rose-600 text-white', value: 'text-rose-700' },
}

const STAT_CARDS = [
  { key: 'total', icon: '▥', label: 'Total POCs', tone: 'blue' },
  { key: 'inProgress', icon: '⟳', label: 'In Progress', tone: 'yellow' },
  { key: 'signed', icon: '✓', label: 'Signed', tone: 'green' },
  { key: 'freezer', icon: '❄', label: 'Freezer', tone: 'red' },
]

export default function Onboarding() {
  const summary = useAsync(api.onboardingSummary, [])
  const list = useAsync(() => api.onboardings(), [])
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRecord, setEditingRecord] = useState(null)

  const reload = () => { summary.reload(); list.reload() }

  function openAdd() { setEditingRecord(null); setModalOpen(true) }
  function openEdit(o) { setEditingRecord(o); setModalOpen(true) }

  async function quickDelete(o) {
    if (!confirm(`Delete ${o.company}?`)) return
    await api.deleteOnboarding(o.shortCode)
    reload()
  }

  const activeAll = (list.data?.onboardings || []).filter((o) => o.status !== 'freezer')
  const q = search.trim().toLowerCase()
  const matches = q
    ? activeAll.filter((o) => o.company.toLowerCase().includes(q) || (o.primaryContact || '').toLowerCase().includes(q))
    : activeAll
  // Without a search query, cap to the 6 most recently updated; a query shows every match.
  const active = q ? matches : matches.slice(0, 6)

  return (
    <div className="space-y-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Onboarding</h1>
          <p className="text-sm text-slate-500">Track every POC from kickoff to sign-off.</p>
        </div>
        <button onClick={openAdd} className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700">
          + Add POC
        </button>
      </header>

      {summary.error && <ErrorBox error={summary.error} />}
      {summary.data && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          {STAT_CARDS.map((c) => (
            <StatCard key={c.key} icon={c.icon} label={c.label} value={summary.data[c.key]} tone={c.tone} />
          ))}
        </div>
      )}

      <div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-xl font-bold text-slate-900">Active POCs</h2>
          <div className="flex items-center gap-4">
            <SearchInput value={search} onChange={setSearch} placeholder="Search active POCs…" className="w-56" />
            <Link to="/onboarding/list" className="whitespace-nowrap text-sm font-medium text-blue-600 hover:text-blue-700">
              View comprehensive list ›
            </Link>
          </div>
        </div>

        {list.loading ? <Spinner /> : list.error ? <ErrorBox error={list.error} /> : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {active.map((o) => (
              <PocCard
                key={o.id}
                o={o}
                onView={() => navigate(`/onboarding/${o.shortCode}`)}
                onAddLog={() => navigate(`/onboarding/${o.shortCode}`)}
                onEdit={() => openEdit(o)}
                onDelete={() => quickDelete(o)}
              />
            ))}
            {active.length === 0 && (
              <div className="col-span-full rounded-xl border border-dashed border-slate-200 py-10 text-center text-slate-400">
                {q ? 'No active POCs match your search.' : 'No active POCs yet — add one above.'}
              </div>
            )}
          </div>
        )}
      </div>

      {modalOpen && (
        <PocFormModal
          initial={editingRecord}
          onClose={() => setModalOpen(false)}
          onSaved={() => { setModalOpen(false); reload() }}
        />
      )}
    </div>
  )
}

function StatCard({ icon, value, label, tone }) {
  const t = TONES[tone]
  return (
    <div className={`rounded-xl border p-5 ${t.bg} ${t.border}`}>
      <div className={`mb-4 flex h-10 w-10 items-center justify-center rounded-lg text-lg ${t.icon}`}>{icon}</div>
      <div className={`text-3xl font-bold ${t.value}`}>{value ?? '—'}</div>
      <div className="text-sm text-slate-600">{label}</div>
    </div>
  )
}
