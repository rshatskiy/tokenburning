const token = new URLSearchParams(location.search).get("t") || "";
let period = "30d";
let lastSummary = null;
let sessTool = null;

const $ = (sel) => document.querySelector(sel);
const el = (html) => { const t = document.createElement("template"); t.innerHTML = html.trim(); return t.content.firstChild; };
const esc = (s) => String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
const fmtUSD = (n) => "$" + Math.round(n).toLocaleString("ru-RU");
const fmtTok = (n) => n >= 1e9 ? (n/1e9).toFixed(1)+"B" : n >= 1e6 ? (n/1e6).toFixed(0)+"M" : n.toLocaleString("ru-RU");
const fmtDate = (s) => { const p = String(s).split("-"); return p.length === 3 ? p[2]+"."+p[1] : s; };
// tip-строка уже html-безопасна (имена через esc, теги <br>/<b> литеральные);
// для атрибута data-tip остаётся экранировать только кавычки
const escAttr = (s) => String(s).replace(/"/g, "&quot;");

function renderPeriod() {
  const seg = $("#period"); seg.innerHTML = "";
  for (const p of ["7d","30d","90d","all"]) {
    const s = el(`<span${p===period?' class="on"':''}>${p==="all"?"всё":p}</span>`);
    s.onclick = () => { period = p; load(); };
    seg.appendChild(s);
  }
}

// area-chart по дням
function areaChart(data) {
  if (!data.length) return '<div class="subtitle">нет данных</div>';
  const W=640,H=200, max=Math.max(...data.map(d=>d.cost),1);
  const y=(c)=> H-10 - (c/max)*(H-30);
  let pts;
  if (data.length === 1) { const yy=y(data[0].cost); pts=[[0,yy],[W,yy]]; }
  else { pts = data.map((d,i)=>[i/(data.length-1)*W, y(d.cost)]); }
  const line = "M" + pts.map(p=>`${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(" L");
  const area = line + ` L${W},${H} L0,${H} Z`;
  // прозрачные точки-цели для tooltip по каждому дню
  const xAt = (i) => data.length < 2 ? W/2 : i/(data.length-1)*W;
  const hits = data.map((d,i)=>{
    const cx=xAt(i), cy=y(d.cost);
    const tip = `${fmtDate(d.date)}<br><b>${fmtUSD(d.cost)}</b>`;
    return `<circle cx="${cx.toFixed(1)}" cy="${cy.toFixed(1)}" r="9" fill="transparent" pointer-events="all" data-tip="${tip}"/>`;
  }).join("");
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" preserveAspectRatio="none" style="display:block">
    <defs><linearGradient id="ar" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fb923c" stop-opacity=".38"/><stop offset="1" stop-color="#fb923c" stop-opacity="0"/></linearGradient></defs>
    <path d="${area}" fill="url(#ar)"/>
    <path d="${line}" fill="none" stroke="#fb923c" stroke-width="2.5" stroke-linecap="round" style="filter:drop-shadow(0 0 6px rgba(251,146,60,.55))"/>
    ${hits}
  </svg>`;
}

function bars(items, label, val, max, tip) {
  return items.map(it => {
    const v = val(it);
    const w = Math.max(2, (v/max)*100);
    const cls = v > 0 ? "bar-fill" : "bar-fill dim";
    const dt = tip ? ` data-tip="${escAttr(tip(it))}"` : "";
    return `<div class="bar-row"${dt}><span class="bar-name">${esc(label(it))}</span><span class="bar-track"><span class="${cls}" style="width:${w}%"></span></span><span class="bar-val">${v>0?fmtUSD(v):"~est"}</span></div>`;
  }).join("");
}

function scatter(points) {
  if (!points.length) return '<div class="subtitle">нет данных</div>';
  const W=640,H=210, maxD=Math.max(...points.map(p=>p.durationMin),1), maxC=Math.max(...points.map(p=>p.cost),1);
  const dots = points.map(p=>{
    const cx=20+(p.durationMin/maxD)*(W-40), cy=H-20-(p.cost/maxC)*(H-40);
    const r=p.outlier?9:4, fill=p.outlier?"rgba(251,146,60,.85)":"rgba(255,255,255,.28)";
    const glow=p.outlier?'style="filter:drop-shadow(0 0 8px rgba(251,146,60,.7))"':"";
    const proj = p.project && p.project !== "(нет)" ? esc(p.project.split("/").pop()) : "сессия";
    const tip = `<b>${proj}</b><br>${Math.round(p.durationMin)} мин · ${p.iterations} итер<br>${fmtUSD(p.cost)} · ${fmtTok(p.tokens)} ток`;
    return `<circle cx="${cx.toFixed(0)}" cy="${cy.toFixed(0)}" r="${r}" fill="${fill}" ${glow} data-tip="${escAttr(tip)}"/>`;
  }).join("");
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" style="display:block">
    <line x1="0" y1="${H-20}" x2="${W}" y2="${H-20}" stroke="rgba(255,255,255,.08)"/>
    <line x1="0" y1="0" x2="0" y2="${H-20}" stroke="rgba(255,255,255,.08)"/>
    <text x="6" y="14" fill="#5f5b56" font-size="10" font-family="ui-monospace,monospace">$ за сессию ↑</text>
    <text x="${W-150}" y="${H-6}" fill="#5f5b56" font-size="10" font-family="ui-monospace,monospace">длительность →</text>
    ${dots}</svg>
    <div class="legend"><span><i style="background:rgba(255,255,255,.28)"></i>обычные сессии</span><span><i style="background:#fb923c"></i>долго и дорого — трение</span></div>`;
}

function kpiCard(label, num, sub, accent) {
  return `<div class="shell"><div class="core"><div class="klabel">${accent?'<span class="kdot"></span> ':''}${label}</div><div class="knum">${num}</div><div class="ksub">${sub||""}</div></div></div>`;
}

function render(s) {
  lastSummary = s;
  renderPeriod();
  const app = $("#app"); app.innerHTML = "";
  const k = s.kpis;
  // KPI
  app.appendChild(el(`<div class="kpis">
    ${kpiCard("Стоимость", fmtUSD(k.cost), (k.tokens?Math.round(k.cacheReadTokens/k.tokens*100):0)+"% кэш-чтения", true)}
    ${kpiCard("Токены", fmtTok(k.tokens), "")}
    ${kpiCard("Активных дней", k.activeDays+"", k.sessions+" сессий")}
    ${kpiCard("Инструменты", (k.tools||[]).length+"", esc((k.tools||[]).join(" · ")))}
  </div>`));
  // cost over time + by model
  const maxModel = Math.max(...(s.byModel||[]).map(m=>m.cost),1);
  app.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>Стоимость во времени</h3><span class="meta">USD / день</span></div>${areaChart(s.costOverTime||[])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>По моделям</h3><span class="meta">доля $</span></div>${bars(s.byModel||[], m=>m.model, m=>m.cost, maxModel, m=>`${esc(m.model)}<br><b>${m.cost>0?fmtUSD(m.cost):"оценка ~est"}</b> · ${m.events} событий`)}</div></div>
  </div>`));
  // by tool + top projects + activity
  const maxTool = Math.max(...(s.byTool||[]).map(t=>t.cost),1);
  const proj = (s.topProjects||[]).map(p=>`<div class="proj"><div><div>${esc(p.project)}</div><div class="p-meta">${p.sessions} сессий</div></div><span style="font-family:var(--mono);font-variant-numeric:tabular-nums">${fmtUSD(p.cost)}</span></div>`).join("");
  app.appendChild(el(`<div class="grid3">
    <div class="shell"><div class="core"><div class="ctitle"><h3>По инструментам</h3></div>${bars(s.byTool||[], t=>t.tool, t=>t.cost, maxTool, t=>`${esc(t.tool)}<br><b>${t.cost>0?fmtUSD(t.cost):"оценка ~est"}</b> · ${fmtTok(t.tokens)} ток · ${t.events} соб`)}<div style="margin-top:12px;font-size:11px;color:var(--dim);font-family:var(--mono)">cursor — только активность</div></div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>Топ проектов</h3><span class="meta">$ / задача</span></div>${proj||'<div class="subtitle">нет данных</div>'}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>Активность</h3><span class="meta">сессий/день</span></div><div style="font-size:30px;font-weight:600;font-variant-numeric:tabular-nums">${k.sessions}</div><div class="ksub">сессий за период</div></div></div>
  </div>`));
  // session analytics
  app.appendChild(el(`<div id="sess"></div>`));
  renderSessions(s);
}

function renderSessions(s) {
  const host = $("#sess");
  host.innerHTML = "";
  host.appendChild(el(`<div class="eyebrow">аналитика сессий · <b style="color:var(--acc)">сигнал, не точно</b> · самокоучинг</div>`));
  const tools = s.sessionsByTool || [];
  if (!tools.length) { host.appendChild(el('<div class="subtitle">нет данных по сессиям</div>')); return; }
  if (sessTool === null || !tools.find(t => t.tool === sessTool)) sessTool = tools[0].tool;

  // селектор инструментов
  const seg = el(`<div class="seg" style="margin-bottom:14px;width:max-content"></div>`);
  for (const t of tools) {
    const sp = el(`<span${t.tool === sessTool ? ' class="on"' : ''}>${esc(t.tool)}</span>`);
    sp.onclick = () => { sessTool = t.tool; renderSessions(s); };
    seg.appendChild(sp);
  }
  host.appendChild(seg);

  const cur = tools.find(t => t.tool === sessTool) || tools[0];
  const ss = cur.stats || {};
  const note = cur.tool === "cursor"
    ? '<div style="margin-top:14px;font-size:11px;color:var(--dim);font-family:var(--mono)">Cursor: токены/стоимость за серверным API — локально недоступны</div>'
    : "";
  const flagged = (ss.flagged || []).map(f =>
    `<div class="flag"><div>${esc(f.project)} · ${Math.round(f.durationMin)} мин<div class="p-meta" style="color:var(--dim);font-family:var(--mono);font-size:10px">${f.iterations} итераций · ${fmtUSD(f.cost)}</div></div><span class="tag">застревание?</span></div>`
  ).join("");
  host.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>Сессии: длительность × стоимость</h3><span class="meta">оранжевым — кандидаты на трение</span></div>${scatter(ss.scatter || [])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>На сессию (медиана)</h3><span class="meta">${esc(cur.tool)}</span></div>
      <div class="sess-stats"><div class="sess-stat"><div class="n">${Math.round(ss.medianDurationMin || 0)}<span style="font-size:12px;color:var(--dim)">мин</span></div><div class="l">активная длительность</div><div class="h">p90 ${Math.round(ss.p90DurationMin || 0)}м</div></div>
      <div class="sess-stat"><div class="n">${fmtTok(ss.medianTokens || 0)}</div><div class="l">токенов</div><div class="h">p90 ${fmtTok(ss.p90Tokens || 0)}</div></div></div>
      <div class="sess-stats" style="margin-top:8px"><div class="sess-stat"><div class="n">${Math.round(ss.medianIterations || 0)}</div><div class="l">обращений к модели</div><div class="h">p90 ${Math.round(ss.p90Iterations || 0)}</div></div>
      <div class="sess-stat"><div class="n">${fmtUSD(ss.medianCost || 0)}</div><div class="l">стоимость</div><div class="h">p90 ${fmtUSD(ss.p90Cost || 0)}</div></div></div>
      ${note}
      <div style="margin-top:16px;border-top:1px solid var(--line);padding-top:12px"><div style="font-size:11px;color:var(--muted);margin-bottom:8px">Требуют внимания</div>${flagged || '<div class="subtitle">нет выраженных выбросов</div>'}</div>
    </div></div>
  </div>`));
}

// плавающий tooltip: следует за курсором над любым элементом с атрибутом data-tip
const tip = el('<div class="tooltip"></div>');
document.body.appendChild(tip);
document.addEventListener("mousemove", (e) => {
  const target = e.target.closest ? e.target.closest("[data-tip]") : null;
  if (!target) { tip.style.opacity = "0"; return; }
  tip.innerHTML = target.getAttribute("data-tip");
  tip.style.opacity = "1";
  const r = tip.getBoundingClientRect();
  let x = e.clientX + 14, y = e.clientY + 14;
  if (x + r.width > window.innerWidth - 8) x = e.clientX - r.width - 14;
  if (y + r.height > window.innerHeight - 8) y = e.clientY - r.height - 14;
  tip.style.left = Math.max(8, x) + "px";
  tip.style.top = Math.max(8, y) + "px";
});

async function load() {
  try {
    const r = await fetch(`/api/summary?period=${period}&t=${encodeURIComponent(token)}`);
    if (!r.ok) throw new Error("HTTP "+r.status);
    render(await r.json());
  } catch (e) {
    $("#app").innerHTML = `<p class="subtitle" style="padding:40px 0">Ошибка загрузки: ${e.message}</p>`;
  }
}
load();
