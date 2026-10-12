import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { AUTH_EXPIRED, PASSWORD_CHANGE_REQUIRED, api, post, type User } from './api'

interface AuthState {
  user: User | null
  loading: boolean
  login: (username: string, password: string) => Promise<User>
  logout: () => Promise<void>
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api<User>('/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))

    const onExpired = () => setUser(null)
    const onPwRequired = () => setUser((u) => (u ? { ...u, mustChangePassword: true } : u))
    window.addEventListener(AUTH_EXPIRED, onExpired)
    window.addEventListener(PASSWORD_CHANGE_REQUIRED, onPwRequired)
    return () => {
      window.removeEventListener(AUTH_EXPIRED, onExpired)
      window.removeEventListener(PASSWORD_CHANGE_REQUIRED, onPwRequired)
    }
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const u = await post<User>('/auth/login', { username, password })
    setUser(u)
    return u
  }, [])

  const logout = useCallback(async () => {
    await post('/auth/logout', {}).catch(() => {})
    setUser(null)
  }, [])

  const changePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    setUser(await post<User>('/auth/password', { currentPassword, newPassword }))
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, changePassword }}>{children}</AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}
