import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from './lib/auth'

const nav = [
  { to: '/', label: 'Overview', end: true, icon: '◧' },
  { to: '/zoho', label: 'Zoho Desk', icon: '✉' },
  { to: '/devtron', label: 'Devtron Releases', icon: '⎈' },
  { to: '/onboarding', label: 'Onboarding', icon: '➜' },
  { to: '/licenses', label: 'Licenses', icon: '🔑' },
]

export default function App() {
  const { user, logout } = useAuth()
  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 flex-col bg-slate-900 text-slate-300">
        <div className="px-5 py-5">
          <div className="text-lg font-bold text-white">Central<span className="text-emerald-400">·</span>Devtron</div>
          <div className="text-xs text-slate-500">Unified operations dashboard</div>
        </div>
        <nav className="flex-1 space-y-1 px-3">
          {nav.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              end={n.end}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                  isActive ? 'bg-slate-800 text-white' : 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
                }`
              }
            >
              <span className="w-4 text-center">{n.icon}</span>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t border-slate-800 px-4 py-4">
          <div className="mb-2 truncate text-sm text-slate-300" title={user?.email}>
            {user?.name || user?.email}
          </div>
          <div className="mb-3 truncate text-xs text-slate-500">{user?.email}</div>
          <button
            onClick={logout}
            className="w-full rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 hover:bg-slate-800 hover:text-white"
          >
            Sign out
          </button>
          <div className="mt-3 text-xs text-slate-600">v0.1.0</div>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-6xl px-8 py-8">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
