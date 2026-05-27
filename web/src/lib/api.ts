export interface PackageSummary {
  org: string
  namespace: string
  name: string
  displayName: string
  description: string
  latestVersion: string
  visibility: string
  lifecycle: string
}

export interface MaintainerInfo {
  userId: string
  displayName: string
  email: string
  role: string
}

export interface PackageDetail {
  org: string
  namespace: string
  name: string
  displayName: string
  description: string
  visibility: string
  lifecycle: string
  latestVersion: string
  labels: string[]
  versions: { version: string; lifecycle: string; digest: string; publishedAt: string }[]
}

export interface ResolveResult {
  org: string
  namespace: string
  package: string
  selector: string
  resolvedVersion: string
  digest: string
  manifestUrl: string
  archiveUrl: string
}

export interface SearchResult {
  items: PackageSummary[]
  nextCursor: string | null
}

export interface ListPackagesResult {
  items: PackageSummary[]
  nextCursor: string | null
}

export interface AdoptionResult {
  projectCount: number
  versionCount: number
  byVersion: { version: string; count: number }[]
}

export interface ManifestData {
  metadata: {
    org: string
    namespace: string
    name: string
    displayName: string
    description: string
  }
  spec: {
    version: string
    artifacts: {
      path: string
      type: string
      required: boolean
      targetPath: string
    }[]
  }
}

export interface ReviewStatus {
  versionId: string
  status: string
  approvals: number
  requiredApprovals: number
  hasEnoughApprovals: boolean
  reviews: { id: string; action: string; comment: string; createdAt: string }[]
}

export interface AppConfig {
  org: string
  allowCreateOrg: boolean
  authMode: string
  cookieSecure: boolean
}

export interface AuthMeResult {
  authenticated: boolean
  user?: {
    id: string
    email: string
    displayName: string
    isDev: boolean
  }
}

export interface OrgResource {
  id: string
  slug: string
  displayName: string
  visibility: string
  createdAt: string
  updatedAt: string
}

const API_BASE = '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    redirect: init?.redirect ?? 'follow',
    ...init,
  })
  if (res.status >= 400) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.error?.message || `Request failed: ${res.status}`)
  }
  if (res.status === 0 || res.redirected) {
    return {} as T
  }
  const text = await res.text()
  if (!text) return {} as T
  return JSON.parse(text)
}

export const api = {
  getConfig: () => request<AppConfig>('/config'),

  createOrg: (body: { slug: string; displayName: string; visibility: string }) =>
    request<OrgResource>('/orgs', { method: 'POST', body: JSON.stringify(body) }),

  listPackages: (params?: { limit?: number; cursor?: string }) => {
    const qs = new URLSearchParams()
    if (params?.limit) qs.set('limit', String(params.limit))
    if (params?.cursor) qs.set('cursor', params.cursor)
    return request<ListPackagesResult>(`/packages?${qs}`)
  },

  getPackage: (org: string, namespace: string, name: string) =>
    request<PackageDetail>(`/packages/${org}/${namespace}/${name}`),

  resolve: (org: string, namespace: string, pkg: string, selector: string) =>
    request<ResolveResult>(`/resolve/${org}/${namespace}/${pkg}@${selector}`),

  search: (query: string, params?: { org?: string; namespace?: string; artifactType?: string; tool?: string; limit?: number }) => {
    const qs = new URLSearchParams({ q: query })
    if (params?.org) qs.set('org', params.org)
    if (params?.namespace) qs.set('namespace', params.namespace)
    if (params?.artifactType) qs.set('artifactType', params.artifactType)
    if (params?.tool) qs.set('tool', params.tool)
    if (params?.limit) qs.set('limit', String(params.limit))
    return request<SearchResult>(`/search/packages?${qs}`)
  },

  getManifest: (org: string, namespace: string, name: string, version: string) =>
    request<ManifestData>(`/packages/${org}/${namespace}/${name}/versions/${version}/manifest`),

  getAdoption: (org: string, namespace: string, name: string) =>
    request<AdoptionResult>(`/packages/${org}/${namespace}/${name}/adoption`),

  getApprovalStatus: (org: string, namespace: string, name: string, version: string) =>
    request<ReviewStatus>(`/reviews/${org}/${namespace}/${name}/versions/${version}/approval-status`),

  getRawUrl: (org: string, namespace: string, name: string, version: string, path: string) =>
    `/raw/${org}/${namespace}/${name}/${version}/${path}`,

  getArchiveUrl: (org: string, namespace: string, name: string, version: string, format: 'tar.gz' | 'zip' = 'tar.gz') =>
    `/api/v1/packages/${org}/${namespace}/${name}/versions/${version}/archive.${format}`,

  getMaintainers: (org: string, namespace: string, name: string) =>
    request<MaintainerInfo[]>(`/packages/${org}/${namespace}/${name}/maintainers`),

  addMaintainer: (org: string, namespace: string, name: string, userId: string, role: string = 'maintainer') =>
    request<void>(`/packages/${org}/${namespace}/${name}/maintainers`, {
      method: 'POST',
      body: JSON.stringify({ userId, role }),
    }),

  removeMaintainer: (org: string, namespace: string, name: string, userId: string) =>
    request<void>(`/packages/${org}/${namespace}/${name}/maintainers/${userId}`, {
      method: 'DELETE',
    }),

  updatePackageVisibility: (org: string, namespace: string, name: string, visibility: string) =>
    request<PackageDetail>(`/packages/${org}/${namespace}/${name}/visibility`, {
      method: 'PATCH',
      body: JSON.stringify({ visibility }),
    }),

  getAuthMe: () => request<AuthMeResult>('/auth/me'),

  loginDev: (token: string) =>
    request<void>('/auth/dev/login', {
      method: 'POST',
      body: JSON.stringify({ token }),
      redirect: 'manual',
    }).catch(() => {}),

  loginLocal: () =>
    request<void>('/auth/local/login', {
      method: 'POST',
      redirect: 'manual',
    }).catch(() => {}),

  logout: () =>
    request<void>('/auth/logout', {
      method: 'POST',
      redirect: 'manual',
    }).catch(() => {}),
}
