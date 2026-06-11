// Package opencode читает локальное хранилище OpenCode (https://opencode.ai).
//
// OpenCode держит данные в SQLite-БД в <paths.OpenCodeData> (обычно
// $XDG_DATA_HOME/opencode или ~/.local/share/opencode); файлы БД начинаются
// с "opencode" и оканчиваются на ".db". Схема:
//
//	session(id, parent_id, directory, title, time_created, time_archived, ...)
//	message(id, session_id, time_created, data)  -- data: JSON-блоб сообщения
//	part(id, message_id, session_id, data)       -- куски сообщения (текст/тулзы)
//
// Биллинг несут message.data с role=assistant (у части провайдеров — "model"):
// поле tokens{input,output,reasoning,cache{read,write}} либо, в старых записях,
// usage{input_tokens,output_tokens,cache_*_input_tokens}; модель — modelID/model.
// У старых версий токены лежат агрегатом на строке session (tokens_input, ...) —
// используем их как фолбэк, когда у сообщений сессии токенов нет.
package opencode

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	_ "modernc.org/sqlite"
)

var _ adapter.Adapter = (*Adapter)(nil)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolOpenCode }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityFull, // формат отдаёт полную разбивку, включая кэш и reasoning
		HasCache:     true,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent, model.BillingAPIUsage},
	}
}

// Discover находит файлы БД opencode*.db в каталоге данных OpenCode.
func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source
	root := paths.OpenCodeData
	if root == "" {
		return out, nil
	}
	matches, err := filepath.Glob(filepath.Join(root, "opencode*.db"))
	if err != nil {
		return out, nil // кривой паттерн невозможен, но отсутствие данных — не ошибка
	}
	for _, p := range matches {
		out = append(out, adapter.Source{Path: p})
	}
	return out, nil
}

// messageData — нужные нам поля JSON-блоба message.data.
// Поля токенов — указатели: различаем «нет поля» (фолбэк на usage) и «ноль».
type messageData struct {
	Role    string `json:"role"`
	ModelID string `json:"modelID"`
	Model   string `json:"model"`
	Tokens  *struct {
		Input     *int64 `json:"input"`
		Output    *int64 `json:"output"`
		Reasoning *int64 `json:"reasoning"`
		Cache     *struct {
			Read  *int64 `json:"read"`
			Write *int64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
	Usage *struct {
		Input      int64 `json:"input_tokens"`
		Output     int64 `json:"output_tokens"`
		CacheWrite int64 `json:"cache_creation_input_tokens"`
		CacheRead  int64 `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	// read-only + immutable: не трогаем WAL живого OpenCode (как в cursor-адаптере).
	db, err := sql.Open("sqlite", "file:"+src.Path+"?mode=ro&immutable=1")
	if err != nil {
		return adapter.Cursor{}, err
	}
	defer db.Close()

	// emitted — сессии, по которым уже ушли по-сообщенческие события;
	// для них агрегат session-уровня не нужен (иначе двойной счёт).
	emitted := map[string]bool{}

	rows, err := db.Query(`
		SELECT m.id, m.session_id, m.time_created,
		       CAST(m.data AS BLOB),
		       CAST(IFNULL(s.directory, '') AS BLOB)
		FROM message m
		LEFT JOIN session s ON s.id = m.session_id
		ORDER BY m.time_created ASC, m.id ASC`)
	if err != nil {
		// БД без ожидаемых таблиц (старая версия / миграции не прошли) — не ошибка, пусто.
		if isMissingSchema(err) {
			return adapter.Cursor{}, nil
		}
		return adapter.Cursor{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var msgID, sessionID string
		var tc int64
		var data, dir []byte
		if err := rows.Scan(&msgID, &sessionID, &tc, &data, &dir); err != nil {
			return adapter.Cursor{}, err
		}
		if e, ok := a.messageEvent(msgID, sessionID, tc, data, string(dir), quarantine); ok {
			emitted[sessionID] = true
			emit(e)
		}
	}
	if err := rows.Err(); err != nil {
		return adapter.Cursor{}, err
	}

	a.collectSessionFallback(db, emitted, emit)
	// Cursor пустой: SQLite — не append-источник, перечитываем целиком
	// (идемпотентно по event_id).
	return adapter.Cursor{}, nil
}

// messageEvent разбирает одно сообщение. Битый JSON — в карантин; сообщения
// без биллинга (user, нулевые токены) просто пропускаются.
func (a *Adapter) messageEvent(msgID, sessionID string, tc int64, data []byte, dir string, quarantine adapter.QuarantineFunc) (model.Event, bool) {
	var d messageData
	if err := json.Unmarshal(data, &d); err != nil {
		quarantine(append([]byte(nil), data...), err)
		return model.Event{}, false
	}
	// Биллинг несут только ответы модели ("model" — вариант роли у части провайдеров).
	if d.Role != "assistant" && d.Role != "model" {
		return model.Event{}, false
	}
	tk := d.tokens()
	if tk.TotalOrSum() == 0 {
		return model.Event{}, false // служебное сообщение без токенов — не событие
	}
	return model.Event{
		// id сообщений OpenCode (msg_...) уникальны, префикс с сессией — для читаемости
		// и защиты от коллизий между БД.
		EventID:     "opencode:" + sessionID + ":" + msgID,
		Tool:        model.ToolOpenCode,
		TS:          parseTS(tc),
		Model:       normalizeModel(d.ModelID, d.Model),
		BillingMode: model.BillingFlatEquivalent,
		Tokens:      tk,
		SessionID:   sessionID,
		ProjectKey:  dir,
	}, true
}

// collectSessionFallback эмитит агрегаты session-уровня для сессий, у которых
// нет по-сообщенческих токенов (старые версии OpenCode хранили только итог на
// строке session). Best-effort: колонок может не быть — тогда тихо выходим.
func (a *Adapter) collectSessionFallback(db *sql.DB, emitted map[string]bool, emit adapter.EmitFunc) {
	rows, err := db.Query(`
		SELECT id, CAST(IFNULL(directory, '') AS BLOB), time_created,
		       IFNULL(tokens_input, 0), IFNULL(tokens_output, 0), IFNULL(tokens_reasoning, 0),
		       IFNULL(tokens_cache_read, 0), IFNULL(tokens_cache_write, 0), IFNULL(model_id, '')
		FROM session`)
	if err != nil {
		return // нет таблицы/колонок — у этой версии схемы агрегатов нет
	}
	defer rows.Close()

	for rows.Next() {
		var id, modelID string
		var dir []byte
		var tc int64
		var tk model.Tokens
		if err := rows.Scan(&id, &dir, &tc, &tk.Input, &tk.Output, &tk.Reasoning, &tk.CacheRead, &tk.Cache5m, &modelID); err != nil {
			return
		}
		if emitted[id] || tk.TotalOrSum() == 0 {
			continue
		}
		emit(model.Event{
			EventID:     "opencode:" + id + ":session", // один агрегат на сессию — стабилен
			Tool:        model.ToolOpenCode,
			TS:          parseTS(tc),
			Model:       normalizeModel(modelID, ""),
			BillingMode: model.BillingFlatEquivalent,
			Tokens:      tk,
			SessionID:   id,
			ProjectKey:  string(dir),
		})
	}
}

// tokens нормализует разбивку: современное поле tokens{...} приоритетно,
// usage{...} (старый Anthropic-формат) — пофлейдовый фолбэк.
func (d *messageData) tokens() model.Tokens {
	var u struct{ in, out, cr, cw int64 }
	if d.Usage != nil {
		u.in, u.out, u.cr, u.cw = d.Usage.Input, d.Usage.Output, d.Usage.CacheRead, d.Usage.CacheWrite
	}
	var tk model.Tokens
	t := d.Tokens
	var cr, cw *int64
	if t != nil && t.Cache != nil {
		cr, cw = t.Cache.Read, t.Cache.Write
	}
	if t != nil {
		tk.Input = pick(t.Input, u.in)
		tk.Output = pick(t.Output, u.out)
		tk.Reasoning = pick(t.Reasoning, 0)
	} else {
		tk.Input, tk.Output = u.in, u.out
	}
	tk.CacheRead = pick(cr, u.cr)
	tk.Cache5m = pick(cw, u.cw) // OpenCode не различает горизонты кэша — пишем в 5m, как claudecode без breakdown
	return tk
}

func pick(p *int64, fallback int64) int64 {
	if p != nil {
		return *p
	}
	return fallback
}

// parseTS: OpenCode пишет time_created в миллисекундах, но на всякий случай
// (старые записи) различаем секунды эвристикой, как и конкуренты: < 1e12 — секунды.
func parseTS(raw int64) time.Time {
	if raw < 1e12 {
		return time.Unix(raw, 0).UTC()
	}
	return time.UnixMilli(raw).UTC()
}

// normalizeModel выбирает modelID (приоритет) или model и срезает префикс
// провайдера вида "anthropic/" — каталог цен знает голые id моделей.
func normalizeModel(modelID, modelAlt string) string {
	name := modelID
	if name == "" {
		name = modelAlt
	}
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		return "unknown"
	}
	return name
}

// isMissingSchema — БД есть, но ожидаемых таблиц нет (OpenCode ещё не мигрировал).
func isMissingSchema(err error) bool {
	return strings.Contains(err.Error(), "no such table")
}
