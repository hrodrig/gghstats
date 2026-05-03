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

document.addEventListener('DOMContentLoaded', () => {
  initThemeToggle();
  initRepoCharts();
  initIndexListCharts();
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
