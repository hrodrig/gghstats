function currentTheme() {
  const theme = document.documentElement.getAttribute('data-bs-theme');
  return theme === 'dark' ? 'dark' : 'light';
}

const mouseLinePlugin = {
  afterDraw: chart => {
    if (!chart.tooltip?._active?.length) return;
    const x = chart.tooltip._active[0].element.x;
    const yAxis = chart.scales.y;
    const ctx = chart.ctx;
    ctx.save();
    ctx.beginPath();
    ctx.moveTo(x, yAxis.top);
    ctx.lineTo(x, yAxis.bottom);
    ctx.lineWidth = 1;
    ctx.strokeStyle = currentTheme() === 'dark'
      ? 'rgba(255, 255, 255, 0.35)'
      : 'rgba(100, 149, 237, 0.45)';
    ctx.stroke();
    ctx.restore();
  }
};

function chartThemeColors() {
  const dark = currentTheme() === 'dark';
  const body = document.body;
  const root = getComputedStyle(body.classList.contains('app-brutalist') ? body : document.documentElement);

  if (dark) {
    return {
      fg: '#f2f2ed',
      grid: 'rgba(242, 242, 237, 0.18)',
      border: 'rgba(242, 242, 237, 0.35)',
      primary: root.getPropertyValue('--bs-primary').trim() || '#7cfc00',
      info: root.getPropertyValue('--bs-info').trim() || '#5ce1e6'
    };
  }

  return {
    fg: root.getPropertyValue('--bs-body-color').trim() || '#1a1a1b',
    grid: root.getPropertyValue('--bs-border-color-translucent').trim() ||
      root.getPropertyValue('--bs-border-color').trim() ||
      'rgba(0,0,0,0.12)',
    border: 'rgba(26, 26, 27, 0.25)',
    primary: root.getPropertyValue('--bs-primary').trim() || 'rgb(230, 57, 70)',
    info: root.getPropertyValue('--bs-info').trim() || 'rgb(29, 53, 87)'
  };
}

function chartTooltipOptions() {
  const dark = currentTheme() === 'dark';
  return {
    intersect: false,
    backgroundColor: dark ? '#16161a' : '#ffffff',
    titleColor: dark ? '#fafaf6' : '#121212',
    bodyColor: dark ? '#e8e8e3' : '#2a2a2a',
    borderColor: dark ? 'rgba(242, 242, 237, 0.4)' : '#121212',
    borderWidth: 2,
    padding: 10,
    displayColors: true
  };
}

function applyTheme(theme) {
  document.documentElement.setAttribute('data-bs-theme', theme);
  localStorage.setItem('gghstats-theme', theme);
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.textContent = theme === 'dark' ? 'Dark' : 'Light';
  }
}

function toggleTheme() {
  applyTheme(currentTheme() === 'dark' ? 'light' : 'dark');
  requestAnimationFrame(() => {
    refreshRepoCharts();
    refreshIndexListCharts();
  });
}

function initThemeToggle() {
  const btn = document.getElementById('theme-toggle');
  if (!btn) return;
  if (!document.documentElement.getAttribute('data-bs-theme')) {
    const preferredDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    applyTheme(preferredDark ? 'dark' : 'light');
  } else {
    applyTheme(currentTheme());
  }
  btn.addEventListener('click', toggleTheme);
}

const repoChartCanvasIds = ['chart_clones', 'chart_views', 'chart_stars'];

function destroyRepoCharts() {
  if (typeof Chart === 'undefined' || typeof Chart.getChart !== 'function') return;
  for (const id of repoChartCanvasIds) {
    const el = document.getElementById(id);
    if (!el) continue;
    const existing = Chart.getChart(el);
    if (existing) existing.destroy();
  }
}

function initRepoCharts() {
  const payload = window.gghstatsChartData;
  if (!payload) return;

  renderMetrics('chart_clones', payload.clones, 'uniques', 'count');
  renderMetrics('chart_views', payload.views, 'uniques', 'count');
  if (payload.stars && payload.stars.length > 0) {
    renderStars('chart_stars', payload.stars);
  }
}

function refreshRepoCharts() {
  if (!window.gghstatsChartData) return;
  destroyRepoCharts();
  initRepoCharts();
}

const indexListChartCanvasIds = ['chart_index_clones'];

function destroyIndexListCharts() {
  if (typeof Chart === 'undefined' || typeof Chart.getChart !== 'function') return;
  for (const id of indexListChartCanvasIds) {
    const el = document.getElementById(id);
    if (!el) continue;
    const existing = Chart.getChart(el);
    if (existing) existing.destroy();
  }
}

function renderIndexClonesOverTime(canvasId, data) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  new Chart(el, {
    type: 'line',
    data: {
      labels: data.map(d => d.date),
      datasets: [{
        label: 'Clones (count)',
        data: data.map(d => d.count),
        borderColor: c.primary,
        backgroundColor: 'transparent',
        pointStyle: false,
        tension: 0.1,
        borderWidth: 2
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index' },
      scales: {
        x: {
          ticks: {
            color: c.fg,
            maxRotation: 45,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: c.fg,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: chartTooltipOptions()
      }
    },
    plugins: [mouseLinePlugin]
  });
}

function initIndexListCharts() {
  const payload = window.gghstatsListClonesData;
  if (!payload || !Array.isArray(payload) || payload.length === 0) return;
  renderIndexClonesOverTime('chart_index_clones', payload);
}

function refreshIndexListCharts() {
  destroyIndexListCharts();
  initIndexListCharts();
}

function initBadgeEmbed() {
  const root = document.getElementById('gghstats-badge-embed');
  if (!root) return;

  const base = (root.dataset.baseUrl || window.location.origin).replace(/\/$/, '');
  const repo = root.dataset.repo;
  if (!repo) return;

  const metricEl = document.getElementById('badge-metric');
  const preview = document.getElementById('badge-preview');
  const markdown = document.getElementById('badge-markdown');
  const copyBtn = document.getElementById('badge-copy-btn');
  const copyStatus = document.getElementById('badge-copy-status');
  if (!metricEl || !preview || !markdown) return;

  const metricLabels = {
    clones: 'clones',
    clones_30d: 'clones 30d',
    views: 'views',
    stars: 'stars'
  };

  function badgeURL(metric) {
    const u = new URL(`${base}/api/v1/badge/${repo}`);
    u.searchParams.set('metric', metric);
    return u.toString();
  }

  function repoPageURL() {
    const path = repo.split('/').map(encodeURIComponent).join('/');
    return `${base}/${path}`;
  }

  function altText(metric) {
    return `gghstats ${metricLabels[metric] || metric}`;
  }

  function markdownSnippet(metric) {
    const img = badgeURL(metric);
    return `[![${altText(metric)}](${img})](${repoPageURL()})`;
  }

  function update() {
    const metric = metricEl.value;
    const url = badgeURL(metric);
    preview.src = url;
    preview.alt = `${altText(metric)} for ${repo}`;
    markdown.value = markdownSnippet(metric);
    const previewLink = document.getElementById('badge-preview-link');
    if (previewLink) {
      previewLink.href = repoPageURL();
    }
    if (copyStatus) copyStatus.classList.add('d-none');
  }

  metricEl.addEventListener('change', update);
  update();

  if (copyBtn) {
    copyBtn.addEventListener('click', async () => {
      const text = markdown.value;
      try {
        await navigator.clipboard.writeText(text);
      } catch {
        markdown.focus();
        markdown.select();
        document.execCommand('copy');
      }
      if (copyStatus) copyStatus.classList.remove('d-none');
    });
  }
}

const SYNC_TOKEN_KEY = 'gghstats-api-token';

function syncApiToken() {
  return sessionStorage.getItem(SYNC_TOKEN_KEY) || '';
}

function syncScopeRepo() {
  const btn = document.getElementById('sync-now-btn');
  return btn?.dataset?.syncRepo?.trim() || '';
}

function syncPostURL() {
  const repo = syncScopeRepo();
  if (repo) return `/api/v1/sync?repo=${encodeURIComponent(repo)}`;
  return '/api/v1/sync';
}

function formatSyncStatus(st) {
  if (!st) return '';
  if (st.running) {
    if (st.scope === 'repo' && st.repo) return `Syncing ${st.repo}…`;
    return 'Syncing all repositories…';
  }
  if (st.last_error) return `Last sync failed: ${st.last_error}`;
  if (st.last_finished_at) {
    const when = new Date(st.last_finished_at).toLocaleString();
    if (st.scope === 'repo' && st.repo) return `Last sync (${st.repo}): ${when}`;
    return `Last sync: ${when}`;
  }
  return 'No sync completed yet';
}

async function fetchSyncStatus() {
  const token = syncApiToken();
  if (!token) return null;
  const res = await fetch('/api/v1/sync', { headers: { 'x-api-token': token } });
  if (!res.ok) return null;
  return res.json();
}

/** @returns {Promise<string|null>} */
function requestSyncTokenModal({ invalid = false } = {}) {
  const modalEl = document.getElementById('sync-token-modal');
  if (!modalEl || typeof bootstrap === 'undefined') {
    return Promise.resolve(null);
  }

  const input = document.getElementById('sync-token-input');
  const errorEl = document.getElementById('sync-token-error');
  const submitBtn = document.getElementById('sync-token-submit');
  if (!input || !errorEl || !submitBtn) return Promise.resolve(null);

  return new Promise((resolve) => {
    let settled = false;
    const modal = bootstrap.Modal.getOrCreateInstance(modalEl);

    const finish = (token) => {
      if (settled) return;
      settled = true;
      input.removeEventListener('keydown', onKeydown);
      submitBtn.removeEventListener('click', onSubmit);
      modalEl.removeEventListener('hidden.bs.modal', onDismiss);
      modal.hide();
      resolve(token);
    };

    const showError = (msg) => {
      errorEl.textContent = msg;
      errorEl.classList.remove('d-none');
    };

    const onSubmit = () => {
      const token = input.value.trim();
      if (!token) {
        showError('Token is required.');
        input.focus();
        return;
      }
      finish(token);
    };

    const onKeydown = (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        onSubmit();
      }
    };

    const onDismiss = () => finish(null);

    input.value = '';
    errorEl.classList.add('d-none');
    errorEl.textContent = '';
    if (invalid) {
      showError('Invalid API token. Check GGHSTATS_API_TOKEN and try again.');
    }

    submitBtn.addEventListener('click', onSubmit);
    input.addEventListener('keydown', onKeydown);
    modalEl.addEventListener('hidden.bs.modal', onDismiss, { once: true });

    modal.show();
    modalEl.addEventListener(
      'shown.bs.modal',
      () => input.focus(),
      { once: true }
    );
  });
}

function initSyncControl() {
  const btn = document.getElementById('sync-now-btn');
  const statusEl = document.getElementById('sync-status');
  if (!btn || !statusEl) return;

  let pollTimer = null;
  let wasRunning = false;
  const pageRepo = syncScopeRepo();

  const refreshStatus = async () => {
    try {
      const st = await fetchSyncStatus();
      if (st) statusEl.textContent = formatSyncStatus(st);
      if (wasRunning && st && !st.running && pageRepo && st.repo === pageRepo && !st.last_error) {
        window.location.reload();
        return;
      }
      wasRunning = !!st?.running;
      if (st?.running) {
        btn.disabled = true;
        if (!pollTimer) pollTimer = setInterval(refreshStatus, 2000);
      } else {
        btn.disabled = false;
        if (pollTimer) {
          clearInterval(pollTimer);
          pollTimer = null;
        }
      }
    } catch {
      statusEl.textContent = 'Could not load sync status';
    }
  };

  const runSync = async (token) => {
    btn.disabled = true;
    statusEl.textContent = pageRepo ? `Starting sync for ${pageRepo}…` : 'Starting sync…';
    try {
      const res = await fetch(syncPostURL(), {
        method: 'POST',
        headers: { 'x-api-token': token }
      });
      if (res.status === 401) {
        sessionStorage.removeItem(SYNC_TOKEN_KEY);
        statusEl.textContent = 'Invalid API token';
        btn.disabled = false;
        const retry = await requestSyncTokenModal({ invalid: true });
        if (retry) {
          sessionStorage.setItem(SYNC_TOKEN_KEY, retry);
          await runSync(retry);
        }
        return;
      }
      if (res.status === 404) {
        statusEl.textContent = 'Sync API disabled (set GGHSTATS_API_TOKEN)';
        btn.disabled = false;
        return;
      }
      if (res.status === 409) {
        statusEl.textContent = 'Sync already running';
      } else if (!res.ok) {
        statusEl.textContent = 'Could not start sync';
      }
      await refreshStatus();
    } catch {
      statusEl.textContent = 'Could not start sync';
      btn.disabled = false;
    }
  };

  btn.addEventListener('click', async () => {
    let token = syncApiToken();
    if (!token) {
      token = await requestSyncTokenModal();
      if (!token) return;
      sessionStorage.setItem(SYNC_TOKEN_KEY, token);
    }
    await runSync(token);
  });

  refreshStatus();
}

document.addEventListener('DOMContentLoaded', () => {
  initThemeToggle();
  initRepoCharts();
  initIndexListCharts();
  initBadgeEmbed();
  initSyncControl();
});

function renderMetrics(canvasId, data, uniqueCol, countCol) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  new Chart(el, {
    type: 'bar',
    data: {
      labels: data.map(d => d.date),
      datasets: [
        {
          label: 'Unique',
          data: data.map(d => d[uniqueCol]),
          backgroundColor: c.primary,
          borderWidth: 0,
          borderRadius: 4
        },
        {
          label: 'Count',
          data: data.map(d => d[countCol]),
          backgroundColor: c.info,
          borderWidth: 0,
          borderRadius: 4
        }
      ]
    },
    options: {
      responsive: true,
      interaction: { mode: 'index' },
      scales: {
        x: {
          stacked: true,
          ticks: {
            color: c.fg,
            maxRotation: 45,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: c.fg,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: chartTooltipOptions()
      }
    },
    plugins: [mouseLinePlugin]
  });
}

function renderStars(canvasId, data) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  new Chart(el, {
    type: 'line',
    data: {
      labels: data.map(d => d.date),
      datasets: [{
        label: 'Stars',
        data: data.map(d => d.total),
        borderColor: c.primary,
        backgroundColor: 'transparent',
        pointStyle: false,
        tension: 0.1,
        borderWidth: 2
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index' },
      scales: {
        x: {
          ticks: {
            color: c.fg,
            maxRotation: 45,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: c.fg,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: chartTooltipOptions()
      }
    },
    plugins: [mouseLinePlugin]
  });
}
