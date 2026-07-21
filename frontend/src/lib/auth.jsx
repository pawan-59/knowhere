import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import { api, onUnauthorized } from './api'

const AuthCtx = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  // On mount, ask the server who we are (validates the session cookie).
  const refresh = useCallback(async () => {
    try {
      const { user } = await api.me()
      setUser(user)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { refresh() }, [refresh])

  // If any request 401s, drop back to logged-out state.
  useEffect(() => onUnauthorized(() => setUser(null)), [])

  const login = useCallback(async (email, password) => {
    const { user } = await api.login(email, password)
    setUser(user)
    return user
  }, [])

  const logout = useCallback(async () => {
    try { await api.logout() } finally { setUser(null) }
  }, [])

  return (
    <AuthCtx.Provider value={{ user, loading, login, logout }}>
      {children}
    </AuthCtx.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthCtx)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
