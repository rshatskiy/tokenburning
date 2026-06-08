const I18N = {
  en: {
    all: "all", nodata: "no data", none: "(none)", session: "session",
    cost: "Cost", cacheRead: "% cache read", tokens: "Tokens", activeDays: "Active days",
    sessionsSuffix: " sessions", tools: "Tools",
    costOverTime: "Cost over time", usdPerDay: "USD / day",
    byModel: "By model", shareUsd: "$ share", events: "events",
    byTool: "By tool", tokShort: "tok", evShort: "ev", cursorActivityOnly: "cursor — activity only",
    cursorNote: "Cursor: tokens/cost are behind the server API — unavailable locally",
    topProjects: "Top projects", perTask: "$ / task",
    activity: "Activity", sessionsPerDay: "sessions/day", sessionsInPeriod: "sessions in period",
    sessionAnalytics: "session analytics", signalNotExact: "signal, not exact", selfCoaching: "self-coaching",
    noSessionData: "no session data",
    perSessionUsdUp: "$ per session ↑", durationRight: "duration →",
    normalSessions: "normal sessions", longExpensiveFriction: "long & expensive — friction",
    sessDurCost: "Sessions: duration × cost", orangeCandidates: "orange — friction candidates",
    perSessionMedian: "Per session (median)", min: "min", activeDuration: "active duration",
    modelCalls: "model calls", needsAttention: "Needs attention", noClearOutliers: "no clear outliers",
    stuck: "stuck?", iter: "iter", iterations: "iterations", loadError: "Load error",
    tokensLabel: "tokens", costLabel: "cost",
    share: "Share ↗", shareTitle: "Share your stats", dl: "Download", copyImg: "Copy image", copied: "Copied!", postX: "Post on X",
    cardHeadline: "My AI coding spend", periodAll: "all time", periodPrefix: "last ", periodDays: " days",
    trackYours: "track yours →", shareHint: "Download or copy the card, then attach it to your post.",
    tweet: "I spent {cost} on AI coding tools ({period}) 🔥 tracked locally with tokenburning",
  },
  ru: {
    all: "всё", nodata: "нет данных", none: "(нет)", session: "сессия",
    cost: "Стоимость", cacheRead: "% кэш-чтения", tokens: "Токены", activeDays: "Активных дней",
    sessionsSuffix: " сессий", tools: "Инструменты",
    costOverTime: "Стоимость во времени", usdPerDay: "USD / день",
    byModel: "По моделям", shareUsd: "доля $", events: "событий",
    byTool: "По инструментам", tokShort: "ток", evShort: "соб", cursorActivityOnly: "cursor — только активность",
    cursorNote: "Cursor: токены/стоимость за серверным API — локально недоступны",
    topProjects: "Топ проектов", perTask: "$ / задача",
    activity: "Активность", sessionsPerDay: "сессий/день", sessionsInPeriod: "сессий за период",
    sessionAnalytics: "аналитика сессий", signalNotExact: "сигнал, не точно", selfCoaching: "самокоучинг",
    noSessionData: "нет данных по сессиям",
    perSessionUsdUp: "$ за сессию ↑", durationRight: "длительность →",
    normalSessions: "обычные сессии", longExpensiveFriction: "долго и дорого — трение",
    sessDurCost: "Сессии: длительность × стоимость", orangeCandidates: "оранжевым — кандидаты на трение",
    perSessionMedian: "На сессию (медиана)", min: "мин", activeDuration: "активная длительность",
    modelCalls: "обращений к модели", needsAttention: "Требуют внимания", noClearOutliers: "нет выраженных выбросов",
    stuck: "застревание?", iter: "итер", iterations: "итераций", loadError: "Ошибка загрузки",
    tokensLabel: "токенов", costLabel: "стоимость",
    share: "Поделиться ↗", shareTitle: "Поделиться статистикой", dl: "Скачать", copyImg: "Копировать", copied: "Скопировано!", postX: "В X",
    cardHeadline: "Мои траты на ИИ-код", periodAll: "всё время", periodPrefix: "последние ", periodDays: " дн.",
    trackYours: "посчитай свои →", shareHint: "Скачайте или скопируйте карточку и приложите к посту.",
    tweet: "Потратил {cost} на ИИ-инструменты для кода ({period}) 🔥 считаю локально через tokenburning",
  },
};
let lang = localStorage.getItem("tb_lang");
if (lang !== "en" && lang !== "ru") {
  lang = (navigator.language || "en").toLowerCase().startsWith("ru") ? "ru" : "en";
}
const t = (k) => (I18N[lang] && I18N[lang][k] != null) ? I18N[lang][k] : (I18N.en[k] != null ? I18N.en[k] : k);
const locale = () => (lang === "ru" ? "ru-RU" : "en-US");

const token = new URLSearchParams(location.search).get("t") || "";
let period = "30d";
let lastSummary = null;
let sessTool = null;

const $ = (sel) => document.querySelector(sel);
const el = (html) => { const tmpl = document.createElement("template"); tmpl.innerHTML = html.trim(); return tmpl.content.firstChild; };
const esc = (s) => String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
const fmtUSD = (n) => "$" + Math.round(n).toLocaleString(locale());
const fmtTok = (n) => n >= 1e9 ? (n/1e9).toFixed(1)+"B" : n >= 1e6 ? (n/1e6).toFixed(0)+"M" : n.toLocaleString(locale());
const fmtDate = (s) => { const p = String(s).split("-"); return p.length === 3 ? p[2]+"."+p[1] : s; };
// tip-string is already html-safe (names via esc, tags <br>/<b> are literal);
// for data-tip attribute only quotes need escaping
const escAttr = (s) => String(s).replace(/"/g, "&quot;");

function renderLang() {
  const seg = $("#lang"); if (!seg) return; seg.innerHTML = "";
  for (const l of ["EN","RU"]) {
    const code = l.toLowerCase();
    const s = el(`<span${code===lang?' class="on"':''}>${l}</span>`);
    s.onclick = () => { lang = code; localStorage.setItem("tb_lang", code); rerender(); };
    seg.appendChild(s);
  }
}

function rerender() {
  renderLang(); renderPeriod(); wireShare();
  if (lastSummary) render(lastSummary);
}

function renderPeriod() {
  const seg = $("#period"); seg.innerHTML = "";
  for (const p of ["7d","30d","90d","all"]) {
    const s = el(`<span${p===period?' class="on"':''}>${p==="all"?t('all'):p}</span>`);
    s.onclick = () => { period = p; load(); };
    seg.appendChild(s);
  }
}

// area chart by day
function areaChart(data) {
  if (!data.length) return `<div class="subtitle">${t('nodata')}</div>`;
  const W=640,H=200, max=Math.max(...data.map(d=>d.cost),1);
  const y=(c)=> H-10 - (c/max)*(H-30);
  let pts;
  if (data.length === 1) { const yy=y(data[0].cost); pts=[[0,yy],[W,yy]]; }
  else { pts = data.map((d,i)=>[i/(data.length-1)*W, y(d.cost)]); }
  const line = "M" + pts.map(p=>`${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(" L");
  const area = line + ` L${W},${H} L0,${H} Z`;
  // transparent hit targets for tooltip per day
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
  if (!points.length) return `<div class="subtitle">${t('nodata')}</div>`;
  const W=640,H=210, maxD=Math.max(...points.map(p=>p.durationMin),1), maxC=Math.max(...points.map(p=>p.cost),1);
  const dots = points.map(p=>{
    const cx=20+(p.durationMin/maxD)*(W-40), cy=H-20-(p.cost/maxC)*(H-40);
    const r=p.outlier?9:4, fill=p.outlier?"rgba(251,146,60,.85)":"rgba(255,255,255,.28)";
    const glow=p.outlier?'style="filter:drop-shadow(0 0 8px rgba(251,146,60,.7))"':"";
    const proj = p.project && p.project !== t('none') ? esc(p.project.split("/").pop()) : t('session');
    const tip = `<b>${proj}</b><br>${Math.round(p.durationMin)} ${t('min')} · ${p.iterations} ${t('iter')}<br>${fmtUSD(p.cost)} · ${fmtTok(p.tokens)} ${t('tokShort')}`;
    return `<circle cx="${cx.toFixed(0)}" cy="${cy.toFixed(0)}" r="${r}" fill="${fill}" ${glow} data-tip="${escAttr(tip)}"/>`;
  }).join("");
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" style="display:block">
    <line x1="0" y1="${H-20}" x2="${W}" y2="${H-20}" stroke="rgba(255,255,255,.08)"/>
    <line x1="0" y1="0" x2="0" y2="${H-20}" stroke="rgba(255,255,255,.08)"/>
    <text x="6" y="14" fill="#5f5b56" font-size="10" font-family="ui-monospace,monospace">${t('perSessionUsdUp')}</text>
    <text x="${W-150}" y="${H-6}" fill="#5f5b56" font-size="10" font-family="ui-monospace,monospace">${t('durationRight')}</text>
    ${dots}</svg>
    <div class="legend"><span><i style="background:rgba(255,255,255,.28)"></i>${t('normalSessions')}</span><span><i style="background:#fb923c"></i>${t('longExpensiveFriction')}</span></div>`;
}

function kpiCard(label, num, sub, accent) {
  return `<div class="shell"><div class="core"><div class="klabel">${accent?'<span class="kdot"></span> ':''}${label}</div><div class="knum">${num}</div><div class="ksub">${sub||""}</div></div></div>`;
}

function render(s) {
  lastSummary = s;
  renderLang();
  renderPeriod();
  const app = $("#app"); app.innerHTML = "";
  const k = s.kpis;
  // KPI
  app.appendChild(el(`<div class="kpis">
    ${kpiCard(t('cost'), fmtUSD(k.cost), (k.tokens?Math.round(k.cacheReadTokens/k.tokens*100):0)+t('cacheRead'), true)}
    ${kpiCard(t('tokens'), fmtTok(k.tokens), "")}
    ${kpiCard(t('activeDays'), k.activeDays+"", k.sessions+t('sessionsSuffix'))}
    ${kpiCard(t('tools'), (k.tools||[]).length+"", esc((k.tools||[]).join(" · ")))}
  </div>`));
  // cost over time + by model
  const maxModel = Math.max(...(s.byModel||[]).map(m=>m.cost),1);
  app.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('costOverTime')}</h3><span class="meta">${t('usdPerDay')}</span></div>${areaChart(s.costOverTime||[])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('byModel')}</h3><span class="meta">${t('shareUsd')}</span></div>${bars(s.byModel||[], m=>m.model, m=>m.cost, maxModel, m=>`${esc(m.model)}<br><b>${m.cost>0?fmtUSD(m.cost):"~est"}</b> · ${m.events} ${t('events')}`)}</div></div>
  </div>`));
  // by tool + top projects + activity
  const maxTool = Math.max(...(s.byTool||[]).map(tool=>tool.cost),1);
  const proj = (s.topProjects||[]).map(p=>`<div class="proj"><div><div>${esc(p.project)}</div><div class="p-meta">${p.sessions}${t('sessionsSuffix')}</div></div><span style="font-family:var(--mono);font-variant-numeric:tabular-nums">${fmtUSD(p.cost)}</span></div>`).join("");
  app.appendChild(el(`<div class="grid3">
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('byTool')}</h3></div>${bars(s.byTool||[], tool=>tool.tool, tool=>tool.cost, maxTool, tool=>`${esc(tool.tool)}<br><b>${tool.cost>0?fmtUSD(tool.cost):"~est"}</b> · ${fmtTok(tool.tokens)} ${t('tokShort')} · ${tool.events} ${t('evShort')}`)}<div style="margin-top:12px;font-size:11px;color:var(--dim);font-family:var(--mono)">${t('cursorActivityOnly')}</div></div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('topProjects')}</h3><span class="meta">${t('perTask')}</span></div>${proj||`<div class="subtitle">${t('nodata')}</div>`}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('activity')}</h3><span class="meta">${t('sessionsPerDay')}</span></div><div style="font-size:30px;font-weight:600;font-variant-numeric:tabular-nums">${k.sessions}</div><div class="ksub">${t('sessionsInPeriod')}</div></div></div>
  </div>`));
  // session analytics
  app.appendChild(el(`<div id="sess"></div>`));
  renderSessions(s);
}

function renderSessions(s) {
  const host = $("#sess");
  host.innerHTML = "";
  host.appendChild(el(`<div class="eyebrow">${t('sessionAnalytics')} · <b style="color:var(--acc)">${t('signalNotExact')}</b> · ${t('selfCoaching')}</div>`));
  const tools = s.sessionsByTool || [];
  if (!tools.length) { host.appendChild(el(`<div class="subtitle">${t('noSessionData')}</div>`)); return; }
  if (sessTool === null || !tools.find(tool => tool.tool === sessTool)) sessTool = tools[0].tool;

  // tool selector
  const seg = el(`<div class="seg" style="margin-bottom:14px;width:max-content"></div>`);
  for (const tool of tools) {
    const sp = el(`<span${tool.tool === sessTool ? ' class="on"' : ''}>${esc(tool.tool)}</span>`);
    sp.onclick = () => { sessTool = tool.tool; renderSessions(s); };
    seg.appendChild(sp);
  }
  host.appendChild(seg);

  const cur = tools.find(tool => tool.tool === sessTool) || tools[0];
  const ss = cur.stats || {};
  const note = cur.tool === "cursor"
    ? `<div style="margin-top:14px;font-size:11px;color:var(--dim);font-family:var(--mono)">${t('cursorNote')}</div>`
    : "";
  const flagged = (ss.flagged || []).map(f =>
    `<div class="flag"><div>${esc(f.project)} · ${Math.round(f.durationMin)} ${t('min')}<div class="p-meta" style="color:var(--dim);font-family:var(--mono);font-size:10px">${f.iterations} ${t('iterations')} · ${fmtUSD(f.cost)}</div></div><span class="tag">${t('stuck')}</span></div>`
  ).join("");
  host.appendChild(el(`<div class="grid2">
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('sessDurCost')}</h3><span class="meta">${t('orangeCandidates')}</span></div>${scatter(ss.scatter || [])}</div></div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('perSessionMedian')}</h3><span class="meta">${esc(cur.tool)}</span></div>
      <div class="sess-stats"><div class="sess-stat"><div class="n">${Math.round(ss.medianDurationMin || 0)}<span style="font-size:12px;color:var(--dim)">${t('min')}</span></div><div class="l">${t('activeDuration')}</div><div class="h">p90 ${Math.round(ss.p90DurationMin || 0)}m</div></div>
      <div class="sess-stat"><div class="n">${fmtTok(ss.medianTokens || 0)}</div><div class="l">${t('tokensLabel')}</div><div class="h">p90 ${fmtTok(ss.p90Tokens || 0)}</div></div></div>
      <div class="sess-stats" style="margin-top:8px"><div class="sess-stat"><div class="n">${Math.round(ss.medianIterations || 0)}</div><div class="l">${t('modelCalls')}</div><div class="h">p90 ${Math.round(ss.p90Iterations || 0)}</div></div>
      <div class="sess-stat"><div class="n">${fmtUSD(ss.medianCost || 0)}</div><div class="l">${t('costLabel')}</div><div class="h">p90 ${fmtUSD(ss.p90Cost || 0)}</div></div></div>
      ${note}
      <div style="margin-top:16px;border-top:1px solid var(--line);padding-top:12px"><div style="font-size:11px;color:var(--muted);margin-bottom:8px">${t('needsAttention')}</div>${flagged || `<div class="subtitle">${t('noClearOutliers')}</div>`}</div>
    </div></div>
  </div>`));
}

// floating tooltip: follows cursor over any element with data-tip attribute
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

// ---- Share card (client-side; only safe aggregates — no project paths/sessions) ----
function periodLabel() {
  return period === "all" ? t('periodAll') : t('periodPrefix') + period.replace('d','') + t('periodDays');
}
function rrect(c,x,y,w,h,r){c.beginPath();c.moveTo(x+r,y);c.arcTo(x+w,y,x+w,y+h,r);c.arcTo(x+w,y+h,x,y+h,r);c.arcTo(x,y+h,x,y,r);c.arcTo(x,y,x+w,y,r);c.closePath();}
function drawShareCard(canvas) {
  const W=1200,H=630,dpr=2,F="-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif";
  canvas.width=W*dpr; canvas.height=H*dpr;
  const c=canvas.getContext('2d'); c.scale(dpr,dpr);
  const k=(lastSummary&&lastSummary.kpis)||{cost:0,tokens:0,activeDays:0,tools:[]};
  c.fillStyle="#0a0a0c"; c.fillRect(0,0,W,H);
  const g=c.createRadialGradient(W-150,70,30,W-150,70,640);
  g.addColorStop(0,"rgba(251,146,60,.22)"); g.addColorStop(1,"rgba(251,146,60,0)");
  c.fillStyle=g; c.fillRect(0,0,W,H);
  const P=72;
  const lg=c.createLinearGradient(P,P,P+46,P+46); lg.addColorStop(0,"#ea580c"); lg.addColorStop(1,"#fb923c");
  c.fillStyle=lg; rrect(c,P,P,46,46,13); c.fill();
  c.fillStyle="#1a0d04"; c.textBaseline="middle"; c.textAlign="center"; c.font="700 23px "+F; c.fillText("tb",P+23,P+24);
  c.textAlign="left"; c.fillStyle="#fafafa"; c.font="600 25px "+F; c.fillText("tokenburning",P+60,P+24);
  c.textBaseline="alphabetic";
  c.fillStyle="#8a857d"; c.font="600 20px "+F; c.fillText((t('cardHeadline')+"  ·  "+periodLabel()).toUpperCase(), P, 252);
  c.fillStyle="#fb923c"; c.font="800 124px "+F; c.fillText(fmtUSD(k.cost), P, 372);
  c.fillStyle="#d6d3d1"; c.font="500 30px "+F;
  c.fillText(fmtTok(k.tokens)+" "+t('tokensLabel')+"   ·   "+k.activeDays+" "+t('activeDays').toLowerCase(), P, 430);
  let x=P; const ty=472; c.font="500 22px "+F; c.textBaseline="middle";
  for (const tool of (k.tools||[])) {
    const cw=c.measureText(tool).width+36;
    c.fillStyle="rgba(255,255,255,.06)"; rrect(c,x,ty,cw,44,22); c.fill();
    c.strokeStyle="rgba(255,255,255,.13)"; c.lineWidth=1; rrect(c,x+.5,ty+.5,cw-1,43,21.5); c.stroke();
    c.fillStyle="#e7e5e4"; c.fillText(tool, x+18, ty+23);
    x+=cw+12;
  }
  c.textBaseline="alphabetic";
  c.fillStyle="#71717a"; c.font="500 22px "+F; c.fillText("tokenburning.online", P, H-P+6);
  c.fillStyle="#fb923c"; c.textAlign="right"; c.fillText(t('trackYours'), W-P, H-P+6); c.textAlign="left";
}
function openShareModal() {
  const bg=el(`<div class="modal-bg"></div>`);
  const m=el(`<div class="modal"><button class="modal-close" aria-label="close">×</button><h3>${esc(t('shareTitle'))}</h3><canvas></canvas><div class="modal-actions"><button class="primary" data-a="dl">${esc(t('dl'))}</button><button data-a="copy">${esc(t('copyImg'))}</button><button data-a="x">${esc(t('postX'))}</button></div><div class="modal-hint">${esc(t('shareHint'))}</div></div>`);
  bg.appendChild(m); document.body.appendChild(bg);
  drawShareCard(m.querySelector('canvas'));
  const canvas=m.querySelector('canvas');
  const close=()=>bg.remove();
  bg.addEventListener('click', e => { if (e.target===bg) close(); });
  m.querySelector('.modal-close').onclick=close;
  m.querySelector('[data-a=dl]').onclick=()=>canvas.toBlob(b=>{ const a=document.createElement('a'); a.href=URL.createObjectURL(b); a.download="tokenburning-stats.png"; document.body.appendChild(a); a.click(); a.remove(); });
  m.querySelector('[data-a=copy]').onclick=ev=>{ const btn=ev.currentTarget; canvas.toBlob(async b=>{ try { await navigator.clipboard.write([new ClipboardItem({'image/png':b})]); const o=btn.textContent; btn.textContent=t('copied'); setTimeout(()=>btn.textContent=o,1600); } catch(_){} }); };
  m.querySelector('[data-a=x]').onclick=()=>{ const k=(lastSummary&&lastSummary.kpis)||{cost:0}; const txt=t('tweet').replace('{cost}',fmtUSD(k.cost)).replace('{period}',periodLabel()); window.open("https://twitter.com/intent/tweet?text="+encodeURIComponent(txt)+"&url="+encodeURIComponent("https://tokenburning.online"),"_blank","noopener"); };
}
function wireShare(){ const b=$("#share"); if (b) { b.textContent=t('share'); b.onclick=openShareModal; } }

async function load() {
  try {
    const r = await fetch(`/api/summary?period=${period}&t=${encodeURIComponent(token)}`);
    if (!r.ok) throw new Error("HTTP "+r.status);
    render(await r.json());
  } catch (e) {
    $("#app").innerHTML = `<p class="subtitle" style="padding:40px 0">${t('loadError')}: ${e.message}</p>`;
  }
}
renderLang();
renderPeriod();
wireShare();
load();
