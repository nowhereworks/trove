import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, type MaintainerInfo } from '../lib/api'
import { Copy, Package, Clock, Shield, Users } from 'lucide-react'

function roleBadgeClass(role: string) {
  return role === 'owner'
    ? 'bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
    : 'bg-blue-100 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
}

export default function PackagePage() {
  const params = useParams<{ org: string; namespace: string; name: string }>()
  const org = params.org!
  const namespace = params.namespace!
  const name = params.name!
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null)

  const { data: pkg, isLoading: pkgLoading } = useQuery({
    queryKey: ['package', org, namespace, name],
    queryFn: () => api.getPackage(org, namespace, name),
  })

  const { data: maintainers } = useQuery({
    queryKey: ['maintainers', org, namespace, name],
    queryFn: () => api.getMaintainers(org, namespace, name),
  })

  const { data: adoption } = useQuery({
    queryKey: ['adoption', org, namespace, name],
    queryFn: () => api.getAdoption(org, namespace, name),
  })

  const { data: manifest } = useQuery({
    queryKey: ['manifest', org, namespace, name, selectedVersion || pkg?.latestVersion],
    queryFn: () =>
      api.getManifest(org, namespace, name, selectedVersion || pkg?.latestVersion || ''),
    enabled: !!(selectedVersion || pkg?.latestVersion),
  })

  if (pkgLoading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 bg-muted rounded w-1/3" />
        <div className="h-4 bg-muted rounded w-2/3" />
        <div className="grid grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-24 bg-muted rounded" />
          ))}
        </div>
      </div>
    )
  }

  if (!pkg) {
    return (
      <div className="text-center py-12">
        <Package className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
        <h2 className="text-xl font-semibold">Package not found</h2>
        <Link to="/" className="text-primary hover:underline mt-2 inline-block">
          Back to packages
        </Link>
      </div>
    )
  }

  const versions = pkg.versions || []
  const currentVersion = selectedVersion || pkg.latestVersion || pkg.latestVersion

  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-mono text-muted-foreground">
              {pkg.org}/{pkg.namespace}/{pkg.name}
            </p>
            <h1 className="text-2xl font-bold">{pkg.displayName}</h1>
            <p className="text-muted-foreground mt-1">{pkg.description}</p>
          </div>
          <div className="flex gap-2">
            <span
              className={`text-xs px-2 py-1 rounded-full ${
                pkg.visibility === 'public'
                  ? 'bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-300'
                  : pkg.visibility === 'internal'
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
                    : 'bg-gray-100 text-gray-700 dark:bg-slate-800 dark:text-slate-300'
              }`}
            >
              {pkg.visibility}
            </span>
            <span
              className={`text-xs px-2 py-1 rounded-full ${
                pkg.lifecycle === 'published'
                  ? 'bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-300'
                  : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-950/40 dark:text-yellow-300'
              }`}
            >
              {pkg.lifecycle}
            </span>
          </div>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard icon={<Clock className="w-4 h-4" />} label="Latest" value={pkg.latestVersion || '—'} />
        <StatCard icon={<Package className="w-4 h-4" />} label="Versions" value={String(versions.length)} />
        <StatCard icon={<Shield className="w-4 h-4" />} label="Visibility" value={pkg.visibility} />
        <StatCard icon={<Users className="w-4 h-4" />} label="Projects" value={String(adoption?.projectCount || 0)} />
      </div>

      {maintainers && maintainers.length > 0 && (
        <div>
          <h2 className="text-lg font-semibold mb-2">Maintainers</h2>
          <div className="flex flex-wrap gap-2">
            {maintainers.map((m: MaintainerInfo) => (
              <span key={m.userId} className="inline-flex items-center gap-1.5 text-sm px-3 py-1 bg-muted rounded-full">
                {m.displayName}
                <span className={`text-xs px-1.5 py-0.5 rounded ${roleBadgeClass(m.role)}`}>
                  {m.role}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <section>
            <h2 className="text-lg font-semibold mb-3">Versions</h2>
            <div className="border rounded-lg divide-y">
              {versions.map((v) => (
                <button
                  key={v.version}
                  onClick={() => setSelectedVersion(v.version)}
                  className={`w-full flex items-center justify-between p-3 text-left hover:bg-muted/50 transition-colors ${
                    currentVersion === v.version ? 'bg-muted' : ''
                  }`}
                >
                  <div>
                    <span className="font-mono text-sm font-medium">{v.version}</span>
                    <span
                      className={`ml-2 text-xs px-1.5 py-0.5 rounded ${
                        v.lifecycle === 'published'
                          ? 'bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-300'
                          : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-950/40 dark:text-yellow-300'
                      }`}
                    >
                      {v.lifecycle}
                    </span>
                  </div>
                  <span className="text-xs text-muted-foreground font-mono">
                    {v.digest?.slice(0, 12)}
                  </span>
                </button>
              ))}
            </div>
          </section>

          {manifest && (
            <section>
              <h2 className="text-lg font-semibold mb-3">
                Manifest — {currentVersion}
              </h2>
              <div className="border rounded-lg p-4 bg-muted/30">
                <pre className="text-sm overflow-x-auto whitespace-pre-wrap">
                  {JSON.stringify(manifest, null, 2)}
                </pre>
              </div>
            </section>
          )}
        </div>

        <div className="space-y-6">
          <section>
            <h2 className="text-lg font-semibold mb-3">Install</h2>
            <div className="border rounded-lg p-4 space-y-3">
              <div>
                <p className="text-xs text-muted-foreground mb-1">Resolve</p>
                <code className="text-sm bg-muted px-2 py-1 rounded block font-mono">
                  trove resolve {pkg.org}/{pkg.namespace}/{pkg.name}@stable
                </code>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">Install</p>
                <code className="text-sm bg-muted px-2 py-1 rounded block font-mono">
                  trove install {pkg.org}/{pkg.namespace}/{pkg.name}@stable
                </code>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">Raw URL</p>
                <div className="flex items-center gap-2">
                  <code className="text-sm bg-muted px-2 py-1 rounded block font-mono truncate flex-1">
                    /raw/{pkg.org}/{pkg.namespace}/{pkg.name}/{currentVersion}/AGENTS.md
                  </code>
                  <button
                    onClick={() =>
                      navigator.clipboard.writeText(
                        `/raw/${pkg.org}/${pkg.namespace}/${pkg.name}/${currentVersion}/AGENTS.md`,
                      )
                    }
                    className="p-1 hover:bg-muted rounded"
                    aria-label="Copy raw URL"
                  >
                    <Copy className="w-3 h-3" />
                  </button>
                </div>
              </div>
            </div>
          </section>

          {adoption && adoption.projectCount > 0 && (
            <section>
              <h2 className="text-lg font-semibold mb-3">Adoption</h2>
              <div className="border rounded-lg p-4">
                <p className="text-2xl font-bold">{adoption.projectCount}</p>
                <p className="text-sm text-muted-foreground">projects using this package</p>
                {adoption.versions && adoption.versions.length > 0 && (
                  <div className="mt-3 space-y-1">
                    {adoption.versions.map((v) => (
                      <div key={v.version} className="flex justify-between text-sm">
                        <span className="font-mono">{v.version}</span>
                        <span className="text-muted-foreground">{v.installCount}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  )
}

function StatCard({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="border rounded-lg p-4">
      <div className="flex items-center gap-2 text-muted-foreground mb-1">
        {icon}
        <span className="text-sm">{label}</span>
      </div>
      <p className="text-xl font-semibold">{value}</p>
    </div>
  )
}
