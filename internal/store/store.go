package store

import (
	"database/sql"

	"github.com/rshatskiy/tokenburning/internal/model"
	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

// Open открывает/создаёт БД с включённым WAL и busy_timeout, прогоняет миграции.
func Open(path string) (*DB, error) {
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
	_, err := d.db.Exec(`
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
    session_id      TEXT,
    project_key     TEXT,
    extra_raw       BLOB
);
CREATE INDEX IF NOT EXISTS idx_events_model ON events(model);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
`)
	return err
}

// Insert записывает события идемпотентно (INSERT OR IGNORE по event_id) одной транзакцией.
func (d *DB) Insert(events []model.Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO events
        (event_id,tool,ts,model,billing_mode,cost_amount,cost_currency,cost_basis,pricing_version,
         tok_input,tok_output,tok_cache_read,tok_cache_1h,tok_cache_5m,tok_reasoning,session_id,project_key,extra_raw)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		if _, err := stmt.Exec(e.EventID, string(e.Tool), e.TS.Unix(), e.Model, string(e.BillingMode),
			e.Cost.Amount, e.Cost.Currency, string(e.Cost.Basis), e.Cost.PricingVersion,
			e.Tokens.Input, e.Tokens.Output, e.Tokens.CacheRead, e.Tokens.Cache1h, e.Tokens.Cache5m, e.Tokens.Reasoning,
			nullStr(e.SessionID), nullStr(e.ProjectKey), e.ExtraRaw); err != nil {
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

// ModelSummary — агрегат стоимости по модели.
type ModelSummary struct {
	Model      string  `json:"model"`
	Events     int64   `json:"events"`
	CostAmount float64 `json:"cost"`
}

func (d *DB) SummaryByModel() ([]ModelSummary, error) {
	rows, err := d.db.Query(`SELECT model, COUNT(*), COALESCE(SUM(cost_amount),0)
        FROM events GROUP BY model ORDER BY 3 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModelSummary
	for rows.Next() {
		var m ModelSummary
		if err := rows.Scan(&m.Model, &m.Events, &m.CostAmount); err != nil {
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

func (d *DB) SummaryByTool() ([]ToolSummary, error) {
	rows, err := d.db.Query(`SELECT tool, COUNT(*),
        COALESCE(SUM(tok_input+tok_output+tok_cache_read+tok_cache_1h+tok_cache_5m+tok_reasoning),0),
        COALESCE(SUM(cost_amount),0)
        FROM events GROUP BY tool ORDER BY 4 DESC, 3 DESC`)
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
