const I18N = {
  en: {
    all: "all", nodata: "no data", none: "(none)", session: "session",
    emptyTitle: "No data yet",
    emptyBody: "No AI coding tool logs were found on this machine (we look for Claude Code, Codex, Cursor, Gemini CLI, Copilot, OpenCode and the Cline family). Use your tools, then refresh — or run <code>tokenburning scan</code> in a terminal.",
    cost: "Cost", cacheRead: "% cache read", tokens: "Tokens", activeDays: "Active days",
    sessionsSuffix: " sessions", tools: "Tools",
    costOverTime: "Cost over time", usdPerDay: "USD / day",
    byModel: "By model", shareUsd: "$ share", events: "events",
    byTool: "By tool", tokShort: "tok", evShort: "ev", cursorActivityOnly: "cursor — activity only",
    cursorNote: "Cursor: tokens/cost are behind the server API — unavailable locally",
    topProjects: "Top projects", perTask: "$ / task",
    activity: "Activity", sessionsPerDay: "sessions/day", sessionsInPeriod: "sessions in period",
    sessionAnalytics: "session analytics", signalNotExact: "estimates, not exact", selfCoaching: "for yourself",
    noSessionData: "no session data",
    perSessionUsdUp: "$ per session ↑", durationRight: "duration →",
    normalSessions: "normal sessions", longExpensiveFriction: "long & costly",
    sessDurCost: "Sessions: duration × cost", orangeCandidates: "orange = unusually long & costly",
    perSessionMedian: "Per session (median)", min: "min", activeDuration: "active duration",
    modelCalls: "model calls", needsAttention: "Needs attention", noClearOutliers: "no clear outliers",
    stuck: "stuck?", iter: "iter", iterations: "iterations", loadError: "Load error",
    tokensLabel: "tokens", costLabel: "cost",
    bln: "B", mln: "M",
    costTip: "Total across all your AI coding tools (Claude Code, Codex, Cursor).",
    activeDaysTip: "Days you used an AI coding tool in this period.",
    cacheTip: "Share of tokens read from cache — much cheaper than fresh ones.",
    sessTip: "Session numbers are estimated from your local logs — directional signals to understand your own workflow, not exact billing.",
    statTip: "Median = a typical session (half higher, half lower). p90 = 90% of sessions are below this; the heaviest 10% go above it.",
    stuckTip: "Many model calls and high cost in one session — the model likely got stuck here.",
    barTip: "Bar length = this model's share of total cost (longest = most expensive). Gray / ~est = models with no known price.",
    activeDaysWord: "active days", perDaySfx: "/day", avgWord: "avg",
    activityTip: "Sessions per day over the period — taller bars = busier days. Hover a bar for the date.",
    cacheSaved: "cache saved",
    savedTip: "Estimated savings: your cache-read tokens cost ~10× less than fresh input would. Without caching this bill would be far higher.",
    share: "Share ↗", shareTitle: "Share your stats", dl: "Download", copyImg: "Copy image", copied: "Copied!", postX: "Post on X",
    planLine: "extracted ×{x} from your ${m}/mo plan this month",
    pToday: "today", pMonth: "month",
    planForecast: "on pace for ×{x} (≈{c}) by month end",
    insightsOk: "no obvious leaks: cache is stable, sessions look normal, every model is priced ✓",
    qModel: "Model",
    act_cache_drop: "→ compare what changed at the start of your context: MCP set, system rules, CLAUDE.md — a stable prefix brings the cache back",
    act_expensive_session: "→ split long sessions: start fresh after a big task — the expensive part is the tail, when context has ballooned",
    act_unpriced_model: "→ run: <code>tokenburning alias {model} &lt;canonical-name&gt;</code>",
    act_claude_md_big: "→ move rarely-needed parts into skills/files — only what every request needs should ride in context",
    act_mcp_many: "→ disable unused servers in ~/.claude.json — each one ships its tool schemas with every request",
    qualityT: "Model quality", qualityHint: "one-shot = edit accepted without re-editing the same file after a shell command; local-log estimate (Claude Code)",
    qEdits: "edits", qRetries: "retries", qOneShot: "one-shot",
    insightsT: "Insights", insightsHint: "deterministic signals from your local data — what to fix, not just numbers",
    in_cache_drop: "cache hit in {project} fell {from}% → {to}% this week — something breaks the prompt prefix",
    in_expensive_session: "session {session} ({tool}) cost {cost} — far above your {median} median; likely stuck or context bloat",
    in_unpriced_model: "model {model} is unpriced ($0) — map it: tokenburning alias {model} <canonical-name>",
    in_claude_md_big: "CLAUDE.md is {kb} KB (~{tok} tokens) and rides along in every request — trim it or split into skills",
    in_mcp_many: "{count} MCP servers connected — each adds tool schemas to context; disable unused ones",
    cardHeadline: "My AI coding spend", periodAll: "all time", periodPrefix: "last ", periodDays: " days",
    trackYours: "track yours →",
    shareHint: "\"Copy\" puts the card on your clipboard — paste into a post (⌘/Ctrl+V). Or \"Download\" the PNG.",
    pasteHint: "Copied ✓ Now paste it into your post (⌘/Ctrl+V).", savedInstead: "Saved PNG ↓",
    tweet: "I spent {cost} on AI coding tools ({period}) 🔥 tracked locally with tokenburning",
  },
  ru: {
    all: "всё", nodata: "нет данных", none: "(нет)", session: "сессия",
    planLine: "извлечено ×{x} из подписки ${m}/мес за этот месяц",
    pToday: "сегодня", pMonth: "месяц",
    planForecast: "темп — ×{x} (≈{c}) к концу месяца",
    insightsOk: "явных утечек нет: кэш стабилен, сессии в норме, все модели оценены ✓",
    qModel: "Модель",
    act_cache_drop: "→ сравните, что изменилось в начале контекста: набор MCP, системные правила, CLAUDE.md — стабильный префикс вернёт кэш",
    act_expensive_session: "→ дробите длинные сессии: после большой задачи начинайте новую — дороже всего хвост, когда контекст разросся",
    act_unpriced_model: "→ выполните: <code>tokenburning alias {model} &lt;каноническое-имя&gt;</code>",
    act_claude_md_big: "→ вынесите редко нужное в скиллы/отдельные файлы — в контексте должно ехать только то, что нужно каждому запросу",
    act_mcp_many: "→ отключите неиспользуемые серверы в ~/.claude.json — каждый добавляет свои схемы в каждый запрос",
    qualityT: "Качество по моделям", qualityHint: "one-shot = правка принята без повторного редактирования файла после shell-команды; оценка по локальным логам (Claude Code)",
    qEdits: "правок", qRetries: "повторов", qOneShot: "one-shot",
    insightsT: "Инсайты", insightsHint: "детерминированные сигналы из ваших локальных данных — что исправить, а не просто цифры",
    in_cache_drop: "кэш-хит в {project} упал {from}% → {to}% за неделю — что-то ломает префикс промпта",
    in_expensive_session: "сессия {session} ({tool}) стоила {cost} — сильно выше вашей медианы {median}; вероятно, модель забуксовала",
    in_unpriced_model: "модель {model} не оценена ($0) — задайте: tokenburning alias {model} <каноническое-имя>",
    in_claude_md_big: "CLAUDE.md занимает {kb} КБ (~{tok} ток.) и входит в каждый запрос — сократите или разнесите по скиллам",
    in_mcp_many: "подключено {count} MCP-серверов — каждый добавляет схемы в контекст; отключите неиспользуемые",
    emptyTitle: "Данных пока нет",
    emptyBody: "Логи ИИ-инструментов на этой машине не найдены (ищем Claude Code, Codex, Cursor, Gemini CLI, Copilot, OpenCode и семейство Cline). Поработайте с инструментами и обновите страницу — или выполните <code>tokenburning scan</code> в терминале.",
    cost: "Стоимость", cacheRead: "% кэш-чтения", tokens: "Токены", activeDays: "Активных дней",
    sessionsSuffix: " сессий", tools: "Инструменты",
    costOverTime: "Стоимость во времени", usdPerDay: "USD / день",
    byModel: "По моделям", shareUsd: "доля $", events: "событий",
    byTool: "По инструментам", tokShort: "ток", evShort: "соб", cursorActivityOnly: "cursor — только активность",
    cursorNote: "Cursor: токены/стоимость за серверным API — локально недоступны",
    topProjects: "Топ проектов", perTask: "$ / задача",
    activity: "Активность", sessionsPerDay: "сессий/день", sessionsInPeriod: "сессий за период",
    sessionAnalytics: "аналитика сессий", signalNotExact: "оценки, не точно", selfCoaching: "для себя",
    noSessionData: "нет данных по сессиям",
    perSessionUsdUp: "$ за сессию ↑", durationRight: "длительность →",
    normalSessions: "обычные сессии", longExpensiveFriction: "долгие и дорогие",
    sessDurCost: "Сессии: длительность × стоимость", orangeCandidates: "оранжевые — необычно долгие и дорогие",
    perSessionMedian: "На сессию (медиана)", min: "мин", activeDuration: "активная длительность",
    modelCalls: "обращений к модели", needsAttention: "Требуют внимания", noClearOutliers: "нет выраженных выбросов",
    stuck: "застрял?", iter: "итер", iterations: "итераций", loadError: "Ошибка загрузки",
    tokensLabel: "токенов", costLabel: "стоимость",
    bln: " млрд", mln: " млн",
    costTip: "Суммарно по всем твоим ИИ-инструментам (Claude Code, Codex, Cursor).",
    activeDaysTip: "Дни, когда ты пользовался ИИ-инструментом за этот период.",
    cacheTip: "Доля токенов, прочитанных из кэша — они намного дешевле обычных.",
    sessTip: "Цифры по сессиям — оценка по локальным логам: ориентир, чтобы понять свой стиль работы, а не точный счёт.",
    statTip: "Медиана — типичная сессия (половина выше, половина ниже). p90 — порог: 90% сессий ниже него, верхние 10% (самые тяжёлые) — выше.",
    stuckTip: "Много обращений к модели и высокая стоимость за одну сессию — модель, вероятно, забуксовала.",
    barTip: "Длина полоски = доля стоимости модели (самая длинная — самая дорогая). Серое / ~est — модели без известной цены.",
    activeDaysWord: "активных дней", perDaySfx: "/день", avgWord: "в среднем",
    activityTip: "Сессии по дням за период — выше столбик, активнее день. Наведи на столбик, чтобы увидеть дату.",
    cacheSaved: "кэш сэкономил",
    savedTip: "Оценка экономии: токены из кэша примерно в 10 раз дешевле свежего input. Без кэширования счёт был бы намного выше.",
    share: "Поделиться ↗", shareTitle: "Поделиться статистикой", dl: "Скачать", copyImg: "Копировать", copied: "Скопировано!", postX: "В X",
    cardHeadline: "Мои траты на ИИ-код", periodAll: "всё время", periodPrefix: "последние ", periodDays: " дн.",
    trackYours: "посчитай свои →",
    shareHint: "«Копировать» — карточка ляжет в буфер, вставьте в пост (⌘/Ctrl+V). Или «Скачать» — сохранит PNG.",
    pasteHint: "Скопировано ✓ Вставьте в пост (⌘/Ctrl+V).", savedInstead: "Сохранил PNG ↓",
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
const fmtUSD = (n) => {
  const c = lastSummary && lastSummary.currency;
  if (c && c.rate > 0) return Math.round(n * c.rate).toLocaleString(locale()) + " " + c.symbol;
  return "$" + Math.round(n).toLocaleString(locale());
};
const fmtTok = (n) => n >= 1e9 ? (n/1e9).toFixed(1)+t('bln') : n >= 1e6 ? (n/1e6).toFixed(0)+t('mln') : n.toLocaleString(locale());
const fmtDate = (s) => { const p = String(s).split("-"); return p.length === 3 ? p[2]+"."+p[1] : s; };
// tip-string is already html-safe (names via esc, tags <br>/<b> are literal);
// for data-tip attribute only quotes need escaping
const escAttr = (s) => String(s).replace(/"/g, "&quot;");
// «?»-иконка с подсказкой (показывается по наведению/клику — см. обработчик ниже)
const info = (tip) => `<span class="info" data-info="${escAttr(tip)}" role="button" tabindex="0" aria-label="?">?</span>`;

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
  for (const p of ["today","7d","30d","90d","month","all"]) {
    const lbl = p==="all"?t('all'):p==="today"?t('pToday'):p==="month"?t('pMonth'):p;
    const s = el(`<span${p===period?' class="on"':''}>${lbl}</span>`);
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
  // точка на каждый день (бусины) — чтобы было видно, где именно замеры
  const dotR = data.length > 45 ? "cl-pt sm" : "cl-pt";
  const labels = data.length < 2 ? "" : data.map((d,i)=>`<div class="${dotR}" style="left:${(xAt(i)/W*100).toFixed(2)}%;top:${y(d.cost).toFixed(0)}px"></div>`).join("");
  return `<div class="chartwrap" style="height:${H}px">
  <svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" preserveAspectRatio="none" style="display:block">
    <defs><linearGradient id="ar" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fb923c" stop-opacity=".38"/><stop offset="1" stop-color="#fb923c" stop-opacity="0"/></linearGradient></defs>
    <path d="${area}" fill="url(#ar)"/>
    <path d="${line}" fill="none" stroke="#fb923c" stroke-width="2.5" stroke-linecap="round" style="filter:drop-shadow(0 0 6px rgba(251,146,60,.55))"/>
    ${hits}
  </svg>${labels}</div>`;
}

function bars(items, label, val, max, tip, sub) {
  return items.map(it => {
    const v = val(it);
    const w = Math.max(2, (v/max)*100);
    const cls = v > 0 ? "bar-fill" : "bar-fill dim";
    const dt = tip ? ` data-tip="${escAttr(tip(it))}"` : "";
    const sv = sub ? `<span class="bar-sub">${sub(it)}</span>` : "";
    return `<div class="bar-row"${dt}><span class="bar-name">${esc(label(it))}</span><span class="bar-track"><span class="${cls}" style="width:${w}%"></span></span><span class="bar-val">${v>0?fmtUSD(v):"~est"}${sv}</span></div>`;
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

function kpiCard(label, num, sub, accent, tip) {
  return `<div class="shell"><div class="core"><div class="klabel">${accent?'<span class="kdot"></span> ':''}${label}${tip?info(tip):''}</div><div class="knum">${num}</div><div class="ksub">${sub||""}</div></div></div>`;
}

// мини бар-чарт сессий по дням
function activityChart(data) {
  if (!data || !data.length) return `<div class="subtitle" style="margin-top:12px">${t('nodata')}</div>`;
  const W = 300, H = 84, max = Math.max(...data.map(d => d.sessions), 1), n = data.length, bw = W / n;
  const bars = data.map((d, i) => {
    const h = Math.max(2, (d.sessions / max) * (H - 6)), x = i * bw;
    const tip = `${fmtDate(d.date)} · ${d.sessions} ${t('sessionsSuffix').trim()}`;
    return `<rect x="${(x + bw * 0.12).toFixed(1)}" y="${(H - h).toFixed(1)}" width="${(bw * 0.76).toFixed(1)}" height="${h.toFixed(1)}" rx="1.5" fill="url(#actg)" data-tip="${escAttr(tip)}"/>`;
  }).join("");
  return `<svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}" preserveAspectRatio="none" style="display:block;margin-top:12px">
    <defs><linearGradient id="actg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#fb923c"/><stop offset="1" stop-color="#ea580c" stop-opacity=".7"/></linearGradient></defs>
    ${bars}</svg>`;
}

function render(s) {
  lastSummary = s;
  renderLang();
  renderPeriod();
  const app = $("#app"); app.innerHTML = "";
  const k = s.kpis;
  // Пустая БД (first-run): вместо нулей объясняем, откуда возьмутся данные.
  if (!k.tokens && !k.sessions && !(s.byTool||[]).length) {
    app.appendChild(el(`<div class="shell"><div class="core" style="text-align:center;padding:48px 24px">
      <h3 style="margin:0 0 10px">${t('emptyTitle')}</h3>
      <div style="color:var(--dim);max-width:520px;margin:0 auto;line-height:1.5">${t('emptyBody')}</div>
    </div></div>`));
    return;
  }
  // KPI
  const cachePct = k.tokens ? Math.round(k.cacheReadTokens/k.tokens*100) : 0;
  const saved = s.cacheSavings || 0;
  const savedLine = saved > 1 ? `<br><span style="color:var(--acc)">${t('cacheSaved')} ≈${fmtUSD(saved)}</span>${info(t('savedTip'))}` : "";
  let planLine = "";
  if (s.plan && s.plan.multiplier > 0) {
    planLine = `<br><span style="color:var(--acc)">${t('planLine').replace('{x}', s.plan.multiplier.toFixed(1)).replace('{m}', Math.round(s.plan.monthlyUsd))}</span>`;
    if (s.plan.forecastX > 0) {
      planLine += `<br><span style="color:var(--dim);font-size:11px">${t('planForecast').replace('{x}', s.plan.forecastX.toFixed(1)).replace('{c}', fmtUSD(s.plan.forecastCost))}</span>`;
    }
  }
  const cacheSub = `${cachePct}${t('cacheRead')}${info(t('cacheTip'))}${savedLine}${planLine}`;
  app.appendChild(el(`<div class="kpis">
    ${kpiCard(t('cost'), fmtUSD(k.cost), cacheSub, true, t('costTip'))}
    ${kpiCard(t('tokens'), fmtTok(k.tokens), "")}
    ${kpiCard(t('activeDays'), k.activeDays+"", k.sessions+t('sessionsSuffix'), false, t('activeDaysTip'))}
    ${kpiCard(t('tools'), (k.tools||[]).length+"", esc((k.tools||[]).join(" · ")))}
  </div>`));
  // качество по моделям (one-shot/retry) — рендерится под графиком стоимости
  let qualityCard = "";
  if (s.quality && s.quality.length) {
    const qrows = s.quality.map(q => {
      let delta = "";
      if (q.deltaPct != null && Math.abs(q.deltaPct) >= 1) {
        const up = q.deltaPct > 0;
        delta = ` <span style="font-size:11px;color:${up?'#86efac':'#f87171'}">${up?'↑':'↓'}${Math.abs(q.deltaPct).toFixed(0)}</span>`;
      }
      return `<tr><td>${esc(q.model)}</td><td class="r">${q.editTurns}</td><td class="r">${q.retries}</td><td class="r"><b style="color:${q.oneShotPct>=85?'#86efac':q.oneShotPct>=70?'#fbbf77':'#f87171'}">${q.oneShotPct.toFixed(0)}%</b>${delta}</td></tr>`;
    }).join('');
    qualityCard = `<div class="shell"><div class="core"><div class="ctitle"><h3>${t('qualityT')}</h3><span class="meta">${t('qualityHint')}</span></div><table class="qtable"><tr><th>${t('qModel')}</th><th class="r">${t('qEdits')}</th><th class="r">${t('qRetries')}</th><th class="r">${t('qOneShot')}</th></tr>${qrows}</table></div></div>`;
  }
  // cost over time + by model
  const maxModel = Math.max(...(s.byModel||[]).map(m=>m.cost),1);
  app.appendChild(el(`<div class="grid2">
    <div class="stack"><div class="shell"><div class="core"><div class="ctitle"><h3>${t('costOverTime')}</h3><span class="meta">${t('usdPerDay')}</span></div>${areaChart(s.costOverTime||[])}</div></div>${qualityCard}</div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('byModel')}</h3><span class="meta">${t('shareUsd')} · ${t('tokShort')}${info(t('barTip'))}</span></div>${bars(s.byModel||[], m=>m.model, m=>m.cost, maxModel, m=>`${esc(m.model)}<br><b>${m.cost>0?fmtUSD(m.cost):"~est"}</b> · ${fmtTok(m.tokens)} ${t('tokShort')} · ${m.events} ${t('events')}`, m=>`${fmtTok(m.tokens)} ${t('tokShort')}`)}</div></div>
  </div>`));
  // слева стопкой: По инструментам + Активность · справа: Топ проектов
  const maxTool = Math.max(...(s.byTool||[]).map(tool=>tool.cost),1);
  const shortProj = (p) => p.replace(/^(\/Users\/[^/]+\/|\/home\/[^/]+\/|[A-Za-z]:[\\/]Users[\\/][^\\/]+[\\/])/, "") || p;
  const proj = (s.topProjects||[]).slice(0,6).map(p=>`<div class="proj" data-tip="${escAttr(p.project)}"><div><div>${esc(shortProj(p.project))}</div><div class="p-meta">${p.sessions}${t('sessionsSuffix')}</div></div><span style="font-family:var(--mono);font-variant-numeric:tabular-nums">${fmtUSD(p.cost)}</span></div>`).join("");
  const avgDay = k.activeDays ? (k.sessions/k.activeDays).toFixed(1) : "0";
  app.appendChild(el(`<div class="rowA">
    <div class="stack">
      <div class="shell"><div class="core"><div class="ctitle"><h3>${t('byTool')}</h3><span class="meta">${t('shareUsd')} · ${t('tokShort')}</span></div>${bars(s.byTool||[], tool=>tool.tool, tool=>tool.cost, maxTool, tool=>`${esc(tool.tool)}<br><b>${tool.cost>0?fmtUSD(tool.cost):"~est"}</b> · ${fmtTok(tool.tokens)} ${t('tokShort')} · ${tool.events} ${t('evShort')}`, tool=>`${fmtTok(tool.tokens)} ${t('tokShort')}`)}<div style="margin-top:12px;font-size:11px;color:var(--dim);font-family:var(--mono)">${t('cursorActivityOnly')}</div></div></div>
      <div class="shell"><div class="core"><div class="ctitle"><h3>${t('activity')}${info(t('activityTip'))}</h3><span class="meta">${t('sessionsPerDay')}</span></div><div style="display:flex;align-items:baseline;gap:9px;flex-wrap:wrap"><div style="font-size:30px;font-weight:600;font-variant-numeric:tabular-nums">${k.sessions}</div><div class="ksub">${k.activeDays} ${t('activeDaysWord')} · ${t('avgWord')} ${avgDay}${t('perDaySfx')}</div></div>${activityChart(s.activity)}</div></div>
    </div>
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('topProjects')}</h3><span class="meta">${t('perTask')}</span></div>${proj||`<div class="subtitle">${t('nodata')}</div>`}</div></div>
  </div>`));
  // session analytics
  app.appendChild(el(`<div id="sess"></div>`));
  renderSessions(s);
  // инсайты — «что исправить», по сигналам сервера; пусто = хорошие новости
  if (!s.insights || !s.insights.length) {
    app.appendChild(el(`<div class="shell solo"><div class="core" style="padding:14px 22px"><span style="font-size:13px;color:#86efac">${t('insightsOk')}</span></div></div>`));
  }
  if (s.insights && s.insights.length) {
    const fmtIns = (i) => {
      const d = i.data || {};
      const base = (p) => { const parts = String(p||'').split(/[\\/]/); return parts[parts.length-1] || p; };
      switch (i.kind) {
        case 'cache_drop': return t('in_cache_drop').replace('{project}', esc(base(d.project))).replace('{from}', d.fromPct).replace('{to}', d.toPct);
        case 'expensive_session': return t('in_expensive_session').replace('{session}', esc(String(d.session||'').slice(0,8))).replace('{tool}', esc(d.tool)).replace('{cost}', fmtUSD(d.cost)).replace('{median}', fmtUSD(d.median));
        case 'unpriced_model': return t('in_unpriced_model').replaceAll('{model}', esc(d.model));
        case 'claude_md_big': return t('in_claude_md_big').replace('{kb}', d.kb).replace('{tok}', fmtTok(d.estTokens||0));
        case 'mcp_many': return t('in_mcp_many').replace('{count}', d.count);
        default: return esc(i.text||i.kind);
      }
    };
    const rows = s.insights.map(i => {
      const actKey = 'act_' + i.kind;
      const d = i.data || {};
      let act = I18N[lang][actKey] != null || I18N.en[actKey] != null ? t(actKey) : '';
      if (act) act = act.replaceAll('{model}', esc(d.model||''));
      return `<div class="ins"><span style="color:${i.severity==='warn'?'#fbbf77':'#a8a29e'}">${i.severity==='warn'?'!':'•'}</span><span>${fmtIns(i)}${act?`<span class="act">${act}</span>`:''}</span></div>`;
    }).join('');
    app.appendChild(el(`<div class="shell solo"><div class="core"><div class="ctitle"><h3>${t('insightsT')}</h3><span class="meta">${t('insightsHint')}</span></div>${rows}</div></div>`));
  }
}

function renderSessions(s) {
  const host = $("#sess");
  host.innerHTML = "";
  host.appendChild(el(`<div class="eyebrow">${t('sessionAnalytics')} · <b style="color:var(--acc)">${t('signalNotExact')}</b> · ${t('selfCoaching')}${info(t('sessTip'))}</div>`));
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
    <div class="shell"><div class="core"><div class="ctitle"><h3>${t('perSessionMedian')}${info(t('statTip'))}</h3><span class="meta">${esc(cur.tool)}</span></div>
      <div class="sess-stats"><div class="sess-stat"><div class="n">${Math.round(ss.medianDurationMin || 0)}<span style="font-size:12px;color:var(--dim)">${t('min')}</span></div><div class="l">${t('activeDuration')}</div><div class="h">p90 ${Math.round(ss.p90DurationMin || 0)}m</div></div>
      <div class="sess-stat"><div class="n">${fmtTok(ss.medianTokens || 0)}</div><div class="l">${t('tokensLabel')}</div><div class="h">p90 ${fmtTok(ss.p90Tokens || 0)}</div></div></div>
      <div class="sess-stats" style="margin-top:8px"><div class="sess-stat"><div class="n">${Math.round(ss.medianIterations || 0)}</div><div class="l">${t('modelCalls')}</div><div class="h">p90 ${Math.round(ss.p90Iterations || 0)}</div></div>
      <div class="sess-stat"><div class="n">${fmtUSD(ss.medianCost || 0)}</div><div class="l">${t('costLabel')}</div><div class="h">p90 ${fmtUSD(ss.p90Cost || 0)}</div></div></div>
      ${note}
      <div style="margin-top:16px;border-top:1px solid var(--line);padding-top:12px"><div style="font-size:11px;color:var(--muted);margin-bottom:8px">${t('needsAttention')}${info(t('stuckTip'))}</div>${flagged || `<div class="subtitle">${t('noClearOutliers')}</div>`}</div>
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

// info-иконки «?»: подсказка по наведению (десктоп) и по клику/тапу (мобайл),
// позиционируется у иконки, текст переносится, не вылезает за экран.
const itip = el('<div class="info-tip"></div>');
document.body.appendChild(itip);
let itipPinned = null;
function showInfoTip(icon) {
  itip.textContent = icon.getAttribute("data-info") || "";
  itip.style.display = "block";
  const r = icon.getBoundingClientRect();
  const tr = itip.getBoundingClientRect();
  let x = r.left + r.width / 2 - tr.width / 2;
  let y = r.bottom + 8;
  x = Math.max(8, Math.min(x, window.innerWidth - tr.width - 8));
  if (y + tr.height > window.innerHeight - 8) y = r.top - tr.height - 8;
  itip.style.left = Math.max(8, x) + "px";
  itip.style.top = Math.max(8, y) + "px";
}
function hideInfoTip() { itip.style.display = "none"; itipPinned = null; }
document.addEventListener("mouseover", (e) => {
  const i = e.target.closest ? e.target.closest(".info") : null;
  if (i) showInfoTip(i);
});
document.addEventListener("mouseout", (e) => {
  const i = e.target.closest ? e.target.closest(".info") : null;
  if (i && itipPinned !== i) hideInfoTip();
});
document.addEventListener("click", (e) => {
  const i = e.target.closest ? e.target.closest(".info") : null;
  if (i) { e.preventDefault(); e.stopPropagation(); itipPinned === i ? hideInfoTip() : (showInfoTip(i), itipPinned = i); return; }
  if (itipPinned) hideInfoTip();
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
  const saveBlob=(b)=>{ const a=document.createElement('a'); a.href=URL.createObjectURL(b); a.download="tokenburning-stats.png"; document.body.appendChild(a); a.click(); a.remove(); };
  const flash=(btn,txt)=>{ const o=btn.textContent; btn.textContent=txt; setTimeout(()=>{btn.textContent=o;},1800); };
  m.querySelector('[data-a=dl]').onclick=()=>canvas.toBlob(saveBlob);
  m.querySelector('[data-a=copy]').onclick=ev=>{
    const btn=ev.currentTarget;
    const blobP=new Promise(res=>canvas.toBlob(res,'image/png'));
    // ВАЖНО: clipboard.write вызываем синхронно в обработчике клика, передавая Promise
    // картинки в ClipboardItem — иначе Safari теряет «жест пользователя» и отказывает.
    if(navigator.clipboard && window.ClipboardItem){
      try {
        navigator.clipboard.write([new ClipboardItem({'image/png':blobP})]).then(()=>{
          flash(btn,t('copied'));
          const h=m.querySelector('.modal-hint'); if(h) h.textContent=t('pasteHint');
        }).catch(()=>{ blobP.then(saveBlob); flash(btn,t('savedInstead')); });
        return;
      } catch(_){}
    }
    // браузер вовсе не умеет копировать картинку → скачиваем как запасной путь
    blobP.then(saveBlob); flash(btn,t('savedInstead'));
  };
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
