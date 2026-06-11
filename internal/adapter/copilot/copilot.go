// Package copilot — адаптер GitHub Copilot. Два источника:
//
//  1. CLI-сессии: <CopilotSessions>/<sessionId>/events.jsonl (legacy-формат
//     с реальными outputTokens; input локально не пишется).
//  2. Транскрипты VS Code: <VSCodeStorageDirs[i]>/workspaceStorage/*/
//     GitHub.copilot-chat/transcripts/*.jsonl — токенов в формате нет,
//     оцениваем по длине контента (~4 символа на токен), модель выводим
//     из префиксов tool call id (toolu_* → Anthropic, call_* → OpenAI).
//
// Все числа — оценка (Capabilities: FidelityPartial), кэш-полей источник
// не отдаёт, поэтому в событиях их нет.
package copilot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

var _ adapter.Adapter = (*Adapter)(nil)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolCopilot }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityPartial, // транскрипты — оценка по символам, CLI — только output
		HasCache:     false,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent},
	}
}

// charsPerToken — грубая оценка «4 символа ≈ 1 токен» (как у конкурента).
const charsPerToken = 4

// Синтетические идентификаторы модели, когда точная модель неизвестна.
const (
	modelAuto          = "copilot-auto"           // не удалось определить семейство
	modelOpenAIAuto    = "copilot-openai-auto"    // tool call id вида call_*
	modelAnthropicAuto = "copilot-anthropic-auto" // tool call id вида toolu_*
)

// toolCallModelHints — известные префиксы tool call id по провайдерам.
// Порядок важен: более длинные префиксы первыми.
var toolCallModelHints = []struct {
	prefix string
	model  string
}{
	{"toolu_bdrk_", modelAnthropicAuto},
	{"toolu_vrtx_", modelAnthropicAuto},
	{"tooluse_", modelAnthropicAuto},
	{"toolu_", modelAnthropicAuto},
	{"call_", modelOpenAIAuto},
}

// Discover находит оба вида источников; отсутствие каталогов — не ошибка.
func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source

	// 1) CLI-сессии: по одному events.jsonl на каталог сессии.
	if dir := paths.CopilotSessions; dir != "" {
		entries, err := os.ReadDir(dir)
		if err == nil { // нет каталога — просто нет CLI-сессий
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				p := filepath.Join(dir, e.Name(), "events.jsonl")
				if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
					out = append(out, adapter.Source{Path: p})
				}
			}
		}
	}

	// 2) Транскрипты VS Code: workspaceStorage/*/GitHub.copilot-chat/transcripts/*.jsonl
	for _, userDir := range paths.VSCodeStorageDirs {
		wsRoot := filepath.Join(userDir, "workspaceStorage")
		wsDirs, err := os.ReadDir(wsRoot)
		if err != nil {
			continue // редактор не установлен — ок
		}
		for _, ws := range wsDirs {
			if !ws.IsDir() {
				continue
			}
			tDir := filepath.Join(wsRoot, ws.Name(), "GitHub.copilot-chat", "transcripts")
			files, err := os.ReadDir(tDir)
			if err != nil {
				continue // у workspace нет транскриптов Copilot — ок
			}
			for _, f := range files {
				if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
					continue
				}
				out = append(out, adapter.Source{Path: filepath.Join(tDir, f.Name())})
			}
		}
	}
	return out, nil
}

// rawEvent — общая обёртка строки JSONL обоих форматов.
type rawEvent struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// rawData — объединение полей data, которые мы читаем (оба формата).
type rawData struct {
	// legacy (CLI)
	NewModel     string `json:"newModel"`
	Model        string `json:"model"`
	OutputTokens int64  `json:"outputTokens"`
	// общие / транскрипт
	MessageID     string          `json:"messageId"`
	Content       string          `json:"content"`
	ReasoningText string          `json:"reasoningText"`
	Producer      string          `json:"producer"`
	ToolRequests  json.RawMessage `json:"toolRequests"` // битые сессии слали не-массив — парсим лениво
}

type toolRequest struct {
	ToolCallID string `json:"toolCallId"`
	Name       string `json:"name"`
}

// parsedLine — успешно разобранная строка с её исходником и порядковым номером.
type parsedLine struct {
	ev  rawEvent
	raw []byte
	idx int // номер непустой строки в файле — fallback для EventID
}

// Collect читает файл целиком (транскриптам нужен весь файл для вывода модели,
// а сами файлы переписываются редактором) и возвращает пустой курсор:
// идемпотентность обеспечивается стабильными EventID.
func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	f, err := os.Open(src.Path)
	if err != nil {
		return adapter.Cursor{}, err
	}
	defer f.Close()

	// bufio.Reader.ReadBytes переживает строки любой длины (Scanner — нет).
	var lines []parsedLine
	r := bufio.NewReader(f)
	idx := 0
	for {
		line, readErr := r.ReadBytes('\n')
		trimmed := bytes.TrimRight(line, "\r\n")
		if len(bytes.TrimSpace(trimmed)) > 0 {
			var ev rawEvent
			cp := append([]byte(nil), trimmed...)
			if err := json.Unmarshal(cp, &ev); err != nil {
				quarantine(cp, err) // битая строка — в карантин, сбор не роняем
			} else {
				lines = append(lines, parsedLine{ev: ev, raw: cp, idx: idx})
			}
			idx++
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return adapter.Cursor{}, readErr
		}
	}

	sessionID := sessionIDFromPath(src.Path)
	project := projectKey(src.Path)

	if isTranscriptFormat(lines) {
		a.collectTranscript(lines, sessionID, project, emit, quarantine)
	} else {
		a.collectLegacy(lines, sessionID, project, emit, quarantine)
	}
	return adapter.Cursor{}, nil
}

// isTranscriptFormat: транскрипт VS Code начинается с session.start
// от producer=copilot-agent; всё остальное считаем legacy CLI-форматом.
func isTranscriptFormat(lines []parsedLine) bool {
	if len(lines) == 0 {
		return false
	}
	first := lines[0]
	if first.ev.Type != "session.start" || len(first.ev.Data) == 0 {
		return false
	}
	var d rawData
	if err := json.Unmarshal(first.ev.Data, &d); err != nil {
		return false
	}
	return d.Producer == "copilot-agent"
}

// collectLegacy — CLI-формат events.jsonl: assistant.message несёт реальные
// outputTokens; модель отслеживается через session.model_change и data.model.
func (a *Adapter) collectLegacy(lines []parsedLine, sessionID, project string, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) {
	currentModel := ""
	for _, p := range lines {
		var d rawData
		if len(p.ev.Data) > 0 {
			if err := json.Unmarshal(p.ev.Data, &d); err != nil {
				// data не разобралась: биллинг несут только assistant.message —
				// их в карантин, остальное молча пропускаем.
				if p.ev.Type == "assistant.message" {
					quarantine(p.raw, err)
				}
				continue
			}
		}

		// Новые события включают модель явно — она важнее текущей.
		if d.Model != "" {
			currentModel = d.Model
		}

		switch p.ev.Type {
		case "session.model_change":
			if d.NewModel != "" {
				currentModel = d.NewModel
			}
		case "assistant.message":
			if d.OutputTokens == 0 || currentModel == "" {
				continue // нулевые/безмодельные записи не несут биллинга
			}
			ts, err := time.Parse(time.RFC3339, p.ev.Timestamp)
			if err != nil {
				quarantine(p.raw, fmt.Errorf("непарсимый timestamp %q: %w", p.ev.Timestamp, err))
				continue
			}
			emit(model.Event{
				EventID:     eventID(sessionID, d.MessageID, p.idx),
				Tool:        model.ToolCopilot,
				TS:          ts,
				Model:       currentModel,
				BillingMode: model.BillingFlatEquivalent,
				// Input локально не пишется — только реальный output из файла.
				Tokens:     model.Tokens{Output: d.OutputTokens},
				SessionID:  sessionID,
				ProjectKey: project,
			})
		}
	}
}

// collectTranscript — формат VS Code: токенов нет, оцениваем по длине текста;
// модель одна на файл, выводится из префиксов tool call id / явных data.model.
func (a *Adapter) collectTranscript(lines []parsedLine, sessionID, project string, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) {
	modelName := inferModel(lines)
	pendingUser := "" // последнее сообщение пользователя — основа оценки input

	for _, p := range lines {
		if len(p.ev.Data) == 0 {
			continue
		}
		switch p.ev.Type {
		case "user.message":
			var d rawData
			if err := json.Unmarshal(p.ev.Data, &d); err == nil {
				pendingUser = truncateRunes(d.Content, 500)
			}
		case "assistant.message":
			var d rawData
			if err := json.Unmarshal(p.ev.Data, &d); err != nil {
				quarantine(p.raw, err)
				continue
			}
			toolReqs := parseToolRequests(d.ToolRequests)
			// Пустые стриминговые заглушки без контента и тулзов пропускаем.
			if d.Content == "" && d.ReasoningText == "" && len(toolReqs) == 0 {
				continue
			}
			ts, err := time.Parse(time.RFC3339, p.ev.Timestamp)
			if err != nil {
				quarantine(p.raw, fmt.Errorf("непарсимый timestamp %q: %w", p.ev.Timestamp, err))
				continue
			}

			// Если файл всё же содержит outputTokens — верим им; иначе оценка.
			out := d.OutputTokens
			var reasoning int64
			if out == 0 {
				out = estimateTokens(d.Content)
				reasoning = estimateTokens(d.ReasoningText)
			}
			in := estimateTokens(pendingUser)

			emit(model.Event{
				EventID:     eventID(sessionID, d.MessageID, p.idx),
				Tool:        model.ToolCopilot,
				TS:          ts,
				Model:       modelName,
				BillingMode: model.BillingFlatEquivalent,
				Tokens:      model.Tokens{Input: in, Output: out, Reasoning: reasoning},
				SessionID:   sessionID,
				ProjectKey:  project,
			})
			pendingUser = "" // input считаем один раз на ответ
		}
	}
}

// inferModel выбирает модель файла: явные data.model весят 100, каждый
// узнанный префикс tool call id — 1; при пустоте — copilot-auto.
func inferModel(lines []parsedLine) string {
	counts := map[string]int{}
	for _, p := range lines {
		if len(p.ev.Data) == 0 {
			continue
		}
		var d rawData
		if err := json.Unmarshal(p.ev.Data, &d); err != nil {
			continue
		}
		// Часть новых событий (tool.execution_complete и т.п.) несёт модель явно.
		if d.Model != "" {
			counts[d.Model] += 100
		}
		if p.ev.Type != "assistant.message" {
			continue
		}
		for _, tr := range parseToolRequests(d.ToolRequests) {
			for _, h := range toolCallModelHints {
				if strings.HasPrefix(tr.ToolCallID, h.prefix) {
					counts[h.model]++
					break
				}
			}
		}
	}
	if len(counts) == 0 {
		return modelAuto
	}
	// Детерминированный выбор: максимум очков, при равенстве — по алфавиту.
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best, bestN := modelAuto, 0
	for _, k := range keys {
		if counts[k] > bestN {
			best, bestN = k, counts[k]
		}
	}
	return best
}

// parseToolRequests терпимо разбирает toolRequests: битые сессии записывали
// туда строку или null — тогда считаем, что тулзов не было.
func parseToolRequests(raw json.RawMessage) []toolRequest {
	if len(raw) == 0 {
		return nil
	}
	var out []toolRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// eventID — стабильный естественный ключ: id сессии + messageId,
// при его отсутствии — порядковый номер строки в файле.
func eventID(sessionID, messageID string, idx int) string {
	if messageID != "" {
		return "copilot:" + sessionID + ":" + messageID
	}
	return fmt.Sprintf("copilot:%s:idx:%d", sessionID, idx)
}

// sessionIDFromPath: транскрипты называются <uuid>.jsonl (36 символов) —
// берём имя файла; иначе (events.jsonl) — имя каталога сессии.
func sessionIDFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if len(base) == 36 {
		return base
	}
	return filepath.Base(filepath.Dir(path))
}

// projectKey достаёт рабочий каталог проекта, если он доступен:
// CLI — workspace.yaml рядом с events.jsonl; VS Code — workspace.json
// каталога workspaceStorage (folder: "file:///...").
func projectKey(srcPath string) string {
	dir := filepath.Dir(srcPath)

	if b, err := os.ReadFile(filepath.Join(dir, "workspace.yaml")); err == nil {
		if cwd := parseCwd(string(b)); cwd != "" {
			return cwd
		}
	}

	if filepath.Base(dir) == "transcripts" {
		// .../workspaceStorage/<ws>/GitHub.copilot-chat/transcripts/<f>.jsonl
		wsDir := filepath.Dir(filepath.Dir(dir))
		if b, err := os.ReadFile(filepath.Join(wsDir, "workspace.json")); err == nil {
			var w struct {
				Folder string `json:"folder"`
			}
			if json.Unmarshal(b, &w) == nil && w.Folder != "" {
				return folderURLToPath(w.Folder)
			}
		}
	}
	return ""
}

var (
	cwdRe     = regexp.MustCompile(`(?m)^cwd:\s*(.+)$`)
	commentRe = regexp.MustCompile(`\s*#.*$`)
)

// parseCwd вытаскивает значение cwd из workspace.yaml без YAML-парсера:
// одна строка, опциональные кавычки и хвостовой комментарий.
func parseCwd(yaml string) string {
	m := cwdRe.FindStringSubmatch(yaml)
	if m == nil {
		return ""
	}
	raw := commentRe.ReplaceAllString(m[1], "")
	raw = strings.Trim(raw, `'"`)
	return strings.TrimSpace(raw)
}

// folderURLToPath превращает file:///path%20x в обычный путь.
func folderURLToPath(folder string) string {
	p := strings.TrimPrefix(folder, "file://")
	if dec, err := url.PathUnescape(p); err == nil {
		p = dec
	}
	return p
}

// estimateTokens — оценка токенов по числу символов (рун), с округлением вверх.
func estimateTokens(s string) int64 {
	n := utf8.RuneCountInString(s)
	return int64((n + charsPerToken - 1) / charsPerToken)
}

// truncateRunes обрезает строку до n символов (а не байт), как .slice() у JS.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
