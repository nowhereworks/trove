import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useQuery as useTQuery } from '@tanstack/react-query'
import { Users, TrendingUp } from 'lucide-react'

export default function AdoptionPage() {
  const { data: packages } = useTQuery({
    queryKey: ['packages'],
    queryFn: () => api.listPackages({ limit: 100 }),
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Adoption Dashboard</h1>
        <p className="text-muted-foreground">
          Aggregate adoption counts across all packages.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {packages?.items.map((pkg) => (
          <AdoptionCard key={`${pkg.org}/${pkg.namespace}/${pkg.name}`} pkg={pkg} />
        ))}
      </div>

      {!packages?.items.length && (
        <div className="text-center py-12 text-muted-foreground">
          <Users className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No packages to display.</p>
        </div>
      )}
    </div>
  )
}

function AdoptionCard({ pkg }: { pkg: { org: string; namespace: string; name: string; displayName: string; stableVersion: string } }) {
  const { data: adoption } = useQuery({
    queryKey: ['adoption', pkg.org, pkg.namespace, pkg.name],
    queryFn: () => api.getAdoption(pkg.org, pkg.namespace, pkg.name),
  })

  return (
    <Link
      to={`/packages/${pkg.org}/${pkg.namespace}/${pkg.name}`}
      className="block p-4 border rounded-lg hover:border-primary/50 hover:shadow-sm transition-colors"
    >
      <div className="flex items-start justify-between">
        <div>
          <p className="font-medium">{pkg.displayName}</p>
          <p className="text-sm text-muted-foreground font-mono">
            {pkg.org}/{pkg.namespace}/{pkg.name}
          </p>
        </div>
        <TrendingUp className="w-4 h-4 text-muted-foreground" />
      </div>
      <div className="mt-3 flex items-baseline gap-2">
        <span className="text-2xl font-bold">{adoption?.projectCount || 0}</span>
        <span className="text-sm text-muted-foreground">projects</span>
      </div>
      {adoption?.byVersion && adoption.byVersion.length > 0 && (
        <div className="mt-2 space-y-1">
          {adoption.byVersion.slice(0, 3).map((v) => (
            <div key={v.version} className="flex justify-between text-xs">
              <span className="font-mono">{v.version}</span>
              <span className="text-muted-foreground">{v.count}</span>
            </div>
          ))}
        </div>
      )}
    </Link>
  )
}
