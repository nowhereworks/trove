import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from './api'

interface UserInfo {
  id: string
  email: string
  displayName: string
  isDev: boolean
}

interface AuthContextValue {
  user: UserInfo | null
  isAuthenticated: boolean
  authMode: string
  loading: boolean
  login: (token?: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  isAuthenticated: false,
  authMode: 'dev',
  loading: true,
  login: async () => {},
  logout: async () => {},
})

export function useAuth() {
  return useContext(AuthContext)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [authMode, setAuthMode] = useState('dev')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.getConfig().catch(() => ({ org: '', allowCreateOrg: true, authMode: 'dev', cookieSecure: true })),
      api.getAuthMe().catch(() => ({ authenticated: false })),
    ]).then(([config, me]) => {
      setAuthMode((config as any).authMode ?? 'dev')
      if ((me as any).authenticated) {
        setUser((me as any).user)
      }
      setLoading(false)
    })
  }, [])

  const login = async (token?: string) => {
    if (authMode === 'dev' && token) {
      await api.loginDev(token)
      window.location.href = '/'
    } else if (authMode === 'local') {
      await api.loginLocal()
      window.location.href = '/'
    } else if (authMode === 'oidc') {
      window.location.href = '/auth/oidc/login'
    }
  }

  const logout = async () => {
    await api.logout()
    setUser(null)
    window.location.href = '/'
  }

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, authMode, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
