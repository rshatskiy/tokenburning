package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	_ "modernc.org/sqlite"
)

// newDB создаёт пустую SQLite-БД с современной схемой OpenCode (session/message/part).
func newDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	stmts := []string{
		`CREATE TABLE session (
			id TEXT PRIMARY KEY, parent_id TEXT, directory TEXT, title TEXT,
			time_created INTEGER, time_archived INTEGER)`,
		`CREATE TABLE message (
			id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, data BLOB)`,
		`CREATE TABLE part (
			id TEXT PRIMARY KEY, message_id TEXT, session_id TEXT, data BLOB)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func insertSession(t *testing.T, db *sql.DB, id, dir string, tc int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO session(id, directory, title, time_created) VALUES(?,?,?,?)`,
		id, dir, "title "+id, tc); err != nil {
		t.Fatal(err)
	}
}

func insertMessage(t *testing.T, db *sql.DB, id, sessionID string, tc int64, data string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO message(id, session_id, time_created, data) VALUES(?,?,?,?)`,
		id, sessionID, tc, []byte(data)); err != nil {
		t.Fatal(err)
	}
}

func collect(t *testing.T, path string) (events []model.Event, quarantined int) {
	t.Helper()
	a := New()
	_, err := a.Collect(adapter.Source{Path: path},
		adapter.Cursor{},
		func(e model.Event) { events = append(events, e) },
		func(raw []byte, err error) { quarantined++ })
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return events, quarantined
}

func TestCollectEmitsAssistantMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	db := newDB(t, path)
	insertSession(t, db, "ses1", "/Users/me/proj", 1770000000000)

	// assistant с современным блоком tokens{...} и провайдерным префиксом модели
	insertMessage(t, db, "msg_a", "ses1", 1770000001000,
		`{"id":"msg_a","role":"assistant","modelID":"anthropic/claude-opus-4-5",
		  "cost":0.12,
		  "tokens":{"input":1200,"output":300,"reasoning":50,"cache":{"read":4000,"write":700}}}`)
	// user — без биллинга, пропускается
	insertMessage(t, db, "msg_u", "ses1", 1770000000500,
		`{"id":"msg_u","role":"user"}`)
	// assistant со старым usage-форматом и моделью без префикса
	insertMessage(t, db, "msg_b", "ses1", 1770000002000,
		`{"id":"msg_b","role":"assistant","model":"gpt-5.2-codex",
		  "usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":3,"cache_creation_input_tokens":2}}`)
	// assistant без токенов (служебное) — пропускается
	insertMessage(t, db, "msg_z", "ses1", 1770000003000,
		`{"id":"msg_z","role":"assistant","modelID":"claude-opus-4-5","tokens":{"input":0,"output":0}}`)
	// битый JSON — карантин, сбор не падает
	insertMessage(t, db, "msg_bad", "ses1", 1770000004000, `{не json`)

	events, quarantined := collect(t, path)

	if quarantined != 1 {
		t.Errorf("карантин = %d, ожидался 1", quarantined)
	}
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d, ожидалось 2: %+v", len(events), events)
	}

	e := events[0] // порядок — по time_created
	if e.EventID != "opencode:ses1:msg_a" {
		t.Errorf("EventID = %q", e.EventID)
	}
	if e.Tool != model.ToolOpenCode {
		t.Errorf("Tool = %q", e.Tool)
	}
	if e.Model != "claude-opus-4-5" {
		t.Errorf("Model = %q, ожидалось claude-opus-4-5 (без префикса провайдера)", e.Model)
	}
	want := model.Tokens{Input: 1200, Output: 300, Reasoning: 50, CacheRead: 4000, Cache5m: 700}
	if e.Tokens != want {
		t.Errorf("Tokens = %+v, ожидалось %+v", e.Tokens, want)
	}
	if e.SessionID != "ses1" {
		t.Errorf("SessionID = %q", e.SessionID)
	}
	if e.ProjectKey != "/Users/me/proj" {
		t.Errorf("ProjectKey = %q", e.ProjectKey)
	}
	if got := e.TS; !got.Equal(time.UnixMilli(1770000001000)) {
		t.Errorf("TS = %v (мс должны распознаваться)", got)
	}

	e2 := events[1]
	if e2.EventID != "opencode:ses1:msg_b" {
		t.Errorf("EventID = %q", e2.EventID)
	}
	if e2.Model != "gpt-5.2-codex" {
		t.Errorf("Model = %q", e2.Model)
	}
	want2 := model.Tokens{Input: 10, Output: 5, CacheRead: 3, Cache5m: 2}
	if e2.Tokens != want2 {
		t.Errorf("Tokens = %+v, ожидалось %+v (usage-фолбэк)", e2.Tokens, want2)
	}
}

func TestCollectSessionLevelFallback(t *testing.T) {
	// Старые версии OpenCode хранили токены агрегатом на строке session.
	path := filepath.Join(t.TempDir(), "opencode.db")
	db := newDB(t, path)
	cols := []string{"cost REAL", "tokens_input INTEGER", "tokens_output INTEGER",
		"tokens_reasoning INTEGER", "tokens_cache_read INTEGER", "tokens_cache_write INTEGER", "model_id TEXT"}
	for _, c := range cols {
		if _, err := db.Exec("ALTER TABLE session ADD COLUMN " + c); err != nil {
			t.Fatal(err)
		}
	}
	// time_created в секундах — проверяем эвристику сек/мс заодно
	insertSession(t, db, "ses1", "/w/old", 1770000000)
	if _, err := db.Exec(`UPDATE session SET tokens_input=100, tokens_output=20, tokens_cache_read=7, model_id='openrouter/qwen3-coder' WHERE id='ses1'`); err != nil {
		t.Fatal(err)
	}
	// сообщения есть, но без токенов — по-сообщенческих событий не будет
	insertMessage(t, db, "msg_1", "ses1", 1770000001, `{"role":"assistant","modelID":"qwen3-coder"}`)

	// вторая сессия: токены есть и на сессии, и в сообщении — агрегат не должен дублировать
	insertSession(t, db, "ses2", "/w/new", 1770000100)
	if _, err := db.Exec(`UPDATE session SET tokens_input=999, tokens_output=999 WHERE id='ses2'`); err != nil {
		t.Fatal(err)
	}
	insertMessage(t, db, "msg_2", "ses2", 1770000101000,
		`{"role":"assistant","modelID":"claude-sonnet-4-5","tokens":{"input":50,"output":9}}`)

	events, quarantined := collect(t, path)
	if quarantined != 0 {
		t.Errorf("карантин = %d, ожидался 0", quarantined)
	}
	if len(events) != 2 {
		t.Fatalf("эмитнуто %d, ожидалось 2 (msg ses2 + агрегат ses1): %+v", len(events), events)
	}

	var agg *model.Event
	for i := range events {
		if events[i].EventID == "opencode:ses1:session" {
			agg = &events[i]
		}
		if events[i].EventID == "opencode:ses2:session" {
			t.Errorf("агрегат ses2 не должен эмититься: у сессии есть по-сообщенческие события")
		}
	}
	if agg == nil {
		t.Fatalf("нет агрегата ses1: %+v", events)
	}
	want := model.Tokens{Input: 100, Output: 20, CacheRead: 7}
	if agg.Tokens != want {
		t.Errorf("Tokens агрегата = %+v, ожидалось %+v", agg.Tokens, want)
	}
	if agg.Model != "qwen3-coder" {
		t.Errorf("Model = %q (префикс провайдера должен срезаться)", agg.Model)
	}
	if agg.ProjectKey != "/w/old" || agg.SessionID != "ses1" {
		t.Errorf("ProjectKey/SessionID = %q/%q", agg.ProjectKey, agg.SessionID)
	}
	if !agg.TS.Equal(time.Unix(1770000000, 0)) {
		t.Errorf("TS = %v (секунды должны распознаваться)", agg.TS)
	}
}

func TestCollectMissingTablesIsEmpty(t *testing.T) {
	// БД без таблиц session/message (миграции не прошли) — не ошибка, просто пусто.
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE something_else (id TEXT)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	events, quarantined := collect(t, path)
	if len(events) != 0 || quarantined != 0 {
		t.Fatalf("эмитнуто %d, карантин %d — ожидалось 0/0", len(events), quarantined)
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"opencode.db", "opencode-v2.db", "other.db", "opencode.db-wal", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := New()
	srcs, err := a.Discover(platform.Paths{OpenCodeData: dir})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(srcs) != 2 {
		t.Fatalf("найдено %d источников, ожидалось 2 (opencode*.db): %+v", len(srcs), srcs)
	}

	// отсутствие каталога — не ошибка
	srcs, err = a.Discover(platform.Paths{OpenCodeData: filepath.Join(dir, "нет-такого")})
	if err != nil || len(srcs) != 0 {
		t.Fatalf("несуществующий каталог: srcs=%v err=%v", srcs, err)
	}
	// пустой путь — не ошибка
	srcs, err = a.Discover(platform.Paths{})
	if err != nil || len(srcs) != 0 {
		t.Fatalf("пустой путь: srcs=%v err=%v", srcs, err)
	}
}
