import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AlertCircle, CheckCircle, Loader2 } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../lib/auth'

export default function CreateOrgPage() {
  const { isAuthenticated, loading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate('/login')
    }
  }, [loading, isAuthenticated, navigate])

  if (loading || !isAuthenticated) {
    return <div className="max-w-md mx-auto text-center py-12 text-muted-foreground">Redirecting...</div>
  }

  const [slug, setSlug] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [createdSlug, setCreatedSlug] = useState('')

  const { data: config } = useQuery({ queryKey: ['config'], queryFn: api.getConfig })

  const createOrg = useMutation({
    mutationFn: () => api.createOrg({ slug, displayName: displayName || slug, visibility: 'private' }),
    onSuccess: (org) => {
      setCreatedSlug(org.slug)
      setSlug('')
      setDisplayName('')
    },
  })

  const allowCreateOrg = config?.allowCreateOrg ?? true

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Create Org</h1>
        <p className="text-muted-foreground">
          Add a top-level organization for namespaces and packages.
        </p>
      </div>

      {!allowCreateOrg && (
        <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-amber-800">
          <AlertCircle className="mt-0.5 h-4 w-4" />
          <div className="text-sm">
            Org creation is disabled by this server. Configure <code>TROVE_ORG</code> at startup or enable <code>TROVE_ALLOW_CREATE_ORG</code>.
          </div>
        </div>
      )}

      {createdSlug && (
        <div className="flex items-center gap-2 rounded-lg bg-green-50 p-3 text-green-700">
          <CheckCircle className="h-4 w-4" />
          <span className="text-sm">Created org {createdSlug}. You can now create namespaces and packages under it.</span>
        </div>
      )}

      {createOrg.error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 p-3 text-red-700">
          <AlertCircle className="h-4 w-4" />
          <span className="text-sm">{createOrg.error.message}</span>
        </div>
      )}

      <div className="space-y-4 rounded-lg border p-6">
        <div>
          <label className="mb-1 block text-sm font-medium">Slug</label>
          <input
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            className="w-full rounded-lg border px-3 py-2 font-mono text-sm"
            placeholder="sample-org"
            disabled={!allowCreateOrg || createOrg.isPending}
          />
          <p className="mt-1 text-xs text-muted-foreground">Lowercase letters, numbers, and hyphens only.</p>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">Display Name</label>
          <input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="w-full rounded-lg border px-3 py-2 text-sm"
            placeholder="Sample Org"
            disabled={!allowCreateOrg || createOrg.isPending}
          />
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => createOrg.mutate()}
            disabled={!allowCreateOrg || !slug || createOrg.isPending}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90 disabled:opacity-50"
          >
            {createOrg.isPending ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" /> Creating...
              </span>
            ) : (
              'Create Org'
            )}
          </button>
          <Link to="/upload" className="text-sm text-muted-foreground hover:text-foreground">
            Publish a package
          </Link>
        </div>
      </div>
    </div>
  )
}
