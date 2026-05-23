import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/trove/docs/',
  title: 'Trove',
  description: 'Concepts and Examples Guide',
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/' },
      { text: 'API Reference', link: '/api/public-apis' },
      { text: 'CLI Reference', link: '/cli/resolve' },
    ],
    sidebar: [
      {
        text: 'Core Concepts',
        items: [
          { text: 'What is Trove?', link: '/concepts/what-is-trove' },
          { text: 'Package References', link: '/concepts/package-references' },
          { text: 'Version Selectors', link: '/concepts/version-selectors' },
          { text: 'Lifecycle States', link: '/concepts/lifecycle-states' },
          { text: 'Visibility', link: '/concepts/visibility' },
          { text: 'Artifact Types', link: '/concepts/artifact-types' },
          { text: 'Immutability', link: '/concepts/immutability' },
        ],
      },
      {
        text: 'Publishing',
        items: [
          { text: 'Manifests', link: '/publishing/manifests' },
          { text: 'Upload & Publish Flow', link: '/publishing/upload-publish-flow' },
          { text: 'Review Workflow', link: '/publishing/review-workflow' },
          { text: 'Security Scanning', link: '/publishing/security-scanning' },
        ],
      },
      {
        text: 'Discovery',
        items: [
          { text: 'Search', link: '/discovery/search' },
          { text: 'Adoption Dashboard', link: '/discovery/adoption-dashboard' },
        ],
      },
      {
        text: 'CLI',
        items: [
          { text: 'resolve', link: '/cli/resolve' },
          { text: 'fetch', link: '/cli/fetch' },
          { text: 'install', link: '/cli/install' },
          { text: 'check', link: '/cli/check' },
          { text: 'update', link: '/cli/update' },
          { text: 'Lockfiles', link: '/cli/lockfiles' },
        ],
      },
      {
        text: 'Security',
        items: [
          { text: 'Authentication', link: '/security/authentication' },
          { text: 'RBAC & Scopes', link: '/security/rbac-scopes' },
          { text: 'API Tokens', link: '/security/api-tokens' },
        ],
      },
      {
        text: 'API',
        items: [
          { text: 'Public APIs', link: '/api/public-apis' },
          { text: 'Management APIs', link: '/api/management-apis' },
          { text: 'Raw Artifacts', link: '/api/raw-artifacts' },
          { text: 'Archives', link: '/api/archives' },
        ],
      },
      {
        text: 'Operations',
        items: [
          { text: 'Configuration', link: '/operations/configuration' },
          { text: 'Deployment', link: '/operations/deployment' },
          { text: 'Helm Chart', link: '/operations/helm-chart' },
          { text: 'Compatibility', link: '/operations/compatibility' },
        ],
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/nowhereworks/trove' },
    ],
    search: {
      provider: 'local',
    },
    editLink: {
      pattern: 'https://github.com/nowhereworks/trove/edit/main/docs-site/:path',
    },
    footer: {
      message: 'Built with VitePress',
    },
  },
})
