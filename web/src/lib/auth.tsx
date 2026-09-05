import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { get, post, TOKEN_KEY } from './api'

export interface SessionUser {
  id: number
  username: string
  nickname?: string
  email?: string
  avatar?: string
  is_admin?: boolean
  role_id?: number
  role_name?: string
  role_code?: string
  permissions?: string[]
  route_names?: string[]
  token: string
}

interface LoginInput {
  username: string
  password: string
  platform: string
  captcha?: string
  captcha_id?: string
}

interface AuthContextValue {
  user: SessionUser | null
  loading: boolean
  isAdmin: boolean
  hasPermission: (permission: string) => boolean
  login: (input: LoginInput) => Promise<void>
  register: (input: { username: string; email: string; password: string; confirm_password: string }) => Promise<void>
  completeOidc: (code: string) => Promise<void>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null)
  const [loading, setLoading] = useState(true)

  const clear = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem('wc-option:local:access_token')
    setUser(null)
  }, [])

  const refresh = useCallback(async () => {
    if (!localStorage.getItem(TOKEN_KEY)) {
      setLoading(false)
      return
    }
    try {
      const current = await get<SessionUser>('/user/current')
      setUser(current)
    } catch {
      clear()
    } finally {
      setLoading(false)
    }
  }, [clear])

  useEffect(() => {
    void refresh()
    const unauthorized = () => clear()
    window.addEventListener('desklink:unauthorized', unauthorized)
    return () => window.removeEventListener('desklink:unauthorized', unauthorized)
  }, [clear, refresh])

  const login = useCallback(async (input: LoginInput) => {
    const next = await post<SessionUser>('/login', input)
    localStorage.setItem(TOKEN_KEY, next.token)
    localStorage.setItem('wc-option:local:access_token', next.token)
    setUser(next)
  }, [])

  const register = useCallback(async (input: { username: string; email: string; password: string; confirm_password: string }) => {
    const next = await post<SessionUser>('/user/register', input)
    localStorage.setItem(TOKEN_KEY, next.token)
    localStorage.setItem('wc-option:local:access_token', next.token)
    setUser(next)
  }, [])

  const completeOidc = useCallback(async (code: string) => {
    const next = await get<SessionUser>('/oidc/auth-query', { code, uuid: '' })
    localStorage.removeItem('desklink_oidc_code')
    localStorage.setItem(TOKEN_KEY, next.token)
    localStorage.setItem('wc-option:local:access_token', next.token)
    setUser(next)
  }, [])

  const logout = useCallback(async () => {
    try {
      await post('/logout')
    } finally {
      clear()
    }
  }, [clear])

  const value = useMemo<AuthContextValue>(() => ({
    user,
    loading,
    isAdmin: Boolean(user?.is_admin || user?.route_names?.includes('*')),
    hasPermission: (permission: string) => Boolean(user?.is_admin || user?.route_names?.includes('*') || user?.permissions?.includes('*') || user?.permissions?.includes(permission)),
    login,
    register,
    completeOidc,
    logout,
    refresh,
  }), [user, loading, login, register, completeOidc, logout, refresh])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
