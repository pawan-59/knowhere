import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import './index.css'
import App from './App.jsx'
import Login from './pages/Login.jsx'
import Overview from './pages/Overview.jsx'
import Zoho from './pages/Zoho.jsx'
import Devtron from './pages/Devtron.jsx'
import Onboarding from './pages/Onboarding.jsx'
import License from './pages/License.jsx'
import { AuthProvider, useAuth } from './lib/auth'

// Gate renders the login screen until a valid session exists.
function Gate() {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-900">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-600 border-t-emerald-400" />
      </div>
    )
  }
  if (!user) return <Login />

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />}>
          <Route index element={<Overview />} />
          <Route path="zoho" element={<Zoho />} />
          <Route path="devtron" element={<Devtron />} />
          <Route path="onboarding" element={<Onboarding />} />
          <Route path="licenses" element={<License />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <AuthProvider>
      <Gate />
    </AuthProvider>
  </StrictMode>,
)
