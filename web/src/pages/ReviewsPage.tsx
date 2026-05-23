import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { MessageSquare, Loader2 } from 'lucide-react'

export default function ReviewsPage() {
  const [comment, setComment] = useState('')
  const [action, setAction] = useState<'approve' | 'request-changes' | null>(null)

  const queryClient = useQueryClient()

  const { data: packages } = useQuery({
    queryKey: ['packages'],
    queryFn: () => api.listPackages({ limit: 50 }),
  })

  const submitReview = useMutation({
    mutationFn: ({ org, namespace, name, version }: { org: string; namespace: string; name: string; version: string }) =>
      fetch(`/api/v1/reviews/${org}/${namespace}/${name}/versions/${version}/submit`, {
        method: 'POST',
      }).then((r) => {
        if (!r.ok) return r.json().then((b) => Promise.reject(new Error(b.error?.message || 'Submit failed')))
        return r.json()
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reviews'] })
    },
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Review Queue</h1>
        <p className="text-muted-foreground">
          Review submitted package versions before publishing.
        </p>
      </div>

      <div className="space-y-4">
        {packages?.items.map((pkg) => (
          <div key={`${pkg.org}/${pkg.namespace}/${pkg.name}`} className="border rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{pkg.displayName}</p>
                <p className="text-sm text-muted-foreground font-mono">
                  {pkg.org}/{pkg.namespace}/{pkg.name}
                </p>
              </div>
              <span
                className={`text-xs px-2 py-1 rounded-full ${
                  pkg.lifecycle === 'published'
                    ? 'bg-green-100 text-green-700'
                    : 'bg-yellow-100 text-yellow-700'
                }`}
              >
                {pkg.lifecycle}
              </span>
            </div>

            {pkg.lifecycle !== 'published' && (
              <div className="mt-3 flex gap-2">
                <button
                  onClick={() => submitReview.mutate({ org: pkg.org, namespace: pkg.namespace, name: pkg.name, version: pkg.stableVersion })}
                  disabled={submitReview.isPending}
                  className="px-3 py-1.5 bg-primary text-primary-foreground rounded text-sm font-medium hover:opacity-90 disabled:opacity-50"
                >
                  {submitReview.isPending ? (
                    <span className="flex items-center gap-1">
                      <Loader2 className="w-3 h-3 animate-spin" /> Submitting...
                    </span>
                  ) : (
                    'Submit for Review'
                  )}
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {action && (
        <div className="border rounded-lg p-4 space-y-3">
          <h3 className="font-medium">
            {action === 'approve' ? 'Approve Version' : 'Request Changes'}
          </h3>
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder="Add a comment..."
            rows={3}
            className="w-full px-3 py-2 border rounded-lg text-sm"
          />
          <div className="flex gap-2">
            <button
              onClick={() => setAction(null)}
              className="px-3 py-1.5 border rounded text-sm font-medium hover:bg-muted"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {!packages?.items.length && (
        <div className="text-center py-12 text-muted-foreground">
          <MessageSquare className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No packages to review.</p>
        </div>
      )}
    </div>
  )
}
