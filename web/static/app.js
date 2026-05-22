/** UI strings injected by the server (locale JSON keys → translated text). */
function uiT(key, vars) {
  const bag = window.gghstatsI18n || {};
  let s = bag[key] || key;
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      s = s.replaceAll(`{{${k}}}`, String(v));
    }
  }
  return s;
}

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
    btn.textContent = theme === 'dark' ? uiT('common.theme_dark') : uiT('common.theme_light');
  }
}

function toggleTheme() {
  applyTheme(currentTheme() === 'dark' ? 'light' : 'dark');
  requestAnimationFrame(() => {
    refreshRepoCharts();
    refreshIndexListCharts();
    refreshH2HCharts();
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

function repoChartLegendLabel(metricLabel, repoName) {
  const base = String(metricLabel || '').trim();
  const repo = String(repoName || '').trim();
  if (!base) return repo;
  if (!repo) return base;
  return `${base} - ${repo}`;
}

function repoChartLegendOptions(c) {
  return {
    display: true,
    position: 'bottom',
    labels: {
      color: c.fg,
      boxWidth: 12,
      padding: 14,
      font: { size: 11, family: "'JetBrains Mono', monospace" }
    }
  };
}

function repoChartTooltipOptions(chartTitle, repoName) {
  const chartLabel = repoChartLegendLabel(chartTitle, repoName);
  const base = chartTooltipOptions();
  return {
    ...base,
    callbacks: {
      ...base.callbacks,
      title(tooltipItems) {
        const date = tooltipItems?.[0]?.label || '';
        if (!chartLabel) return date;
        return date ? `${chartLabel} · ${date}` : chartLabel;
      }
    }
  };
}

function initRepoCharts() {
  const payload = window.gghstatsChartData;
  if (!payload) return;

  const repoName = payload.repoName || '';
  const chartLabels = payload.chartLabels || {};
  const clonesTitle = chartLabels.clones || repoChartLegendLabel('Clones', repoName);
  const viewsTitle = chartLabels.views || repoChartLegendLabel('Views', repoName);
  const starsTitle = chartLabels.stars || repoChartLegendLabel('Stars over time', repoName);
  renderMetrics('chart_clones', payload.clones, 'uniques', 'count', clonesTitle);
  renderMetrics('chart_views', payload.views, 'uniques', 'count', viewsTitle);
  if (payload.stars && payload.stars.length > 0) {
    renderStars('chart_stars', payload.stars, starsTitle);
  }
}

function refreshRepoCharts() {
  if (!window.gghstatsChartData) return;
  destroyRepoCharts();
  initRepoCharts();
}

let h2hCloneChart;
let h2hViewChart;
let h2hMomentumChart;

function destroyH2HCharts() {
  if (h2hCloneChart) {
    h2hCloneChart.destroy();
    h2hCloneChart = null;
  }
  if (h2hViewChart) {
    h2hViewChart.destroy();
    h2hViewChart = null;
  }
  if (h2hMomentumChart) {
    h2hMomentumChart.destroy();
    h2hMomentumChart = null;
  }
}

function coerceSeries(values) {
  if (!Array.isArray(values)) return [];
  return values.map(v => {
    if (v === null || v === undefined) return null;
    const n = Number(v);
    return Number.isFinite(n) ? n : null;
  });
}

function seriesMax(values) {
  let max = 0;
  for (const v of values) {
    if (v === null || v === undefined) continue;
    if (v > max) max = v;
  }
  return max;
}

function seriesMin(values) {
  let min = 0;
  let any = false;
  for (const v of values) {
    if (v === null || v === undefined) continue;
    if (!any) {
      min = v;
      any = true;
    } else if (v < min) {
      min = v;
    }
  }
  return any ? min : 0;
}

function seriesMaxAbs(values) {
  let max = 0;
  for (const v of values) {
    if (v === null || v === undefined) continue;
    const abs = Math.abs(v);
    if (abs > max) max = abs;
  }
  return max;
}

function formatChartTick(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '';
  const abs = Math.abs(n);
  if (abs >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (abs >= 1000) return `${(n / 1000).toFixed(1).replace(/\.0$/, '')}k`;
  return String(Math.round(n));
}

function formatMomentumTick(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '';
  const pct = n * 100;
  const rounded = Math.round(pct);
  return `${rounded > 0 ? '+' : ''}${rounded}%`;
}

function h2hDualAxisNeeded(seriesA, seriesB, asPercent) {
  if (asPercent) {
    const maxAbsA = seriesMaxAbs(seriesA);
    const maxAbsB = seriesMaxAbs(seriesB);
    const smaller = Math.min(maxAbsA, maxAbsB);
    const larger = Math.max(maxAbsA, maxAbsB);
    if (larger >= 0.05 && smaller < 0.05) return true;
    if (smaller >= 1e-9 && larger / smaller >= 8) return true;
    const maxA = seriesMax(seriesA);
    const maxB = seriesMax(seriesB);
    const minA = seriesMin(seriesA);
    const minB = seriesMin(seriesB);
    if (maxA > 0.05 && minB < -0.05) return true;
    if (maxB > 0.05 && minA < -0.05) return true;
    return false;
  }
  const maxA = seriesMax(seriesA);
  const maxB = seriesMax(seriesB);
  if (maxA < 1 || maxB < 1) return false;
  const ratio = maxA > maxB ? maxA / maxB : maxB / maxA;
  return ratio >= 8;
}

function renderH2HLineChart(canvasId, labels, seriesA, seriesB, labelA, labelB, opts = {}) {
  const el = document.getElementById(canvasId);
  if (!el || !labels || labels.length === 0) return null;

  const asPercent = opts.asPercent === true;
  const dataA = coerceSeries(seriesA);
  const dataB = coerceSeries(seriesB);
  const dualAxis = opts.forceDualAxis === true || h2hDualAxisNeeded(dataA, dataB, asPercent);

  const c = chartThemeColors();
  const tickFormatter = asPercent ? formatMomentumTick : formatChartTick;
  const tickStyle = {
    color: c.fg,
    font: { size: 11, family: "'JetBrains Mono', monospace" },
    callback: tickFormatter
  };

  const yAxis = {
    beginAtZero: !asPercent,
    grace: '8%',
    ticks: tickStyle,
    grid: { color: c.grid },
    border: { display: true, color: c.border, width: 1 }
  };

  const showPoints = opts.showPoints === true || asPercent;

  const scales = {
    x: {
      ticks: {
        color: c.fg,
        maxRotation: 45,
        minRotation: 45,
        autoSkip: true,
        maxTicksLimit: 14,
        font: { size: 11, family: "'JetBrains Mono', monospace" }
      },
      grid: { color: c.grid },
      border: { display: true, color: c.border, width: 1 }
    },
    y: { ...yAxis }
  };

  const datasets = [
    {
      label: labelA,
      data: dataA,
      borderColor: c.primary,
      backgroundColor: 'transparent',
      pointStyle: showPoints ? 'circle' : false,
      pointRadius: showPoints ? 3 : 0,
      pointHoverRadius: showPoints ? 4 : 0,
      tension: 0.15,
      borderWidth: 2,
      spanGaps: asPercent,
      yAxisID: 'y'
    },
    {
      label: labelB,
      data: dataB,
      borderColor: c.info,
      backgroundColor: 'transparent',
      pointStyle: showPoints ? 'circle' : false,
      pointRadius: showPoints ? 3 : 0,
      pointHoverRadius: showPoints ? 4 : 0,
      tension: 0.15,
      borderWidth: 2,
      spanGaps: asPercent,
      yAxisID: dualAxis ? 'y1' : 'y'
    }
  ];

  if (dualAxis) {
    scales.y.ticks = { ...tickStyle, color: asPercent ? c.primary : c.fg };
    scales.y1 = {
      ...yAxis,
      position: 'right',
      grid: { drawOnChartArea: false, color: c.grid },
      ticks: { ...tickStyle, color: c.info }
    };
  }

  return new Chart(el, {
    type: 'line',
    data: { labels, datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index' },
      scales,
      plugins: {
        legend: {
          display: true,
          labels: { color: c.fg, font: { family: "'JetBrains Mono', monospace", size: 11 } }
        },
        tooltip: chartTooltipOptions()
      }
    },
    plugins: [mouseLinePlugin]
  });
}

function initH2HCharts() {
  const payload = window.gghstatsH2HChartData;
  if (!payload) return;

  destroyH2HCharts();
  h2hCloneChart = renderH2HLineChart(
    'h2h-clones-chart',
    payload.cloneLabels,
    payload.clonesA,
    payload.clonesB,
    payload.repoA,
    payload.repoB
  );
  h2hViewChart = renderH2HLineChart(
    'h2h-views-chart',
    payload.viewLabels,
    payload.viewsA,
    payload.viewsB,
    payload.repoA,
    payload.repoB
  );
  if (payload.showMomentum && payload.momentumLabels && payload.momentumLabels.length > 0) {
    h2hMomentumChart = renderH2HLineChart(
      'h2h-momentum-chart',
      payload.momentumLabels,
      payload.momentumA,
      payload.momentumB,
      payload.repoA,
      payload.repoB,
      { asPercent: true, showPoints: true, forceDualAxis: true }
    );
  }
}

function refreshH2HCharts() {
  if (!window.gghstatsH2HChartData) return;
  destroyH2HCharts();
  initH2HCharts();
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
        label: uiT('chart.legend_clones_count'),
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
    if (st.scope === 'repo' && st.repo) return uiT('js.syncing_repo', { repo: st.repo });
    return uiT('js.syncing_all');
  }
  if (st.last_error) return uiT('js.sync_last_failed', { error: st.last_error });
  if (st.last_finished_at) {
    const when = new Date(st.last_finished_at).toLocaleString();
    if (st.scope === 'repo' && st.repo) return uiT('js.sync_last_repo', { repo: st.repo, when });
    return uiT('js.sync_last', { when });
  }
  return uiT('js.sync_none_yet');
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
        showError(uiT('js.token_required'));
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
  initH2HCharts();
  initBadgeEmbed();
  initSyncControl();
});

function renderMetrics(canvasId, data, uniqueCol, countCol, chartLabel) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  new Chart(el, {
    type: 'bar',
    data: {
      labels: data.map(d => d.date),
      datasets: [
        {
          label: uiT('chart.legend_unique'),
          data: data.map(d => d[uniqueCol]),
          backgroundColor: c.primary,
          borderWidth: 0,
          borderRadius: 4
        },
        {
          label: uiT('chart.legend_count'),
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
        legend: repoChartLegendOptions(c),
        tooltip: repoChartTooltipOptions(chartLabel, '')
      }
    },
    plugins: [mouseLinePlugin]
  });
}

function renderStars(canvasId, data, chartLabel) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  new Chart(el, {
    type: 'line',
    data: {
      labels: data.map(d => d.date),
      datasets: [{
        label: chartLabel,
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
        legend: repoChartLegendOptions(c),
        tooltip: repoChartTooltipOptions(chartLabel, '')
      }
    },
    plugins: [mouseLinePlugin]
  });
}
