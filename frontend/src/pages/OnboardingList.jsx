import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { SearchInput, Spinner, ErrorBox, useAsync, PocCard } from '../components/ui'
import { PocFormModal } from '../components/PocFormModal'

const STATUSES = ['in_progress', 'signed', 'freezer']
const STATUS_LABELS = { in_progress: 'In Progress', signed: 'Signed', freezer: 'Freezer' }

export default function OnboardingList() {
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState('')
  const [search, setSearch] = useState('')
  const list = useAsync(() => api.onboardings(statusFilter ? `?status=${statusFilter}` : ''), [statusFilter])
  const [editingRecord, setEditingRecord] = useState(null)

  async function remove(o) {
    if (!confirm(`Delete ${o.company}?`)) return
    await api.deleteOnboarding(o.shortCode)
    list.reload()
  }

  const q = search.trim().toLowerCase()
  const visible = useMemo(() => {
    const items = list.data?.onboardings || []
    if (!q) return items
    return items.filter((o) => o.company.toLowerCase().includes(q) || (o.primaryContact || '').toLowerCase().includes(q))
  }, [list.data, q])

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">All POCs</h1>
          <p className="text-sm text-slate-500">Full pipeline — browse, edit, and manage every POC.</p>
        </div>
        <Link to="/onboarding" className="text-sm font-medium text-blue-600 hover:text-blue-700">‹ Back to overview</Link>
      </header>

      <div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-xl font-bold text-slate-900">All POCs</h2>
          <div className="flex items-center gap-2">
            <SearchInput value={search} onChange={setSearch} placeholder="Search company or contact…" className="w-56" />
            <select className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
              <option value="">All statuses</option>
              {STATUSES.map((s) => <option key={s} value={s}>{STATUS_LABELS[s]}</option>)}
            </select>
          </div>
        </div>

        {list.loading ? <Spinner /> : list.error ? <ErrorBox error={list.error} /> : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {visible.map((o) => (
              <PocCard
                key={o.id}
                o={o}
                onView={() => navigate(`/onboarding/${o.shortCode}`)}
                onAddLog={() => navigate(`/onboarding/${o.shortCode}`)}
                onEdit={() => setEditingRecord(o)}
                onDelete={() => remove(o)}
              />
            ))}
            {visible.length === 0 && (
              <div className="col-span-full rounded-xl border border-dashed border-slate-200 py-10 text-center text-slate-400">
                {q || statusFilter ? 'No POCs match your search/filter.' : 'No records yet — add one from the Onboarding overview.'}
              </div>
            )}
          </div>
        )}
      </div>

      {editingRecord && (
        <PocFormModal
          initial={editingRecord}
          onClose={() => setEditingRecord(null)}
          onSaved={() => { setEditingRecord(null); list.reload() }}
        />
      )}
    </div>
  )
}
