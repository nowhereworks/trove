import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../lib/api'
import { Search } from 'lucide-react'

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [query, setQuery] = useState(searchParams.get('q') || '')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  useEffect(() => {
    if (debouncedQuery) {
      setSearchParams({ q: debouncedQuery })
    }
  }, [debouncedQuery])

  const { data, isLoading } = useQuery({
    queryKey: ['search', debouncedQuery],
    queryFn: () => api.search(debouncedQuery, { limit: 50 }),
    enabled: debouncedQuery.length > 0,
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Search Packages</h1>
        <p className="text-muted-foreground">
          Find packages by name, description, or labels.
        </p>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <input
          type="search"
          placeholder="Search packages..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full pl-10 pr-4 py-3 border rounded-lg bg-background text-base focus:outline-none focus:ring-2 focus:ring-ring"
          aria-label="Search packages"
          autoFocus
        />
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="p-4 border rounded-lg animate-pulse">
              <div className="h-4 bg-muted rounded w-1/3 mb-2" />
              <div className="h-3 bg-muted rounded w-2/3" />
            </div>
          ))}
        </div>
      )}

      {data && (
        <div>
          <p className="text-sm text-muted-foreground mb-4">
            {data.items.length} result{data.items.length !== 1 ? 's' : ''}
          </p>
          <div className="space-y-3">
            {data.items.map((pkg) => (
              <Link
                key={`${pkg.org}/${pkg.namespace}/${pkg.name}`}
                to={`/packages/${pkg.org}/${pkg.namespace}/${pkg.name}`}
                className="block p-4 border rounded-lg hover:border-primary/50 hover:shadow-sm transition-colors"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="font-medium">{pkg.displayName}</p>
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
              </Link>
            ))}
          </div>
        </div>
      )}

      {!isLoading && debouncedQuery && data?.items.length === 0 && (
        <div className="text-center py-12 text-muted-foreground">
          <p>No packages match "{debouncedQuery}"</p>
        </div>
      )}

      {!debouncedQuery && (
        <div className="text-center py-12 text-muted-foreground">
          <Search className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>Type to search published active packages.</p>
        </div>
      )}
    </div>
  )
}
