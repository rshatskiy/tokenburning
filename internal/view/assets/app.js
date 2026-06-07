const token = new URLSearchParams(location.search).get("t") || "";
let period = "30d";

const $ = (sel) => document.querySelector(sel);
const el = (html) => { const t = document.createElement("template"); t.innerHTML = html.trim(); return t.content.firstChild; };
const fmtUSD = (n) => "$" + Math.round(n).toLocaleString("ru-RU");
const fmtTok = (n) => n >= 1e9 ? (n/1e9).toFixed(1)+"B" : n >= 1e6 ? (n/1e6).toFixed(0)+"M" : n.toLocaleString("ru-RU");

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
  const x=(i)=> data.length<2?W: i/(data.length-1)*W;
  const y=(c)=> H-10 - (c/max)*(H-30);
  let d=`M${x(0)},${y(data[0].cost)}`;
  for (let i=1;i<data.length;i++) d+=` L${x(i)},${y(data[i].cost)}`;
  const area = d+` L${W},${H} L0,${H} Z`;
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" preserveAspectRatio="none" style="display:block">
    <defs><linearGradient id="ar" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fb923c" stop-opacity=".38"/><stop offset="1" stop-color="#fb923c" stop-opacity="0"/></linearGradient></defs>
    <path d="${area}" fill="url(#ar)"/>
    <path d="${d}" fill="none" stroke="#fb923c" stroke-width="2.5" stroke-linecap="round" style="filter:drop-shadow(0 0 6px rgba(251,146,60,.55))"/>
  </svg>`;
}

function bars(items, label, val, max) {
  return items.map(it => {
    const w = Math.max(2, (val(it)/max)*100);
    return `<div class="bar-row"><span class="bar-name">${label(it)}</span><span class="bar-track"><span class="bar-fill" style="width:${w}%"></span></span><span class="bar-val">${val(it)>0?fmtUSD(val(it)):"~est"}</span></div>`;
  }).join("");
}

function scatter(points) {
  if (!points.length) return '<div class="subtitle">нет данных</div>';
  const W=640,H=210, maxD=Math.max(...points.map(p=>p.durationMin),1), maxC=Math.max(...points.map(p=>p.cost),1);
  const dots = points.map(p=>{
    const cx=20+ (p.durationMin/maxD)*(W-40), cy=H-20-(p.cost/maxC)*(H-40);
    const r = p.outlier?9:4, fill=p.outlier?"rgba(251,146,60,.85)":"rgba(255,255,255,.28)";
    const glow=p.outlier?'style="filter:drop-shadow(0 0 8px rgba(251,146,60,.7))"':"";
    return `<circle cx="${cx.toFixed(0)}" cy="${cy.toFixed(0)}" r="${r}" fill="${fill}" ${glow}/>`;
  }).join("");
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" style="display:block"><line x1="0" y1="${H-20}" x2="${W}" y2="${H-20}" stroke="rgba(255,255,255,.08)"/>${dots}</svg>`;
}

function kpiCard(label, num, sub, accent) {
  return `<div class="shell"><div class="core"><div class="klabel">${accent?'<span class="kdot"></span> ':''}${label}</div><div class="knum">${num}</div><div class="ksub">${sub||""}</div></div></div>`;
}

function render(s) {
  renderPeriod();
  const app = $("#app"); app.innerHTML = "";
  const k = s.kpis;
  // KPI
  app.appendChild(el(`<div class="kpis">
    ${kpiCard("Стоимость", fmtUSD(k.cost), (k.tokens?Math.round(k.cacheReadTokens/k.tokens*100):0)+"% кэш-чтения", true)}
    ${kpiCard("Токены", fmtTok(k.tokens), "")}
    ${kpiCard("Активных дней", k.activeDays+"", k.sessions+" сессий")}
    ${kpiCard("Инструменты", (k.tools||[]).length+"", (k.tools||[]).join(" · "))}
  </div>`));
  // cost over time + by model
  const maxModel = Math.max(...(s.byModel||[]).map(m=>m.cost),1);
  app.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>Стоимость во времени</h3><span class="meta">USD / день</span></div>${areaChart(s.costOverTime||[])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>По моделям</h3><span class="meta">доля $</span></div>${bars(s.byModel||[], m=>m.model, m=>m.cost, maxModel)}</div></div>
  </div>`));
  // by tool + top projects + activity
  const maxTool = Math.max(...(s.byTool||[]).map(t=>t.cost),1);
  const proj = (s.topProjects||[]).map(p=>`<div class="proj"><div><div>${p.project}</div><div class="p-meta">${p.sessions} сессий</div></div><span style="font-family:var(--mono);font-variant-numeric:tabular-nums">${fmtUSD(p.cost)}</span></div>`).join("");
  app.appendChild(el(`<div class="grid3">
    <div class="shell"><div class="core"><div class="ctitle"><h3>По инструментам</h3></div>${bars(s.byTool||[], t=>t.tool, t=>t.cost, maxTool)}<div style="margin-top:12px;font-size:11px;color:var(--dim);font-family:var(--mono)">cursor — только активность</div></div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>Топ проектов</h3><span class="meta">$ / задача</span></div>${proj||'<div class="subtitle">нет данных</div>'}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>Активность</h3><span class="meta">сессий/день</span></div><div style="font-size:30px;font-weight:600;font-variant-numeric:tabular-nums">${k.sessions}</div><div class="ksub">сессий за период</div></div></div>
  </div>`));
  // session analytics
  const ss = s.sessions||{};
  app.appendChild(el(`<div class="eyebrow">аналитика сессий · <b style="color:var(--acc)">сигнал, не точно</b> · самокоучинг</div>`));
  const flagged = (ss.flagged||[]).map(f=>`<div class="flag"><div>${f.project} · ${Math.round(f.durationMin)} мин<div class="p-meta" style="color:var(--dim);font-family:var(--mono);font-size:10px">${f.iterations} итераций · ${fmtUSD(f.cost)}</div></div><span class="tag">застревание?</span></div>`).join("");
  app.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>Сессии: длительность × стоимость</h3><span class="meta">оранжевым — кандидаты на трение</span></div>${scatter(ss.scatter||[])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>На сессию (медиана)</h3></div>
      <div class="sess-stats"><div class="sess-stat"><div class="n">${Math.round(ss.medianDurationMin||0)}<span style="font-size:12px;color:var(--dim)">мин</span></div><div class="l">длительность</div><div class="h">p90 ${Math.round(ss.p90DurationMin||0)}м</div></div>
      <div class="sess-stat"><div class="n">${fmtTok(ss.medianTokens||0)}</div><div class="l">токенов</div><div class="h">p90 ${fmtTok(ss.p90Tokens||0)}</div></div></div>
      <div class="sess-stats" style="margin-top:8px"><div class="sess-stat"><div class="n">${Math.round(ss.medianIterations||0)}</div><div class="l">итераций/задача</div><div class="h">p90 ${Math.round(ss.p90Iterations||0)}</div></div>
      <div class="sess-stat"><div class="n">${fmtUSD(ss.medianCost||0)}</div><div class="l">стоимость</div><div class="h">p90 ${fmtUSD(ss.p90Cost||0)}</div></div></div>
      <div style="margin-top:16px;border-top:1px solid var(--line);padding-top:12px"><div style="font-size:11px;color:var(--muted);margin-bottom:8px">Требуют внимания</div>${flagged||'<div class="subtitle">нет выраженных выбросов</div>'}</div>
    </div></div>
  </div>`));
}

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
