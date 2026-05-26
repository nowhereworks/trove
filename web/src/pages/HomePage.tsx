import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, type PackageSummary } from '../lib/api'
import { Search, Package, ArrowRight } from 'lucide-react'

export default function HomePage() {
  const [searchQuery, setSearchQuery] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['packages'],
    queryFn: () => api.listPackages({ limit: 50 }),
  })

  return (
    <div className="space-y-8">
      <section className="text-center space-y-4 py-8">
        <p className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
          Trove Registry
        </p>
        <h1 className="text-3xl font-bold sm:text-4xl">
          Agent artifacts, resolved exactly.
        </h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Browse packages, publish draft versions, and copy exact raw URLs for agent instructions.
        </p>
      </section>

      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-semibold">Packages</h2>
          <Link
            to="/search"
            className="ml-auto text-sm text-primary hover:underline inline-flex items-center gap-1"
          >
            Advanced search <ArrowRight className="w-3 h-3" />
          </Link>
        </div>

        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="search"
            placeholder="Quick search packages..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border rounded-lg bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            aria-label="Search packages"
          />
        </div>

        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="p-4 border rounded-lg animate-pulse">
                <div className="h-4 bg-muted rounded w-1/3 mb-2" />
                <div className="h-3 bg-muted rounded w-2/3" />
              </div>
            ))}
          </div>
        ) : (
          <PackageList
            items={data?.items || []}
            filter={searchQuery}
          />
        )}
      </section>
    </div>
  )
}

function PackageList({ items, filter }: { items: PackageSummary[]; filter: string }) {
  const filtered = filter
    ? items.filter(
        (p) =>
          p.name.toLowerCase().includes(filter.toLowerCase()) ||
          p.displayName.toLowerCase().includes(filter.toLowerCase()) ||
          p.description.toLowerCase().includes(filter.toLowerCase()),
      )
    : items

  if (filtered.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <Package className="w-12 h-12 mx-auto mb-4 opacity-50" />
        <p>No packages found.</p>
      </div>
    )
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {filtered.map((pkg) => (
        <Link
          key={`${pkg.org}/${pkg.namespace}/${pkg.name}`}
          to={`/packages/${pkg.org}/${pkg.namespace}/${pkg.name}`}
          className="block p-4 border rounded-lg hover:border-primary/50 hover:shadow-sm transition-colors"
        >
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="font-medium truncate">{pkg.displayName}</p>
              <p className="text-sm text-muted-foreground font-mono">
                {pkg.org}/{pkg.namespace}/{pkg.name}
              </p>
            </div>
            <span
              className={`shrink-0 text-xs px-2 py-0.5 rounded-full ${
                pkg.visibility === 'public'
                  ? 'bg-green-100 text-green-700'
                  : pkg.visibility === 'internal'
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-700'
              }`}
            >
              {pkg.visibility}
            </span>
          </div>
          <p className="mt-2 text-sm text-muted-foreground line-clamp-2">
            {pkg.description}
          </p>
          {pkg.latestVersion && (
            <p className="mt-2 text-xs font-mono text-muted-foreground">
              stable: {pkg.latestVersion}
            </p>
          )}
        </Link>
      ))}
    </div>
  )
}
