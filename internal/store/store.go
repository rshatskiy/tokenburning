package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

// DefaultPath — стандартный путь к локальной БД: ~/.tokenburning/tokenburning.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tokenburning", "tokenburning.db"), nil
}

// Open открывает/создаёт БД с включённым WAL и busy_timeout, прогоняет миграции.
// Родительский каталог создаётся здесь: на свежей машине его ещё нет, и без этого
// любая команда падала бы с SQLITE_CANTOPEN(14).
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1) // единственный писатель — сериализация записи (§8.2)
	d := &DB{db: sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) migrate() error {
	if _, err := d.db.Exec(`
CREATE TABLE IF NOT EXISTS events (
    event_id        TEXT PRIMARY KEY,
    tool            TEXT NOT NULL,
    ts              INTEGER NOT NULL,
    model           TEXT NOT NULL,
    billing_mode    TEXT NOT NULL,
    cost_amount     REAL NOT NULL,
    cost_currency   TEXT NOT NULL,
    cost_basis      TEXT NOT NULL,
    pricing_version TEXT NOT NULL,
    tok_input       INTEGER NOT NULL DEFAULT 0,
    tok_output      INTEGER NOT NULL DEFAULT 0,
    tok_cache_read  INTEGER NOT NULL DEFAULT 0,
    tok_cache_1h    INTEGER NOT NULL DEFAULT 0,
    tok_cache_5m    INTEGER NOT NULL DEFAULT 0,
    tok_reasoning   INTEGER NOT NULL DEFAULT 0,
    tok_total       INTEGER NOT NULL DEFAULT 0, -- источник истины для новых БД; для старых добавляется ALTER ниже
    session_id      TEXT,
    project_key     TEXT,
    extra_raw       BLOB
);
CREATE INDEX IF NOT EXISTS idx_events_model ON events(model);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE TABLE IF NOT EXISTS source_cursors (
    path        TEXT PRIMARY KEY,
    fid_a       INTEGER NOT NULL DEFAULT 0,
    fid_b       INTEGER NOT NULL DEFAULT 0,
    size        INTEGER NOT NULL DEFAULT 0,
    mtime       INTEGER NOT NULL DEFAULT 0, -- UnixNano: секундной точности мало для детекта перезаписи
    offset      INTEGER NOT NULL DEFAULT 0,
    header_hash TEXT NOT NULL DEFAULT ''    -- sha256 первых ≤4КБ: детект перезаписи при том же inode
);
`); err != nil {
		return err
	}
	// tok_total добавлен позже; для уже существующих БД добавляем колонку.
	// Игнорируем только "duplicate column name" (колонка уже есть), прочие ошибки — наверх.
	if _, err := d.db.Exec(`ALTER TABLE events ADD COLUMN tok_total INTEGER NOT NULL DEFAULT 0`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	return nil
}

// Insert записывает события идемпотентно одной транзакцией. По конфликту event_id —
// UPDATE, а не IGNORE: codex агрегирует живую сессию в одно «растущее» событие со
// стабильным id, и IGNORE навсегда замораживал бы числа первого скана. WHERE
// ограничивает запись реально изменившимися строками (повторный скан без изменений
// не генерирует write-трафик).
func (d *DB) Insert(events []model.Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO events
        (event_id,tool,ts,model,billing_mode,cost_amount,cost_currency,cost_basis,pricing_version,
         tok_input,tok_output,tok_cache_read,tok_cache_1h,tok_cache_5m,tok_reasoning,tok_total,session_id,project_key,extra_raw)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(event_id) DO UPDATE SET
          tool=excluded.tool, ts=excluded.ts, model=excluded.model, billing_mode=excluded.billing_mode,
          cost_amount=excluded.cost_amount, cost_currency=excluded.cost_currency, cost_basis=excluded.cost_basis,
          pricing_version=excluded.pricing_version,
          tok_input=excluded.tok_input, tok_output=excluded.tok_output, tok_cache_read=excluded.tok_cache_read,
          tok_cache_1h=excluded.tok_cache_1h, tok_cache_5m=excluded.tok_cache_5m,
          tok_reasoning=excluded.tok_reasoning, tok_total=excluded.tok_total,
          session_id=excluded.session_id, project_key=excluded.project_key, extra_raw=excluded.extra_raw
        WHERE excluded.tok_total<>events.tok_total OR excluded.tok_input<>events.tok_input
           OR excluded.tok_output<>events.tok_output OR excluded.tok_cache_read<>events.tok_cache_read
           OR excluded.tok_reasoning<>events.tok_reasoning OR excluded.cost_amount<>events.cost_amount
           OR excluded.model<>events.model`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		if _, err := stmt.Exec(e.EventID, string(e.Tool), e.TS.Unix(), e.Model, string(e.BillingMode),
			e.Cost.Amount, e.Cost.Currency, string(e.Cost.Basis), e.Cost.PricingVersion,
			e.Tokens.Input, e.Tokens.Output, e.Tokens.CacheRead, e.Tokens.Cache1h, e.Tokens.Cache5m, e.Tokens.Reasoning, e.Tokens.Total,
			nullStr(e.SessionID), nullStr(e.ProjectKey), e.ExtraRaw); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SourceCursor — сохранённое состояние инкрементального чтения одного источника.
// FileID детектирует ротацию/подмену файла, Size+MTime — «ничего не изменилось»,
// Offset — позиция дочитанного хвоста (для append-only источников).
type SourceCursor struct {
	Path       string
	FileID     platform.FileID
	Size       int64
	MTime      int64 // UnixNano
	Offset     int64
	HeaderHash string // sha256 первых min(Size, 4096) байт
}

// SourceCursors возвращает все сохранённые курсоры по пути источника.
func (d *DB) SourceCursors() (map[string]SourceCursor, error) {
	rows, err := d.db.Query(`SELECT path, fid_a, fid_b, size, mtime, offset, header_hash FROM source_cursors`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]SourceCursor{}
	for rows.Next() {
		var c SourceCursor
		var a, b int64 // uint64 хранится как int64-битпаттерн (database/sql не умеет uint64 со старшим битом)
		if err := rows.Scan(&c.Path, &a, &b, &c.Size, &c.MTime, &c.Offset, &c.HeaderHash); err != nil {
			return nil, err
		}
		c.FileID.A, c.FileID.B = uint64(a), uint64(b)
		out[c.Path] = c
	}
	return out, rows.Err()
}

// SaveSourceCursors сохраняет курсоры (upsert по path) одной транзакцией.
func (d *DB) SaveSourceCursors(cs []SourceCursor) error {
	if len(cs) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO source_cursors (path, fid_a, fid_b, size, mtime, offset, header_hash)
        VALUES (?,?,?,?,?,?,?)
        ON CONFLICT(path) DO UPDATE SET fid_a=excluded.fid_a, fid_b=excluded.fid_b,
          size=excluded.size, mtime=excluded.mtime, offset=excluded.offset, header_hash=excluded.header_hash`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range cs {
		if _, err := stmt.Exec(c.Path, int64(c.FileID.A), int64(c.FileID.B), c.Size, c.MTime, c.Offset, c.HeaderHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ModelSummary — агрегат стоимости и токенов по модели.
type ModelSummary struct {
	Model      string  `json:"model"`
	Events     int64   `json:"events"`
	Tokens     int64   `json:"tokens"`
	CostAmount float64 `json:"cost"`
}

func (d *DB) SummaryByModel(since time.Time) ([]ModelSummary, error) {
	rows, err := d.db.Query(`SELECT model, COUNT(*),
        COALESCE(SUM(MAX(tok_total, tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning)),0),
        COALESCE(SUM(cost_amount),0)
        FROM events WHERE ts >= ? GROUP BY model ORDER BY 4 DESC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModelSummary
	for rows.Next() {
		var m ModelSummary
		if err := rows.Scan(&m.Model, &m.Events, &m.Tokens, &m.CostAmount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ToolSummary — агрегат по инструменту (кросс-тул вид).
type ToolSummary struct {
	Tool       string  `json:"tool"`
	Events     int64   `json:"events"`
	Tokens     int64   `json:"tokens"`
	CostAmount float64 `json:"cost"`
}

func (d *DB) SummaryByTool(since time.Time) ([]ToolSummary, error) {
	rows, err := d.db.Query(`SELECT tool, COUNT(*),
        COALESCE(SUM(MAX(tok_total, tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning)),0),
        COALESCE(SUM(cost_amount),0)
        FROM events WHERE ts >= ? GROUP BY tool ORDER BY 4 DESC, 3 DESC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ToolSummary
	for rows.Next() {
		var s ToolSummary
		if err := rows.Scan(&s.Tool, &s.Events, &s.Tokens, &s.CostAmount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
