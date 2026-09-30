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
  return theme === 'dark' || theme === 'midnight' ? theme : 'light';
}

function isDarkTheme() {
  return currentTheme() !== 'light';
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
    ctx.strokeStyle = isDarkTheme()
      ? 'rgba(255, 255, 255, 0.35)'
      : 'rgba(100, 149, 237, 0.45)';
    ctx.stroke();
    ctx.restore();
  }
};

function chartThemeColors() {
  const dark = isDarkTheme();
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

function chartTooltipOptions(opts = {}) {
  const dark = isDarkTheme();
  const formatValues = opts.formatValues !== false;
  const asPercent = opts.asPercent === true;
  const base = {
    intersect: false,
    backgroundColor: dark ? '#16161a' : '#ffffff',
    titleColor: dark ? '#fafaf6' : '#121212',
    bodyColor: dark ? '#e8e8e3' : '#2a2a2a',
    borderColor: dark ? 'rgba(242, 242, 237, 0.4)' : '#121212',
    borderWidth: 2,
    padding: 10,
    displayColors: true
  };
  if (!formatValues) {
    return base;
  }
  return {
    ...base,
    callbacks: {
      label(ctx) {
        const label = ctx.dataset?.label ? `${ctx.dataset.label}: ` : '';
        const raw = ctx.parsed?.y;
        if (raw === null || raw === undefined || !Number.isFinite(Number(raw))) {
          return `${label}—`;
        }
        if (asPercent) {
          return `${label}${formatMomentumTick(raw)}`;
        }
        return `${label}${formatChartTick(raw)}`;
      }
    }
  };
}

function applyTheme(theme) {
  if (theme !== 'light' && theme !== 'dark' && theme !== 'midnight') theme = 'light';
  document.documentElement.setAttribute('data-bs-theme', theme);
  localStorage.setItem('gghstats-theme', theme);
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    const nextTheme = theme === 'light' ? 'dark' : theme === 'dark' ? 'midnight' : 'light';
    const labelKey = nextTheme === 'dark' ? 'common.theme_dark' : nextTheme === 'midnight' ? 'common.theme_midnight' : 'common.theme_light';
    const label = uiT(labelKey);
    const labelEl = document.getElementById('theme-toggle-label');
    if (labelEl) labelEl.textContent = label;
    btn.setAttribute('aria-label', label);
    btn.dataset.tooltip = label;
    btn.dataset.themeAction = nextTheme;
    const icon = btn.querySelector('.app-theme-icon');
    if (icon) icon.textContent = nextTheme === 'light' ? '☀' : nextTheme === 'midnight' ? '◉' : '☾';
  }
}

function closeMobileSidebar() {
  if (window.innerWidth >= 992 || !window.bootstrap?.Offcanvas) return;
  const sidebar = document.getElementById('brutalSidebar');
  if (!sidebar) return;
  const instance = window.bootstrap.Offcanvas.getInstance(sidebar);
  if (instance) instance.hide();
}

function initMobileSidebarClose() {
  const sidebar = document.getElementById('brutalSidebar');
  const closeButton = sidebar?.querySelector('[data-gghstats-role="sidebar-close"]');
  if (!sidebar || !closeButton) return;

  closeButton.addEventListener('click', () => {
    if (window.innerWidth >= 992 || !window.bootstrap?.Offcanvas) return;
    window.bootstrap.Offcanvas.getOrCreateInstance(sidebar).hide();
  });
}

function toggleTheme() {
  const theme = currentTheme();
  const nextTheme = theme === 'light' ? 'dark' : theme === 'dark' ? 'midnight' : 'light';
  applyTheme(nextTheme);
  closeMobileSidebar();
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

function initSidebarToggle() {
  const btn = document.getElementById('sidebar-toggle');
  if (!btn) return;

  const update = () => {
    const collapsed = document.documentElement.classList.contains('sidebar-collapsed');
    btn.setAttribute('aria-expanded', String(!collapsed));
    const key = collapsed ? 'nav.expand_nav' : 'nav.collapse_nav';
    const label = uiT(key);
    btn.setAttribute('aria-label', label);
    btn.dataset.tooltip = label;
    const icon = btn.querySelector('span');
    if (icon) icon.textContent = collapsed ? '›' : '‹';
  };

  btn.addEventListener('click', () => {
    const collapsed = document.documentElement.classList.toggle('sidebar-collapsed');
    localStorage.setItem('gghstats-sidebar-collapsed', String(collapsed));
    update();
  });
  update();
}

function initLanguageSelect() {
  const select = document.getElementById('language-select');
  if (!select) return;
  select.addEventListener('change', () => {
    if (select.value) window.location.assign(select.value);
  });
}

function initCollapsiblePanels() {
  const toggles = document.querySelectorAll('.app-panel-toggle');
  for (const toggle of toggles) {
    const target = document.querySelector(toggle.dataset.bsTarget);
    const icon = toggle.querySelector('span');
    if (!target || !icon) continue;
    const update = () => {
      const expanded = toggle.getAttribute('aria-expanded') === 'true';
      icon.textContent = expanded ? '⌃' : '⌄';
    };
    target.addEventListener('shown.bs.collapse', update);
    target.addEventListener('hidden.bs.collapse', update);
    update();
  }
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
    renderStars('chart_stars', payload.stars, starsTitle, payload.starsKPI);
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

function numberFormatConfig() {
  const cfg = window.gghstatsNumberFormat || {};
  return {
    locale: String(cfg.locale || 'en'),
    compact: cfg.compact === true
  };
}

function localeGroupSep(locale) {
  switch (locale) {
    case 'es':
    case 'de':
    case 'pt-br':
      return '.';
    case 'fr':
      return ' ';
    default:
      return ',';
  }
}

function localeDecSep(locale) {
  switch (locale) {
    case 'es':
    case 'de':
    case 'fr':
    case 'pt-br':
      return ',';
    default:
      return '.';
  }
}

function groupedCountJS(n, locale) {
  const s = String(Math.trunc(Math.abs(n)));
  const sep = localeGroupSep(locale);
  let out = '';
  let start = s.length % 3;
  if (start === 0) start = 3;
  out = s.slice(0, start);
  for (let i = start; i < s.length; i += 3) {
    out += sep + s.slice(i, i + 3);
  }
  return out;
}

function compactCountJS(n, locale) {
  const abs = Math.abs(n);
  if (abs < 1000) return String(Math.round(n));
  const dec = localeDecSep(locale);
  const units = [
    [1_000_000_000_000_000, 'P'],
    [1_000_000_000_000, 'T'],
    [1_000_000_000, 'G'],
    [1_000_000, 'M'],
    [1_000, 'k']
  ];
  for (const [div, suf] of units) {
    if (abs >= div) {
      let s = (n / div).toFixed(1);
      if (dec !== '.') s = s.replace('.', dec);
      return s + suf;
    }
  }
  return String(Math.round(n));
}

/** Mirrors i18n.FormatCount for Chart.js ticks/tooltips. */
function formatChartTick(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '';
  const { locale, compact } = numberFormatConfig();
  // Match i18n.FormatCount: clamp negatives to 0, then group or compact.
  let count = Math.round(n);
  if (count < 0) count = 0;
  if (compact) {
    return compactCountJS(count, locale);
  }
  return groupedCountJS(count, locale);
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
        tooltip: chartTooltipOptions({ asPercent })
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
      datasets: [
        {
          label: uiT('chart.legend_clones_count'),
          data: data.map(d => d.count),
          borderColor: c.primary,
          backgroundColor: 'transparent',
          pointStyle: false,
          tension: 0.1,
          borderWidth: 2
        },
        {
          label: uiT('chart.legend_unique'),
          data: data.map(d => d.uniques),
          borderColor: c.info,
          backgroundColor: 'transparent',
          pointStyle: false,
          tension: 0.1,
          borderWidth: 2
        }
      ]
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
            font: { size: 11, family: "'JetBrains Mono', monospace" },
            callback: formatChartTick
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        }
      },
      plugins: {
        legend: {
          display: true,
          position: 'bottom',
          labels: {
            color: c.fg,
            boxWidth: 12,
            padding: 14,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          }
        },
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

function initCloneStatisticsSelector() {
  const buttons = document.querySelectorAll('[data-gghstats-stats-selector]');
  const panels = document.querySelectorAll('[data-gghstats-stats-panel]');
  if (buttons.length === 0 || panels.length === 0) return;

  for (const button of buttons) {
    button.addEventListener('click', () => {
      const selected = button.dataset.gghstatsStatsSelector;
      for (const panel of panels) {
        panel.hidden = panel.dataset.gghstatsStatsPanel !== selected;
      }
      for (const candidate of buttons) {
        const active = candidate === button;
        candidate.classList.toggle('active', active);
        candidate.setAttribute('aria-pressed', String(active));
      }
    });
  }
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
function requestSyncTokenModal({ invalid = false, purpose = 'sync' } = {}) {
  const modalEl = document.getElementById('sync-token-modal');
  if (!modalEl || typeof bootstrap === 'undefined') {
    return Promise.resolve(null);
  }

  const input = document.getElementById('sync-token-input');
  const errorEl = document.getElementById('sync-token-error');
  const submitBtn = document.getElementById('sync-token-submit');
  if (!input || !errorEl || !submitBtn) return Promise.resolve(null);

  submitBtn.textContent = uiT(
    purpose === 'settings' ? 'js.token_continue' : 'js.token_save_sync'
  );

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

async function establishSettingsSession(token) {
  if (!token) return false;
  try {
    const res = await fetch('/api/v1/settings/session', {
      method: 'POST',
      headers: { 'x-api-token': token }
    });
    return res.ok;
  } catch {
    return false;
  }
}

async function obtainSettingsSession() {
  let token = syncApiToken();
  if (token && await establishSettingsSession(token)) {
    return token;
  }
  if (token) sessionStorage.removeItem(SYNC_TOKEN_KEY);

  token = await requestSyncTokenModal({ purpose: 'settings' });
  if (!token) return null;
  if (await establishSettingsSession(token)) {
    sessionStorage.setItem(SYNC_TOKEN_KEY, token);
    return token;
  }

  const retry = await requestSyncTokenModal({ invalid: true, purpose: 'settings' });
  if (!retry || !(await establishSettingsSession(retry))) return null;
  sessionStorage.setItem(SYNC_TOKEN_KEY, retry);
  return retry;
}

function initSettingsAccess() {
  const links = document.querySelectorAll('[data-gghstats-settings-link][data-settings-auth="token"]');
  const start = document.querySelector('[data-settings-auth-start]');
  const openSettings = async (event, target) => {
    event?.preventDefault();
    if (await obtainSettingsSession()) {
      window.location.assign(target);
    }
  };

  links.forEach(link => {
    link.addEventListener('click', event => openSettings(event, link.href));
  });
  if (start) {
    start.addEventListener('click', event => openSettings(event, window.location.href));
  }
}

function initSyncControl() {
  const btn = document.getElementById('sync-now-btn');
  const statusEl = document.getElementById('sync-status');
  if (!btn || !statusEl) return;

  let pollTimer = null;
  let wasRunning = false;
  const pageRepo = syncScopeRepo();
  const baseTooltip = btn.dataset.tooltip || btn.getAttribute('aria-label') || '';
  const setSyncStatus = (message) => {
    statusEl.textContent = message;
    btn.dataset.tooltip = message ? `${baseTooltip}\n${message}` : baseTooltip;
  };

  const refreshStatus = async () => {
    try {
      const st = await fetchSyncStatus();
      if (st) setSyncStatus(formatSyncStatus(st));
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
      setSyncStatus('Could not load sync status');
    }
  };

  const runSync = async (token) => {
    btn.disabled = true;
    setSyncStatus(pageRepo ? `Starting sync for ${pageRepo}…` : 'Starting sync…');
    try {
      const res = await fetch(syncPostURL(), {
        method: 'POST',
        headers: { 'x-api-token': token }
      });
      if (res.status === 401) {
        sessionStorage.removeItem(SYNC_TOKEN_KEY);
        setSyncStatus('Invalid API token');
        btn.disabled = false;
        const retry = await requestSyncTokenModal({ invalid: true });
        if (retry) {
          sessionStorage.setItem(SYNC_TOKEN_KEY, retry);
          await runSync(retry);
        }
        return;
      }
      if (res.status === 404) {
        setSyncStatus('Sync API disabled (set GGHSTATS_API_TOKEN)');
        btn.disabled = false;
        return;
      }
      if (res.status === 403) {
        const body = await res.json().catch(() => ({}));
        setSyncStatus(
          body.error === 'ip_not_whitelisted'
            ? uiT('js.sync_ip_not_whitelisted')
            : uiT('js.sync_failed')
        );
        btn.disabled = false;
        return;
      }
      if (res.status === 429) {
        setSyncStatus(uiT('js.sync_rate_limited'));
        btn.disabled = false;
        return;
      }
      if (res.status === 409) {
        setSyncStatus(uiT('js.sync_already_running'));
      } else if (!res.ok) {
        setSyncStatus(uiT('js.sync_start_failed'));
      }
      await refreshStatus();
    } catch {
      setSyncStatus('Could not start sync');
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

function filenameFromContentDisposition(header, fallback) {
  if (!header) return fallback;
  const m = /filename="([^"]+)"/i.exec(header);
  return m ? m[1] : fallback;
}

function initTrafficJSONDownload() {
  const link = document.querySelector('[data-gghstats-role="traffic-json-download"]');
  if (!link) return;
  const url = link.getAttribute('data-traffic-json-url') || link.getAttribute('href');
  if (!url) return;
  const requiresToken = link.getAttribute('data-requires-api-token') === '1';

  link.addEventListener('click', async (ev) => {
    if (!requiresToken) return; // plain navigation / browser download
    ev.preventDefault();
    let token = syncApiToken();
    if (!token) {
      token = await requestSyncTokenModal();
      if (!token) return;
      sessionStorage.setItem(SYNC_TOKEN_KEY, token);
    }
    const tryDownload = async (tok) => {
      const res = await fetch(url, { headers: { 'x-api-token': tok } });
      if (res.status === 401) {
        sessionStorage.removeItem(SYNC_TOKEN_KEY);
        const retry = await requestSyncTokenModal({ invalid: true });
        if (!retry) return;
        sessionStorage.setItem(SYNC_TOKEN_KEY, retry);
        await tryDownload(retry);
        return;
      }
      if (!res.ok) return;
      const blob = await res.blob();
      const name = filenameFromContentDisposition(
        res.headers.get('Content-Disposition'),
        'gghstats-traffic.json'
      );
      const objectURL = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = objectURL;
      a.download = name;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(objectURL);
    };
    try {
      await tryDownload(token);
    } catch {
      /* ignore network errors for download */
    }
  });
}

function initSettingsForm() {
  const form = document.getElementById('settings-form');
  const statusEl = document.getElementById('settings-save-status');
  if (!form || !statusEl) return;

  form.addEventListener('submit', async event => {
    event.preventDefault();
    const submit = form.querySelector('button[type="submit"]');
    const select = document.getElementById('settings-default-locale');
    const compact = document.getElementById('settings-compact-numbers');
    if (!select || !compact) return;

    const localOnly = form.dataset.localOnly === 'true';
    let token = localOnly ? '' : syncApiToken();

    if (submit) submit.disabled = true;
    statusEl.textContent = '';
    try {
      const save = () => fetch('/api/v1/settings', {
        method: 'POST',
        headers: Object.assign({ 'content-type': 'application/json' }, token ? { 'x-api-token': token } : {}),
        body: JSON.stringify({
          default_locale: select.value,
          compact_numbers: compact.checked
        })
      });
      let res = await save();
      if (res.status === 401 && !localOnly) {
        token = await obtainSettingsSession();
        if (token) res = await save();
      }
      if (res.status === 401) {
        sessionStorage.removeItem(SYNC_TOKEN_KEY);
        statusEl.textContent = uiT('js.settings_invalid_token');
        return;
      }
      if (!res.ok) {
        statusEl.textContent = uiT('js.settings_save_failed');
        return;
      }
      statusEl.textContent = uiT('js.settings_saved');
      document.cookie = `gghstats_locale=${encodeURIComponent(select.value)}; path=/; max-age=31536000; samesite=lax`;
      window.setTimeout(() => window.location.reload(), 350);
    } catch {
      statusEl.textContent = uiT('js.settings_save_failed');
    } finally {
      if (submit) submit.disabled = false;
    }
  });
}

function initInitialSyncRefresh() {
  const state = document.getElementById('initial-sync-state');
  if (!state || state.dataset.syncRunning !== 'true') return;

  const refresh = () => {
    if (document.visibilityState === 'visible') window.location.reload();
  };
  window.setTimeout(refresh, 3000);
}

document.addEventListener('DOMContentLoaded', () => {
  initThemeToggle();
  initMobileSidebarClose();
  initSidebarToggle();
  initLanguageSelect();
  initCollapsiblePanels();
  initRepoCharts();
  initIndexListCharts();
  initCloneStatisticsSelector();
  initH2HCharts();
  initBadgeEmbed();
  initSyncControl();
  initTrafficJSONDownload();
  initSettingsForm();
  initSettingsAccess();
  initInitialSyncRefresh();
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
          borderRadius: 4,
          // JSON null denotes an unreported/unknown UTC day; Chart.js leaves a gap.
          spanGaps: false
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
            font: { size: 11, family: "'JetBrains Mono', monospace" },
            callback: formatChartTick
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

function renderStars(canvasId, data, chartLabel, starsKPI) {
  const el = document.getElementById(canvasId);
  if (!el || !data || data.length === 0) return;

  const c = chartThemeColors();
  const points = alignStarsTimeSeries(data, starsKPI);
  new Chart(el, {
    type: 'line',
    data: {
      datasets: [{
        label: chartLabel,
        data: points,
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
          type: 'time',
          time: {
            unit: 'day',
            tooltipFormat: 'yyyy-MM-dd',
            displayFormats: { day: 'yyyy-MM-dd' }
          },
          ticks: {
            color: c.fg,
            maxRotation: 45,
            autoSkip: true,
            maxTicksLimit: 14,
            font: { size: 11, family: "'JetBrains Mono', monospace" }
          },
          grid: { color: c.grid },
          border: { display: true, color: c.border, width: 1 }
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: c.fg,
            precision: 0,
            font: { size: 11, family: "'JetBrains Mono', monospace" },
            callback: formatChartTick
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

/** Sparse star rows → {x,y} time points; pad to today UTC with Stars KPI when needed. */
function alignStarsTimeSeries(data, starsKPI) {
  const points = (data || [])
    .filter(d => d && d.date)
    .map(d => ({ x: d.date, y: Number(d.total) || 0 }));
  if (points.length === 0) return points;

  const today = new Date().toISOString().slice(0, 10);
  const kpi = Number(starsKPI);
  const last = points[points.length - 1];
  const y = Number.isFinite(kpi) && kpi > last.y ? kpi : last.y;
  if (last.x === today) {
    last.y = y;
    return points;
  }
  points.push({ x: today, y });
  return points;
}
