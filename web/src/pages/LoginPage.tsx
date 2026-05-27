import { useState } from 'react'
import { useAuth } from '../lib/auth'
import { Link } from 'react-router-dom'
import { LogIn, User, ArrowLeft } from 'lucide-react'

export default function LoginPage() {
  const { user, isAuthenticated, authMode, login, loading } = useAuth()
  const [token, setToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (loading) {
    return (
      <div className="max-w-md mx-auto text-center py-12">
        <div className="animate-pulse text-muted-foreground">Loading...</div>
      </div>
    )
  }

  if (isAuthenticated && user) {
    return (
      <div className="max-w-md mx-auto space-y-6">
        <div className="border rounded-lg p-6 space-y-4 text-center">
          <div className="w-12 h-12 mx-auto rounded-full bg-primary/10 flex items-center justify-center">
            <User className="w-6 h-6 text-primary" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">Signed in as</h2>
            <p className="text-muted-foreground">{user.displayName}</p>
            <p className="text-sm text-muted-foreground">{user.email}</p>
          </div>
          <div className="flex gap-3 justify-center">
            <Link to="/" className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90">
              Go to Packages
            </Link>
          </div>
        </div>
      </div>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await login(token)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-md mx-auto space-y-6">
      <div className="border rounded-lg p-6 space-y-4">
        <h1 className="text-xl font-semibold flex items-center gap-2">
          <LogIn className="w-5 h-5" />
          Sign in to Trove
        </h1>

        {error && (
          <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm dark:bg-red-950/40 dark:text-red-300">{error}</div>
        )}

        {authMode === 'dev' && (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="text-sm font-medium mb-1 block">Dev Token</label>
              <input
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm font-mono"
                placeholder="dev-token-local-only"
                autoFocus
              />
            </div>
            <button
              type="submit"
              disabled={!token || submitting}
              className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 disabled:opacity-50"
            >
              {submitting ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        )}

        {authMode === 'local' && (
          <button
            onClick={() => login()}
            disabled={submitting}
            className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 disabled:opacity-50"
          >
            {submitting ? 'Signing in...' : 'Continue as Local User'}
          </button>
        )}

        {authMode === 'oidc' && (
          <button
            onClick={() => login()}
            className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90"
          >
            Login with SSO
          </button>
        )}
      </div>

      <Link to="/" className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="w-4 h-4" />
        Back to packages
      </Link>
    </div>
  )
}
