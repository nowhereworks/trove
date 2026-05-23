---
layout: home
hero:
  name: Trove
  text: Concepts & Examples Guide
  tagline: A curated registry for agent-consumable engineering artifacts. Browse, publish, and manage AGENTS.md files, skills, commands, prompts, and more.
---

<script setup>
import { ref } from 'vue'

const categories = ref([
  {
    title: 'Core Concepts',
    items: [
      { title: 'What is Trove?', desc: 'Understand the problem Trove solves and how the registry works.', link: '/concepts/what-is-trove' },
      { title: 'Package References', desc: 'How org/namespace/package@selector identifies every package.', link: '/concepts/package-references' },
      { title: 'Version Selectors', desc: 'Resolve @latest, @stable, and digests to exact versions.', link: '/concepts/version-selectors' },
      { title: 'Lifecycle States', desc: 'draft → review → approved → published → deprecated | yanked.', link: '/concepts/lifecycle-states' },
      { title: 'Visibility', desc: 'Private, internal, and public access control with inheritance.', link: '/concepts/visibility' },
      { title: 'Artifact Types', desc: '10 first-class types: skills, commands, prompts, and more.', link: '/concepts/artifact-types' },
      { title: 'Immutability', desc: 'Why published versions never change and how fixes work.', link: '/concepts/immutability' },
    ],
  },
  {
    title: 'Publishing',
    items: [
      { title: 'Manifests', desc: 'trove.yaml structure, required fields, and validation rules.', link: '/publishing/manifests' },
      { title: 'Upload & Publish Flow', desc: 'Step-by-step from draft creation to published version.', link: '/publishing/upload-publish-flow' },
      { title: 'Review Workflow', desc: 'Submit, automated checks, human review, and approval.', link: '/publishing/review-workflow' },
      { title: 'Security Scanning', desc: 'Secret detection and unsafe instruction scanning.', link: '/publishing/security-scanning' },
    ],
  },
  {
    title: 'Discovery',
    items: [
      { title: 'Search', desc: 'Find packages by name, type, tool compatibility, and more.', link: '/discovery/search' },
      { title: 'Adoption Dashboard', desc: 'See which projects use your packages.', link: '/discovery/adoption-dashboard' },
    ],
  },
  {
    title: 'CLI',
    items: [
      { title: 'resolve', desc: 'Get the exact version from a selector.', link: '/cli/resolve' },
      { title: 'fetch', desc: 'Download individual artifacts.', link: '/cli/fetch' },
      { title: 'install', desc: 'Install required artifacts and pin versions.', link: '/cli/install' },
      { title: 'check', desc: 'Detect updates, yanked versions, and incompatibilities.', link: '/cli/check' },
      { title: 'update', desc: 'Safe dry-run updates with explicit apply.', link: '/cli/update' },
      { title: 'Lockfiles', desc: 'Reproducible installs with .trove.lock.yaml.', link: '/cli/lockfiles' },
    ],
  },
  {
    title: 'Security',
    items: [
      { title: 'Authentication', desc: 'OIDC for humans, API tokens for agents and CI.', link: '/security/authentication' },
      { title: 'RBAC & Scopes', desc: 'Roles, scopes, and least-privilege access.', link: '/security/rbac-scopes' },
      { title: 'API Tokens', desc: 'Create, scope, restrict, and revoke machine access.', link: '/security/api-tokens' },
    ],
  },
  {
    title: 'API',
    items: [
      { title: 'Public APIs', desc: 'Agent-facing endpoints for resolve, search, and fetch.', link: '/api/public-apis' },
      { title: 'Management APIs', desc: 'Write endpoints for the full package lifecycle.', link: '/api/management-apis' },
      { title: 'Raw Artifacts', desc: 'Direct artifact access with ETags and caching.', link: '/api/raw-artifacts' },
      { title: 'Archives', desc: 'Download full package archives on demand.', link: '/api/archives' },
    ],
  },
  {
    title: 'Operations',
    items: [
      { title: 'Configuration', desc: 'Full config reference for server, auth, storage, and more.', link: '/operations/configuration' },
      { title: 'Deployment', desc: 'Single binary + PostgreSQL, migrations, and health checks.', link: '/operations/deployment' },
      { title: 'Compatibility', desc: 'Tool, model, and runtime compatibility rules.', link: '/operations/compatibility' },
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
