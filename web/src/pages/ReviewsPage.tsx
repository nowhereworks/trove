import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, CheckCircle, Loader2, MessageSquare } from 'lucide-react'
import { api, type ReviewQueueItem } from '../lib/api'
import { useAuth } from '../lib/auth'

type ReviewAction = 'approve' | 'request-changes'

export default function ReviewsPage() {
  const { isAuthenticated, loading } = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const [activeItem, setActiveItem] = useState<ReviewQueueItem | null>(null)
  const [action, setAction] = useState<ReviewAction | null>(null)
  const [comment, setComment] = useState('')
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate('/login')
    }
  }, [loading, isAuthenticated, navigate])

  const { data: queue, isLoading } = useQuery({
    queryKey: ['reviewQueue'],
    queryFn: () => api.listReviewQueue({ limit: 100 }),
    enabled: isAuthenticated,
  })

  const resetAction = () => {
    setActiveItem(null)
    setAction(null)
    setComment('')
  }

  const refreshReviewData = (item: ReviewQueueItem) => {
    queryClient.invalidateQueries({ queryKey: ['reviewQueue'] })
    queryClient.invalidateQueries({ queryKey: ['packages'] })
    queryClient.invalidateQueries({ queryKey: ['package', item.org, item.namespace, item.package] })
  }

  const approve = useMutation({
    mutationFn: (item: ReviewQueueItem) => api.approveReview(item.reviewId, item.packageVersionId, comment || undefined),
    onSuccess: (_, item) => {
      setMessage(`${item.org}/${item.namespace}/${item.package}@${item.version} approved.`)
      setError(null)
      resetAction()
      refreshReviewData(item)
    },
    onError: (e) => {
      setMessage(null)
      setError(errorMessage(e))
    },
  })

  const requestChanges = useMutation({
    mutationFn: (item: ReviewQueueItem) => api.requestReviewChanges(item.reviewId, comment),
    onSuccess: (_, item) => {
      setMessage(`Changes requested for ${item.org}/${item.namespace}/${item.package}@${item.version}.`)
      setError(null)
      resetAction()
      refreshReviewData(item)
    },
    onError: (e) => {
      setMessage(null)
      setError(errorMessage(e))
    },
  })

  const publish = useMutation({
    mutationFn: (item: ReviewQueueItem) => api.publishVersion(item.org, item.namespace, item.package, item.version),
    onSuccess: (_, item) => {
      setMessage(`${item.org}/${item.namespace}/${item.package}@${item.version} published.`)
      setError(null)
      refreshReviewData(item)
    },
    onError: (e) => {
      setMessage(null)
      setError(errorMessage(e))
    },
  })

  if (loading || !isAuthenticated) {
    return <div className="max-w-4xl mx-auto text-center py-12 text-muted-foreground">Redirecting...</div>
  }

  const packageFilter = searchParams.get('package')
  const versionFilter = searchParams.get('version')
  const items = (queue?.items || []).filter((item) => {
    const ref = `${item.org}/${item.namespace}/${item.package}`
    return (!packageFilter || packageFilter === ref) && (!versionFilter || versionFilter === item.version)
  })
  const pendingAction = approve.isPending || requestChanges.isPending

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Review Queue</h1>
        <p className="text-muted-foreground">
          Approve submitted package versions, request changes, then publish approved versions.
        </p>
      </div>

      {message ? (
        <div className="flex items-center gap-2 p-3 bg-green-50 text-green-700 rounded-lg">
          <CheckCircle className="w-4 h-4" />
          <span className="text-sm">{message}</span>
        </div>
      ) : null}

      {error ? (
        <div className="flex items-center gap-2 p-3 bg-red-50 text-red-700 rounded-lg">
          <AlertCircle className="w-4 h-4" />
          <span className="text-sm">{error}</span>
        </div>
      ) : null}

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-28 border rounded-lg animate-pulse bg-muted/30" />
          ))}
        </div>
      ) : items.length > 0 ? (
        <div className="space-y-4">
          {items.map((item) => (
            <ReviewCard
              key={item.reviewId}
              item={item}
              publishing={publish.isPending}
              onApprove={() => {
                setActiveItem(item)
                setAction('approve')
                setComment('')
                setError(null)
              }}
              onRequestChanges={() => {
                setActiveItem(item)
                setAction('request-changes')
                setComment('')
                setError(null)
              }}
              onPublish={() => publish.mutate(item)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-muted-foreground">
          <MessageSquare className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No versions awaiting review.</p>
        </div>
      )}

      {activeItem && action ? (
        <div className="border rounded-lg p-4 space-y-3">
          <div>
            <h3 className="font-medium">
              {action === 'approve' ? 'Approve Version' : 'Request Changes'}
            </h3>
            <p className="text-sm text-muted-foreground font-mono">
              {activeItem.org}/{activeItem.namespace}/{activeItem.package}@{activeItem.version}
            </p>
          </div>
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder={action === 'approve' ? 'Optional approval note...' : 'Describe the required changes...'}
            rows={3}
            className="w-full px-3 py-2 border rounded-lg text-sm"
          />
          <div className="flex gap-2">
            <button
              onClick={resetAction}
              className="px-3 py-1.5 border rounded text-sm font-medium hover:bg-muted"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                if (action === 'approve') {
                  approve.mutate(activeItem)
                  return
                }
                if (!comment.trim()) {
                  setError('A comment is required when requesting changes.')
                  return
                }
                requestChanges.mutate(activeItem)
              }}
              disabled={pendingAction}
              className="px-3 py-1.5 bg-primary text-primary-foreground rounded text-sm font-medium hover:opacity-90 disabled:opacity-50"
            >
              {pendingAction ? 'Saving...' : action === 'approve' ? 'Approve' : 'Request Changes'}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function ReviewCard({
  item,
  publishing,
  onApprove,
  onRequestChanges,
  onPublish,
}: {
  item: ReviewQueueItem
  publishing: boolean
  onApprove: () => void
  onRequestChanges: () => void
  onPublish: () => void
}) {
  const packageRef = `${item.org}/${item.namespace}/${item.package}`

  return (
    <div className="border rounded-lg p-4 space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="font-medium">{item.displayName}</p>
          <p className="text-sm text-muted-foreground font-mono">
            {packageRef}@{item.version}
          </p>
          {item.description ? <p className="mt-1 text-sm text-muted-foreground">{item.description}</p> : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge tone="neutral">{item.visibility}</Badge>
          <Badge tone="warning">{item.reviewStatus}</Badge>
          {item.hasEnoughApprovals ? <Badge tone="success">ready to publish</Badge> : null}
        </div>
      </div>

      <div className="grid gap-3 text-sm sm:grid-cols-3">
        <div className="rounded-lg bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground">Approvals</p>
          <p className="font-medium">{item.currentApprovals}/{item.requiredApprovals}</p>
        </div>
        <div className="rounded-lg bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground">Lifecycle</p>
          <p className="font-medium">{item.lifecycle}</p>
        </div>
        <div className="rounded-lg bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground">Updated</p>
          <p className="font-medium">{formatDate(item.updatedAt)}</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <button
          onClick={onApprove}
          className="px-3 py-1.5 bg-primary text-primary-foreground rounded text-sm font-medium hover:opacity-90"
        >
          Approve
        </button>
        <button
          onClick={onRequestChanges}
          className="px-3 py-1.5 border rounded text-sm font-medium hover:bg-muted"
        >
          Request changes
        </button>
        <button
          onClick={onPublish}
          disabled={!item.hasEnoughApprovals || publishing}
          className="px-3 py-1.5 border rounded text-sm font-medium hover:bg-muted disabled:opacity-50"
        >
          {publishing ? <span className="inline-flex items-center gap-1"><Loader2 className="w-3 h-3 animate-spin" /> Publishing</span> : 'Publish'}
        </button>
        <Link
          to={`/packages/${item.org}/${item.namespace}/${item.package}`}
          className="px-3 py-1.5 text-sm text-primary hover:underline"
        >
          View package
        </Link>
      </div>
    </div>
  )
}

function Badge({ tone, children }: { tone: 'neutral' | 'warning' | 'success'; children: string }) {
  const className = tone === 'success'
    ? 'bg-green-100 text-green-700'
    : tone === 'warning'
      ? 'bg-yellow-100 text-yellow-700'
      : 'bg-gray-100 text-gray-700'

  return <span className={`text-xs px-2 py-1 rounded-full ${className}`}>{children}</span>
}

function formatDate(value: string) {
  if (!value) return 'unknown'
  return new Date(value).toLocaleString()
}

function errorMessage(error: unknown) {
  const message = error instanceof Error ? error.message : 'Request failed'
  if (message.includes('Self-approval is not allowed')) {
    return 'Self-approval is disabled. Set TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true for local browser workflows.'
  }
  return message
}
