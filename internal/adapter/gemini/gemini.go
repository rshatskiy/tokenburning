// Package gemini — адаптер Gemini CLI. Сессии лежат в
// <GeminiTmp>/<project>/chats/: один файл на сессию, либо единый JSON
// (Gemini CLI <=0.38), либо JSONL (>=0.39). Внутри — сообщения с реальными
// токенами на каждый ответ модели (type:"gemini").
package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

var _ adapter.Adapter = (*Adapter)(nil)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolGemini }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityFull,
		HasCache:     true,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent},
	}
}

// Discover обходит <GeminiTmp>/<project>/chats/*.json|*.jsonl.
// Отсутствующий или пустой каталог — не ошибка: Gemini CLI может быть
// не установлен, а проект — не иметь чатов.
func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source
	root := paths.GeminiTmp
	if root == "" {
		return out, nil
	}
	projects, err := os.ReadDir(root)
	if err != nil {
		return out, nil // нет каталога — нет Gemini, не ошибка
	}
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		chats := filepath.Join(root, p.Name(), "chats")
		files, err := os.ReadDir(chats)
		if err != nil {
			continue // проект без chats/ — нормально
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if ext := filepath.Ext(f.Name()); ext != ".json" && ext != ".jsonl" {
				continue
			}
			out = append(out, adapter.Source{Path: filepath.Join(chats, f.Name())})
		}
	}
	return out, nil
}

// rawSession — заголовок сессии (единый JSON хранит и messages).
type rawSession struct {
	SessionID string            `json:"sessionId"`
	StartTime string            `json:"startTime"`
	Messages  []json.RawMessage `json:"messages"`
}

// rawLine — поля для классификации строки JSONL: заголовок сессии,
// сообщение или служебный патч {"$set":...}.
type rawLine struct {
	Set       json.RawMessage `json:"$set"`
	SessionID string          `json:"sessionId"`
	StartTime string          `json:"startTime"`
	ID        string          `json:"id"`
	Type      string          `json:"type"`
}

// rawMessage — поля сообщения, которые несут биллинг.
type rawMessage struct {
	ID        string     `json:"id"`
	Timestamp string     `json:"timestamp"`
	Type      string     `json:"type"` // "user" | "gemini" | "info"
	Model     string     `json:"model"`
	Tokens    *rawTokens `json:"tokens"`
}

type rawTokens struct {
	Input    int64 `json:"input"` // ВКЛЮЧАЕТ cached как подмножество
	Output   int64 `json:"output"`
	Cached   int64 `json:"cached"`
	Thoughts int64 `json:"thoughts"`
	Tool     int64 `json:"tool"`
	Total    int64 `json:"total"`
}

// sessionMeta — контекст файла для сборки событий.
type sessionMeta struct {
	sessionID  string
	startTime  string
	fileName   string
	projectKey string
}

func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	raw, err := os.ReadFile(src.Path)
	if err != nil {
		return adapter.Cursor{}, err
	}

	// Сначала единый JSON (<=0.38), при неудаче — JSONL (>=0.39).
	sess, msgs, ok := parseSingleJSON(raw)
	if !ok {
		sess, msgs = parseJSONL(raw, quarantine)
	}

	meta := sessionMeta{
		sessionID: sess.SessionID,
		startTime: sess.StartTime,
		fileName:  filepath.Base(src.Path),
		// Путь: <GeminiTmp>/<project>/chats/<файл> → ключ проекта — каталог project.
		projectKey: filepath.Base(filepath.Dir(filepath.Dir(src.Path))),
	}
	for i, m := range msgs {
		a.processMessage(m, i, meta, emit, quarantine)
	}
	// Курсор не нужен: файл сессии перезаписывается целиком, неизменённые
	// файлы пропускает collect.Run по FileID/Size, а идемпотентность даёт event_id.
	return adapter.Cursor{}, nil
}

// parseSingleJSON пробует разобрать файл как единый JSON-объект сессии.
func parseSingleJSON(raw []byte) (rawSession, []json.RawMessage, bool) {
	var sess rawSession
	if err := json.Unmarshal(raw, &sess); err != nil {
		return rawSession{}, nil, false
	}
	if sess.SessionID == "" || sess.Messages == nil {
		return rawSession{}, nil, false
	}
	return sess, sess.Messages, true
}

// parseJSONL разбирает формат Gemini CLI >=0.39: первая значимая строка с
// sessionId+startTime — заголовок, строки с id+type — сообщения,
// {"$set":...} — служебные патчи. Непарсимая строка идёт в карантин.
func parseJSONL(raw []byte, quarantine adapter.QuarantineFunc) (rawSession, []json.RawMessage) {
	var sess rawSession
	var msgs []json.RawMessage
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var hdr rawLine
		if err := json.Unmarshal(line, &hdr); err != nil {
			quarantine(append([]byte(nil), line...), err)
			continue
		}
		switch {
		case len(hdr.Set) > 0:
			// служебный патч — биллинга не несёт
		case hdr.SessionID != "" && hdr.StartTime != "" && sess.SessionID == "":
			sess.SessionID = hdr.SessionID
			sess.StartTime = hdr.StartTime
		case hdr.ID != "" && hdr.Type != "":
			msgs = append(msgs, append(json.RawMessage(nil), line...))
		}
	}
	return sess, msgs
}

// processMessage превращает одно сообщение в событие. Любая проблема ведёт
// в карантин, но никогда не роняет сбор и не эмитит частичное событие.
func (a *Adapter) processMessage(raw json.RawMessage, idx int, meta sessionMeta, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) {
	cp := func() []byte { return append([]byte(nil), raw...) }

	var msg rawMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		quarantine(cp(), err)
		return
	}
	// Биллинг несут только ответы модели с usage и именем модели.
	if msg.Type != "gemini" || msg.Tokens == nil || msg.Model == "" {
		return
	}
	t := msg.Tokens
	if t.Input == 0 && t.Output == 0 && t.Cached == 0 && t.Thoughts == 0 {
		return // пустой usage — событие не нужно
	}

	tsRaw := msg.Timestamp
	if tsRaw == "" {
		tsRaw = meta.startTime // у старых записей timestamp бывает пуст
	}
	ts, err := time.Parse(time.RFC3339, tsRaw)
	if err != nil {
		quarantine(cp(), fmt.Errorf("непарсимый timestamp %q: %w", tsRaw, err))
		return
	}
	if ts.UnixMilli() < 1_000_000_000_000 { // раньше 2001 года — мусорная дата
		quarantine(cp(), fmt.Errorf("неправдоподобный timestamp %q", tsRaw))
		return
	}

	// Gemini считает input ВКЛЮЧАЯ cached: вычитаем, чтобы не задвоить
	// кэш-токены сразу по двум ставкам.
	fresh := t.Input - t.Cached
	if fresh < 0 {
		fresh = 0
	}

	eventID := msg.ID
	if eventID == "" {
		eventID = meta.fileName + "#" + strconv.Itoa(idx) // стабильный фоллбэк
	}
	sessionID := meta.sessionID
	if sessionID == "" {
		sessionID = meta.fileName
	}

	emit(model.Event{
		EventID:     eventID,
		Tool:        model.ToolGemini,
		TS:          ts,
		Model:       msg.Model,
		BillingMode: model.BillingFlatEquivalent,
		Tokens: model.Tokens{
			Input:     fresh,
			Output:    t.Output,
			CacheRead: t.Cached,
			Reasoning: t.Thoughts,
		},
		SessionID:  sessionID,
		ProjectKey: meta.projectKey,
		ExtraRaw:   cp(),
	})
}
