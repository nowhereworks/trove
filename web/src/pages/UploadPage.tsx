import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Upload, CheckCircle, AlertCircle, Loader2 } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../lib/auth'

type Step = 'create' | 'upload' | 'review' | 'publish' | 'done'

export default function UploadPage() {
  const { isAuthenticated, loading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      navigate('/login')
    }
  }, [loading, isAuthenticated, navigate])

  if (loading || !isAuthenticated) {
    return <div className="max-w-2xl mx-auto text-center py-12 text-muted-foreground">Redirecting...</div>
  }

  const [step, setStep] = useState<Step>('create')
  const [org, setOrg] = useState('')
  const [namespace, setNamespace] = useState('')
  const [name, setName] = useState('')
  const [version, setVersion] = useState('')
  const [visibility, setVisibility] = useState('public')
  const [manifestContent, setManifestContent] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const queryClient = useQueryClient()
  const { data: config } = useQuery({ queryKey: ['config'], queryFn: api.getConfig })

  useEffect(() => {
    if (!org && config?.org) {
      setOrg(config.org)
    }
  }, [config?.org, org])

  const createDraft = useMutation({
    mutationFn: () =>
      fetch('/api/v1/packages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ org, namespace, name, displayName: name, description: '', visibility }),
      }).then((r) => {
        if (!r.ok) throw new Error('Failed to create package')
        return r.json()
      }).then(() =>
        fetch(`/api/v1/packages/${org}/${namespace}/${name}/versions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ version, visibility }),
        })
      ).then((r) => {
        if (!r.ok) throw new Error('Failed to create draft version')
        return r.json()
      }),
    onSuccess: () => setStep('upload'),
    onError: (e) => setError(e.message),
  })

  const uploadManifest = useMutation({
    mutationFn: () =>
      fetch(`/api/v1/packages/${org}/${namespace}/${name}/versions/${version}/artifacts/Trovefile`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/yaml' },
        body: manifestContent,
      }).then((r) => {
        if (!r.ok) return r.json().then((b) => Promise.reject(new Error(b.error?.message || 'Upload failed')))
        return r.json()
      }),
    onSuccess: () => setStep('review'),
    onError: (e) => setError(e.message),
  })

  const publish = useMutation({
    mutationFn: () =>
      fetch(`/api/v1/packages/${org}/${namespace}/${name}/versions/${version}/publish`, {
        method: 'POST',
      }).then((r) => {
        if (!r.ok) return r.json().then((b) => Promise.reject(new Error(b.error?.message || 'Publish failed')))
        return r.json()
      }),
    onSuccess: () => {
      setStep('done')
      setSuccess(`Version ${version} published successfully!`)
      queryClient.invalidateQueries({ queryKey: ['packages'] })
    },
    onError: (e) => setError(e.message),
  })

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setError(null)
    const contentType = file.name.endsWith('.zip') ? 'application/zip' : 'application/gzip'

    try {
      const res = await fetch(
        `/api/v1/packages/${org}/${namespace}/${name}/versions/${version}/archive`,
        {
          method: 'POST',
          headers: { 'Content-Type': contentType },
          body: file,
        },
      )
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.error?.message || 'Archive upload failed')
      }
      setStep('review')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed')
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Publish Package</h1>
        <p className="text-muted-foreground">
          Create a draft version, upload artifacts, and publish.
        </p>
      </div>

      <div className="flex items-center gap-2 text-sm">
        <StepIndicator current={step} step="create" label="Create" />
        <div className="w-8 h-px bg-border" />
        <StepIndicator current={step} step="upload" label="Upload" />
        <div className="w-8 h-px bg-border" />
        <StepIndicator current={step} step="review" label="Review" />
        <div className="w-8 h-px bg-border" />
        <StepIndicator current={step} step="publish" label="Publish" />
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-red-50 text-red-700 rounded-lg">
          <AlertCircle className="w-4 h-4" />
          <span className="text-sm">{error}</span>
        </div>
      )}

      {success && (
        <div className="flex items-center gap-2 p-3 bg-green-50 text-green-700 rounded-lg">
          <CheckCircle className="w-4 h-4" />
          <span className="text-sm">{success}</span>
        </div>
      )}

      {step === 'create' && (
        <div className="border rounded-lg p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Draft Version</h2>
          <div className="grid gap-4 sm:grid-cols-3">
            <div>
              <label className="text-sm font-medium mb-1 block">Org</label>
              <input
                value={org}
                onChange={(e) => setOrg(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm"
                placeholder="companyx"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">Namespace</label>
              <input
                value={namespace}
                onChange={(e) => setNamespace(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm"
                placeholder="platform"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">Package</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm"
                placeholder="agent-backend"
              />
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="text-sm font-medium mb-1 block">Version</label>
              <input
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm font-mono"
                placeholder="1.0.0"
                pattern="[0-9]+\.[0-9]+\.[0-9]+"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">Visibility</label>
              <select
                value={visibility}
                onChange={(e) => setVisibility(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg text-sm"
              >
                <option value="public">Public</option>
                <option value="internal">Internal</option>
                <option value="private">Private</option>
              </select>
            </div>
          </div>
          <button
            onClick={() => createDraft.mutate()}
            disabled={!org || !namespace || !name || !version || createDraft.isPending}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 disabled:opacity-50"
          >
            {createDraft.isPending ? (
              <span className="flex items-center gap-2">
                <Loader2 className="w-4 h-4 animate-spin" /> Creating...
              </span>
            ) : (
              'Create Draft'
            )}
          </button>
        </div>
      )}

      {step === 'upload' && (
        <div className="border rounded-lg p-6 space-y-6">
          <h2 className="text-lg font-semibold">Upload Artifacts</h2>

          <div>
            <label className="text-sm font-medium mb-1 block">Upload Archive (.tar.gz or .zip)</label>
            <div className="border-2 border-dashed rounded-lg p-8 text-center">
              <Upload className="w-8 h-8 mx-auto mb-2 text-muted-foreground" />
              <p className="text-sm text-muted-foreground mb-2">
                Upload a package archive containing Trovefile and artifacts
              </p>
              <input
                type="file"
                accept=".tar.gz,.zip,.tgz"
                onChange={handleFileUpload}
                className="hidden"
                id="archive-upload"
              />
              <label
                htmlFor="archive-upload"
                className="inline-block px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 cursor-pointer"
              >
                Select Archive
              </label>
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">Or paste manifest</span>
            </div>
          </div>

          <div>
            <label className="text-sm font-medium mb-1 block">Trovefile Content</label>
            <textarea
              value={manifestContent}
              onChange={(e) => setManifestContent(e.target.value)}
              rows={12}
              className="w-full px-3 py-2 border rounded-lg text-sm font-mono"
              placeholder="apiVersion: trove.io/v1..."
            />
          </div>

          <button
            onClick={() => uploadManifest.mutate()}
            disabled={!manifestContent || uploadManifest.isPending}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 disabled:opacity-50"
          >
            {uploadManifest.isPending ? (
              <span className="flex items-center gap-2">
                <Loader2 className="w-4 h-4 animate-spin" /> Uploading...
              </span>
            ) : (
              'Upload Manifest'
            )}
          </button>
        </div>
      )}

      {step === 'review' && (
        <div className="border rounded-lg p-6 space-y-4">
          <h2 className="text-lg font-semibold">Review & Publish</h2>
          <div className="bg-muted/30 rounded-lg p-4 space-y-2">
            <p className="text-sm">
              <span className="font-medium">Package:</span> {org}/{namespace}/{name}
            </p>
            <p className="text-sm">
              <span className="font-medium">Version:</span> {version}
            </p>
            <p className="text-sm">
              <span className="font-medium">Visibility:</span> {visibility}
            </p>
          </div>
          <p className="text-sm text-muted-foreground">
            Review the package details before publishing. Published versions are immutable.
          </p>
          <div className="flex gap-3">
            <button
              onClick={() => setStep('upload')}
              className="px-4 py-2 border rounded-lg text-sm font-medium hover:bg-muted"
            >
              Back
            </button>
            <button
              onClick={() => publish.mutate()}
              disabled={publish.isPending}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 disabled:opacity-50"
            >
              {publish.isPending ? (
                <span className="flex items-center gap-2">
                  <Loader2 className="w-4 h-4 animate-spin" /> Publishing...
                </span>
              ) : (
                'Publish'
              )}
            </button>
          </div>
        </div>
      )}

      {step === 'done' && (
        <div className="border rounded-lg p-6 text-center space-y-4">
          <CheckCircle className="w-12 h-12 mx-auto text-green-600" />
          <h2 className="text-lg font-semibold">Published Successfully</h2>
          <p className="text-muted-foreground">{success}</p>
          <a
            href={`/packages/${org}/${namespace}/${name}`}
            className="inline-block px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90"
          >
            View Package
          </a>
        </div>
      )}
    </div>
  )
}

function StepIndicator({ current, step, label }: { current: Step; step: Step; label: string }) {
  const steps: Step[] = ['create', 'upload', 'review', 'publish', 'done']
  const currentIndex = steps.indexOf(current)
  const stepIndex = steps.indexOf(step)
  const isActive = stepIndex === currentIndex
  const isComplete = stepIndex < currentIndex

  return (
    <div className={`flex items-center gap-1 ${isActive ? 'text-primary font-medium' : isComplete ? 'text-green-600' : 'text-muted-foreground'}`}>
      {isComplete ? <CheckCircle className="w-4 h-4" /> : <span className="w-4 h-4 text-center">{stepIndex + 1}</span>}
      <span className="text-xs">{label}</span>
    </div>
  )
}
