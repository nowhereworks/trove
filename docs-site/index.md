---
layout: home
hero:
  name: Trove
  text: Concepts & Examples Guide
  tagline: A curated registry for agent-consumable engineering artifacts. Browse, publish, and manage AGENTS.md files, skills, commands, prompts, and more.
---

::: warning Pre-v1.0 Software
Trove is pre-v1.0 software. It is highly unstable and subject to heavy changes, including additions, removals, and breaking changes to functionality.

The API contract is not guaranteed at this stage. Use at your own risk.
:::

<script setup>
import { ref } from 'vue'
import { withBase } from 'vitepress'

const categories = ref([
  {
    title: 'Core Concepts',
    items: [
      { title: 'What is Trove?', desc: 'Understand the problem Trove solves and how the registry works.', link: withBase('/concepts/what-is-trove') },
      { title: 'Package References', desc: 'How org/namespace/package@selector identifies every package.', link: withBase('/concepts/package-references') },
      { title: 'Version Selectors', desc: 'Resolve @latest and digests to exact versions.', link: withBase('/concepts/version-selectors') },
      { title: 'Lifecycle States', desc: 'draft → review → approved → published → deprecated | yanked.', link: withBase('/concepts/lifecycle-states') },
      { title: 'Visibility', desc: 'Private, internal, and public access control with inheritance.', link: withBase('/concepts/visibility') },
      { title: 'Artifact Types', desc: '10 first-class types: skills, commands, prompts, and more.', link: withBase('/concepts/artifact-types') },
      { title: 'Immutability', desc: 'Why published versions never change and how fixes work.', link: withBase('/concepts/immutability') },
    ],
  },
  {
    title: 'Publishing',
    items: [
      { title: 'Manifests', desc: 'Trovefile structure, required fields, and validation rules.', link: withBase('/publishing/manifests') },
      { title: 'Upload & Publish Flow', desc: 'Step-by-step from draft creation to published version.', link: withBase('/publishing/upload-publish-flow') },
      { title: 'Review Workflow', desc: 'Submit, automated checks, human review, and approval.', link: withBase('/publishing/review-workflow') },
      { title: 'Security Scanning', desc: 'Secret detection and unsafe instruction scanning.', link: withBase('/publishing/security-scanning') },
    ],
  },
  {
    title: 'Discovery',
    items: [
      { title: 'Search', desc: 'Find packages by name, type, and more.', link: withBase('/discovery/search') },
      { title: 'Adoption Dashboard', desc: 'See which projects use your packages.', link: withBase('/discovery/adoption-dashboard') },
    ],
  },
  {
    title: 'CLI',
    items: [
      { title: 'init', desc: 'Create an editable AGENTS.md package worktree.', link: withBase('/cli/init') },
      { title: 'remote', desc: 'Manage local publishing remotes.', link: withBase('/cli/remote') },
      { title: 'status', desc: 'Check local publishing readiness.', link: withBase('/cli/status') },
      { title: 'push', desc: 'Upload, publish, or submit AGENTS.md for review.', link: withBase('/cli/push') },
      { title: 'clone', desc: 'Clone a published package for maintainer edits.', link: withBase('/cli/clone') },
      { title: 'pull', desc: 'Refresh a cloned publishing worktree safely.', link: withBase('/cli/pull') },
      { title: 'resolve', desc: 'Get the exact version from a selector.', link: withBase('/cli/resolve') },
      { title: 'download', desc: 'Download individual artifacts.', link: withBase('/cli/download') },
      { title: 'install', desc: 'Install required artifacts and pin versions.', link: withBase('/cli/install') },
      { title: 'check', desc: 'Detect updates, yanked versions, and incompatibilities.', link: withBase('/cli/check') },
      { title: 'update', desc: 'Safe dry-run updates with explicit apply.', link: withBase('/cli/update') },
      { title: 'skills', desc: 'Find Trove-hosted reusable agent skills.', link: withBase('/cli/skills') },
      { title: 'Lockfiles', desc: 'Reproducible installs with .trove.lock.yaml.', link: withBase('/cli/lockfiles') },
    ],
  },
  {
    title: 'Security',
    items: [
      { title: 'Authentication', desc: 'OIDC for humans, API tokens for agents and CI.', link: withBase('/security/authentication') },
      { title: 'Azure Entra ID', desc: 'Set up Microsoft Entra ID as Trove\'s OIDC provider.', link: withBase('/security/azure-entra-id') },
      { title: 'RBAC & Scopes', desc: 'Roles, scopes, and least-privilege access.', link: withBase('/security/rbac-scopes') },
      { title: 'API Tokens', desc: 'Create, scope, restrict, and revoke machine access.', link: withBase('/security/api-tokens') },
    ],
  },
  {
    title: 'API',
    items: [
      { title: 'Public APIs', desc: 'Agent-facing endpoints for resolve, search, and raw downloads.', link: withBase('/api/public-apis') },
      { title: 'Management APIs', desc: 'Write endpoints for the full package lifecycle.', link: withBase('/api/management-apis') },
      { title: 'Raw Artifacts', desc: 'Direct artifact access with ETags and caching.', link: withBase('/api/raw-artifacts') },
      { title: 'Archives', desc: 'Download full package archives on demand.', link: withBase('/api/archives') },
    ],
  },
    {
    title: 'Operations',
    items: [
      { title: 'Configuration', desc: 'Full config reference for server, auth, storage, and more.', link: withBase('/operations/configuration') },
      { title: 'Deployment', desc: 'Single binary + PostgreSQL, migrations, and health checks.', link: withBase('/operations/deployment') },
      { title: 'Helm Chart', desc: 'Install from GHCR and configure Kubernetes values.', link: withBase('/operations/helm-chart') },
    ],
  },
])
</script>

<div v-for="cat in categories" :key="cat.title">
  <div class="category-header">
    <h2>{{ cat.title }}</h2>
  </div>
  <div class="card-grid">
    <a v-for="item in cat.items" :key="item.title" :href="item.link" class="card-grid-item">
      <h3>{{ item.title }}</h3>
      <p>{{ item.desc }}</p>
    </a>
  </div>
</div>
