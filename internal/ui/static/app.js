const packagesEl = document.querySelector("#packages");
const detailEl = document.querySelector("#detail");
const refreshEl = document.querySelector("#refresh");

async function fetchJSON(url) {
  const response = await fetch(url, { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return response.json();
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

async function loadPackages() {
  packagesEl.innerHTML = `<p class="muted">Loading packages...</p>`;
  try {
    const data = await fetchJSON("/api/v1/search/packages");
    renderPackages(data.items || []);
    if ((data.items || []).length > 0) {
      loadDetail(packageRef(data.items[0]));
    }
  } catch (error) {
    packagesEl.innerHTML = `<p class="error">Could not load packages: ${error.message}</p>`;
  }
}

async function loadDetail(ref) {
  detailEl.innerHTML = `<p class="muted">Loading ${ref}...</p>`;
  try {
    const detail = await fetchJSON(`/api/v1/packages/${ref}`);
    renderDetail(detail);
  } catch (error) {
    detailEl.innerHTML = `<p class="error">Could not load detail: ${error.message}</p>`;
  }
}

refreshEl.addEventListener("click", loadPackages);
loadPackages();
