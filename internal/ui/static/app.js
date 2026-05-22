const packagesEl = document.querySelector("#packages");
const detailEl = document.querySelector("#detail");
const refreshEl = document.querySelector("#refresh");
const publishFormEl = document.querySelector("#publish-form");
const draftVersionEl = document.querySelector("#draft-version");
const manifestContentEl = document.querySelector("#manifest-content");
const agentsContentEl = document.querySelector("#agents-content");
const publishResultEl = document.querySelector("#publish-result");
const searchInputEl = document.querySelector("#search-input");
const searchResultsEl = document.querySelector("#search-results");
const adoptionEl = document.querySelector("#adoption");

const publishRef = "companyx/platform/agent-backend";

async function fetchJSON(url) {
  const response = await fetch(url, { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return response.json();
}

async function sendJSON(url, method, body) {
  const response = await fetch(url, {
    method,
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  return readResponse(response);
}

async function sendText(url, method, contentType, body) {
  const response = await fetch(url, {
    method,
    headers: {
      Accept: "application/json",
      "Content-Type": contentType,
    },
    body,
  });
  return readResponse(response);
}

async function readResponse(response) {
  const text = await response.text();
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) {
    const message = data?.error?.message || `${response.status} ${response.statusText}`;
    throw new Error(message);
  }
  return data;
}

function packageRef(pkg) {
  return `${pkg.org}/${pkg.namespace}/${pkg.name}`;
}

function renderPackages(items) {
  if (items.length === 0) {
    packagesEl.innerHTML = `<p class="muted">No published packages found.</p>`;
    return;
  }

  packagesEl.innerHTML = items.map((pkg) => `
    <button class="package-card" type="button" data-ref="${packageRef(pkg)}">
      <span class="package-name">${pkg.displayName}</span>
      <span class="package-ref">${packageRef(pkg)}</span>
      <span class="package-meta">stable ${pkg.stableVersion || "none"} · ${pkg.visibility}</span>
      <span>${pkg.description}</span>
    </button>
  `).join("");

  packagesEl.querySelectorAll(".package-card").forEach((button) => {
    button.addEventListener("click", () => loadDetail(button.dataset.ref));
  });
}

function renderDetail(detail) {
  const ref = packageRef(detail);
  const stable = detail.stableVersion || detail.latestVersion;
  const rawURL = `/raw/${ref}/${stable}/AGENTS.md`;
  detailEl.innerHTML = `
    <div class="detail-grid">
      <div>
        <p class="eyebrow">${detail.visibility} · ${detail.lifecycle}</p>
        <h3>${detail.displayName}</h3>
        <p>${detail.description}</p>
      </div>
      <div class="snippet">
        <span>Resolve</span>
        <code>/api/v1/resolve/${ref}@stable</code>
      </div>
      <div class="snippet">
        <span>Raw AGENTS.md</span>
        <code>${rawURL}</code>
      </div>
    </div>
    <h4>Versions</h4>
    <ul class="versions">
      ${detail.versions.map((version) => `<li><strong>${version.version}</strong> ${version.channel || ""} <code>${version.digest}</code></li>`).join("")}
    </ul>
  `;
}

function manifestTemplate(version) {
  return `apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: companyx
  namespace: platform
  name: agent-backend
  displayName: Backend Agent Defaults
  description: Default agent instructions, skills, and commands for backend services.
spec:
  version: ${version}
  channel: stable
  visibility: public
  lifecycle: draft
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
  maintainers:
    - team: platform-engineering
`;
}

function resetManifestTemplate() {
  manifestContentEl.value = manifestTemplate(draftVersionEl.value || "1.0.1");
}

async function publishDraft(event) {
  event.preventDefault();
  const version = draftVersionEl.value.trim();
  const base = `/api/v1/packages/${publishRef}/versions`;
  publishResultEl.className = "publish-result muted";
  publishResultEl.textContent = `Publishing ${publishRef}@${version}...`;

  try {
    await sendJSON(base, "POST", { version, visibility: "public" });
    await sendText(`${base}/${version}/artifacts/trove.yaml`, "PUT", "application/yaml", manifestContentEl.value);
    await sendText(`${base}/${version}/artifacts/AGENTS.md`, "PUT", "text/markdown; charset=utf-8", agentsContentEl.value);
    const published = await sendJSON(`${base}/${version}/publish`, "POST", {});
    publishResultEl.className = "publish-result success";
    publishResultEl.innerHTML = `Published <strong>${publishRef}@${published.version}</strong><br /><code>${published.digest}</code>`;
    await loadPackages();
  } catch (error) {
    publishResultEl.className = "publish-result error";
    publishResultEl.textContent = `Publish failed: ${error.message}`;
  }
}

async function loadPackages() {
  packagesEl.innerHTML = `<p class="muted">Loading packages...</p>`;
  try {
    const data = await fetchJSON("/api/v1/packages");
    renderPackages(data.items || []);
    if ((data.items || []).length > 0) {
      loadDetail(packageRef(data.items[0]));
    }
  } catch (error) {
    packagesEl.innerHTML = `<p class="error">Could not load packages: ${error.message}</p>`;
  }
}

async function searchPackages(query) {
  if (!query || query.trim() === "") {
    searchResultsEl.innerHTML = "";
    searchResultsEl.style.display = "none";
    return;
  }
  try {
    const data = await fetchJSON(`/api/v1/search/packages?q=${encodeURIComponent(query)}`);
    renderSearchResults(data.items || []);
  } catch (error) {
    searchResultsEl.innerHTML = `<p class="error">Search failed: ${error.message}</p>`;
    searchResultsEl.style.display = "block";
  }
}

function renderSearchResults(items) {
  if (items.length === 0) {
    searchResultsEl.innerHTML = `<p class="muted">No results found.</p>`;
    searchResultsEl.style.display = "block";
    return;
  }

  searchResultsEl.innerHTML = items.map((pkg) => `
    <button class="package-card" type="button" data-ref="${packageRef(pkg)}">
      <span class="package-name">${pkg.displayName}</span>
      <span class="package-ref">${packageRef(pkg)}</span>
      <span class="package-meta">${pkg.visibility}</span>
      <span>${pkg.description}</span>
    </button>
  `).join("");

  searchResultsEl.style.display = "block";
  searchResultsEl.querySelectorAll(".package-card").forEach((button) => {
    button.addEventListener("click", () => {
      loadDetail(button.dataset.ref);
      searchResultsEl.style.display = "none";
      searchInputEl.value = "";
    });
  });
}

async function loadAdoption(ref) {
  if (!adoptionEl) return;
  adoptionEl.innerHTML = `<p class="muted">Loading adoption data...</p>`;
  try {
    const data = await fetchJSON(`/api/v1/packages/${ref}/adoption`);
    renderAdoption(data);
  } catch (error) {
    adoptionEl.innerHTML = `<p class="error">Could not load adoption data: ${error.message}</p>`;
  }
}

function renderAdoption(data) {
  if (!data || data.projectCount === 0) {
    adoptionEl.innerHTML = `<p class="muted">No adoption data yet.</p>`;
    return;
  }

  let html = `<h4>Adoption</h4>`;
  html += `<p><strong>${data.projectCount}</strong> project(s) using this package</p>`;
  if (data.versions && data.versions.length > 0) {
    html += `<ul class="versions">`;
    for (const v of data.versions) {
      html += `<li><strong>${v.version}</strong> - ${v.installCount} install(s)</li>`;
    }
    html += `</ul>`;
  }
  adoptionEl.innerHTML = html;
}

async function loadDetail(ref) {
  detailEl.innerHTML = `<p class="muted">Loading ${ref}...</p>`;
  try {
    const detail = await fetchJSON(`/api/v1/packages/${ref}`);
    renderDetail(detail);
    loadAdoption(ref);
  } catch (error) {
    detailEl.innerHTML = `<p class="error">Could not load detail: ${error.message}</p>`;
  }
}

let searchTimeout;
if (searchInputEl) {
  searchInputEl.addEventListener("input", (e) => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => searchPackages(e.target.value), 300);
  });
}

refreshEl.addEventListener("click", loadPackages);
draftVersionEl.addEventListener("input", resetManifestTemplate);
publishFormEl.addEventListener("submit", publishDraft);
resetManifestTemplate();
loadPackages();
