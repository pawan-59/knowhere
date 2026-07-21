// Thin fetch wrapper around the Central-Devtron backend API.
const BASE = import.meta.env.VITE_API_BASE || ''

// Subscribers notified when the server reports we are unauthenticated (401),
// so the app can bounce the user back to the login screen.
const unauthListeners = new Set()
export function onUnauthorized(fn) {
  unauthListeners.add(fn)
  return () => unauthListeners.delete(fn)
}

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include', // send/receive the httpOnly session cookie
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (res.status === 401 && !options._skipAuthRedirect) {
    unauthListeners.forEach((fn) => fn())
  }
  if (res.status === 204) return null
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const err = new Error(body.error || `Request failed (${res.status})`)
    err.status = res.status
    err.details = body.details
    throw err
  }
  return body
}

export const api = {
  health: () => request('/api/health'),

  // Auth
  login: (email, password) =>
    request('/api/auth/login', { method: 'POST', body: JSON.stringify({ email, password }), _skipAuthRedirect: true }),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  me: () => request('/api/auth/me', { _skipAuthRedirect: true }),

  // Zoho Desk
  zohoSummary: () => request('/api/zoho/summary'),
  zohoTickets: (params = '') => request(`/api/zoho/tickets${params}`),

  // Devtron
  devtronSummary: () => request('/api/devtron/summary'),
  devtronDeployments: () => request('/api/devtron/deployments'),

  // Licenses
  licenseSummary: () => request('/api/licenses/summary'),
  licenses: (status = '') => request(`/api/licenses${status ? `?status=${status}` : ''}`),
  saveLicense: (data) => request('/api/licenses', { method: 'POST', body: JSON.stringify(data) }),
  deleteLicense: (id) => request(`/api/licenses/${id}`, { method: 'DELETE' }),

  // Onboarding
  onboardingSummary: () => request('/api/onboarding/summary'),
  onboardings: (params = '') => request(`/api/onboarding${params}`),
  saveOnboarding: (data) => request('/api/onboarding', { method: 'POST', body: JSON.stringify(data) }),
  deleteOnboarding: (id) => request(`/api/onboarding/${id}`, { method: 'DELETE' }),
}
