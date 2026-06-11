// Package cline — адаптер семейства Cline (Cline / Roo Code / KiloCode).
// Все три расширения пишут историю задач в одном формате:
// <root>/tasks/<taskId>/ui_messages.json (+ api_conversation_history.json),
// поэтому семейство покрывается одним адаптером с Name() = model.ToolCline.
package cline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

var _ adapter.Adapter = (*Adapter)(nil)

// Идентификаторы расширений семейства в VS Code globalStorage.
var extensionIDs = []string{
	"saoudrizwan.claude-dev",     // Cline
	"rooveterinaryinc.roo-cline", // Roo Code
	"kilocode.kilo-code",         // KiloCode
}

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolCline }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:   model.FidelityFull,
		HasCache:    true, // cacheReads/cacheWrites в api_req_started
		HasSessions: true, // одна задача (task) = одна сессия
		// BYOK: расширение ходит в API по ключу пользователя — реальные деньги.
		BillingModes: []model.BillingMode{model.BillingAPIUsage},
	}
}

// Discover находит каталоги задач во всех корнях семейства:
//   - VS Code/Insiders/VSCodium: <User>/globalStorage/<extension-id>/tasks/<task>/
//   - standalone Cline: ~/.cline/data/tasks/<task>/
//
// Source.Path — каталог задачи (внутри ui_messages.json и
// api_conversation_history.json).
func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var roots []string
	for _, userDir := range paths.VSCodeStorageDirs {
		for _, ext := range extensionIDs {
			roots = append(roots, filepath.Join(userDir, "globalStorage", ext))
		}
	}
	// Отдельного поля в Paths нет: standalone-корень Cline строим от $HOME.
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".cline", "data"))
	}

	// Cline синхронизирует одну и ту же задачу между ~/.cline/data и
	// globalStorage VS Code — дедуплицируем по task id, побеждает копия
	// со свежайшим ui_messages.json (как у эталонного парсера).
	type candidate struct {
		path    string
		taskID  string
		modTime time.Time
	}
	var cands []candidate
	for _, root := range roots {
		tasksDir := filepath.Join(root, "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			continue // корня нет — нормально, расширение не установлено
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			taskDir := filepath.Join(tasksDir, e.Name())
			fi, err := os.Stat(filepath.Join(taskDir, "ui_messages.json"))
			if err != nil || fi.IsDir() {
				continue // задача без ui_messages.json бесполезна
			}
			cands = append(cands, candidate{path: taskDir, taskID: e.Name(), modTime: fi.ModTime()})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if !cands[i].modTime.Equal(cands[j].modTime) {
			return cands[i].modTime.After(cands[j].modTime)
		}
		return cands[i].path < cands[j].path // стабильный порядок при равных mtime
	})
	seen := make(map[string]bool, len(cands))
	var out []adapter.Source
	for _, c := range cands {
		if seen[c.taskID] {
			continue
		}
		seen[c.taskID] = true
		out = append(out, adapter.Source{Path: c.path})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// uiMessage — запись ui_messages.json (массив таких объектов).
type uiMessage struct {
	TS   int64  `json:"ts"` // epoch millis
	Type string `json:"type"`
	Say  string `json:"say"`
	Text string `json:"text"`
}

// apiReqInfo — JSON внутри поля text записи say=api_req_started.
// Числа парсим как float64: расширение пишет их из JS, где int/float не различимы.
type apiReqInfo struct {
	TokensIn    float64 `json:"tokensIn"`
	TokensOut   float64 `json:"tokensOut"`
	CacheWrites float64 `json:"cacheWrites"`
	CacheReads  float64 `json:"cacheReads"`
	Cost        float64 `json:"cost"`
	Model       string  `json:"model"` // встречается не во всех версиях
}

// Collect читает ui_messages.json задачи целиком (файл перезаписывается,
// не append) и эмитит по одному событию на каждую запись api_req_started
// с ненулевыми токенами. Cursor не используется — возвращаем пустой.
func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	taskDir := src.Path
	taskID := filepath.Base(taskDir)

	raw, err := os.ReadFile(filepath.Join(taskDir, "ui_messages.json"))
	if err != nil {
		return adapter.Cursor{}, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		// Битый файл целиком — в карантин, но сбор остальных задач не роняем.
		quarantine(append([]byte(nil), raw...), err)
		return adapter.Cursor{}, nil
	}

	meta := readHistoryMeta(taskDir)

	// idx — порядковый номер записи api_req_started в задаче. Файл хоть и
	// перезаписывается, но записи только дописываются в конец, поэтому индекс
	// стабилен и даёт детерминированный EventID (= идемпотентный пересбор).
	idx := -1
	for _, rawEntry := range entries {
		cp := func() []byte { return append([]byte(nil), rawEntry...) }

		var m uiMessage
		if err := json.Unmarshal(rawEntry, &m); err != nil {
			quarantine(cp(), err)
			continue
		}
		if m.Type != "say" || m.Say != "api_req_started" {
			continue // биллинг несут только api_req_started
		}
		idx++ // считаем и битые/пустые, чтобы индексы последующих не плыли

		if m.Text == "" {
			continue // запрос ещё в полёте — метрик нет
		}
		var req apiReqInfo
		if err := json.Unmarshal([]byte(m.Text), &req); err != nil {
			quarantine(cp(), err)
			continue
		}
		if req.TokensIn == 0 && req.TokensOut == 0 {
			continue // прерванный/незавершённый запрос: токены не потрачены
		}
		if m.TS <= 0 {
			quarantine(cp(), fmt.Errorf("нет ts у api_req_started #%d задачи %s", idx, taskID))
			continue
		}

		modelName := req.Model
		if modelName == "" {
			modelName = meta.model
		}
		if modelName == "" {
			modelName = "unknown"
		}

		emit(model.Event{
			EventID:     taskID + ":" + strconv.Itoa(idx),
			Tool:        model.ToolCline,
			TS:          time.UnixMilli(m.TS).UTC(),
			Model:       modelName,
			BillingMode: model.BillingAPIUsage,
			Tokens: model.Tokens{
				Input:     int64(req.TokensIn),
				Output:    int64(req.TokensOut),
				CacheRead: int64(req.CacheReads),
				// Горизонт кэша расширение не сообщает — пишем как 5m,
				// аналогично fallback в claudecode.
				Cache5m: int64(req.CacheWrites),
			},
			SessionID:  taskID,
			ProjectKey: meta.workspace,
			ExtraRaw:   cp(),
		})
	}
	return adapter.Cursor{}, nil
}

// Метаданные задачи, извлекаемые из api_conversation_history.json:
// модель — из тега <model>…</model>, проект — из строки
// "Current Workspace Directory (…)" в environment_details.
var (
	modelTagRe     = regexp.MustCompile(`<model>([^<]+)</model>`)
	workspaceDirRe = regexp.MustCompile(`Current Workspace Directory \(([^)]+)\)`)
)

type historyMeta struct {
	model     string
	workspace string
}

// readHistoryMeta лучшим усилием читает api_conversation_history.json.
// Файл опционален: любая проблема — просто пустые метаданные, не карантин.
func readHistoryMeta(taskDir string) historyMeta {
	raw, err := os.ReadFile(filepath.Join(taskDir, "api_conversation_history.json"))
	if err != nil {
		return historyMeta{}
	}
	var msgs []struct {
		Role    string `json:"role"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &msgs); err != nil {
		return historyMeta{}
	}
	var meta historyMeta
	for _, msg := range msgs {
		if msg.Role != "user" {
			continue
		}
		for _, block := range msg.Content {
			if meta.model == "" {
				if mm := modelTagRe.FindStringSubmatch(block.Text); mm != nil {
					meta.model = stripProviderPrefix(mm[1])
				}
			}
			if meta.workspace == "" {
				if wm := workspaceDirRe.FindStringSubmatch(block.Text); wm != nil {
					meta.workspace = wm[1]
				}
			}
			if meta.model != "" && meta.workspace != "" {
				return meta
			}
		}
	}
	return meta
}

// stripProviderPrefix убирает роутерный префикс провайдера:
// "anthropic/claude-sonnet-4-5" → "claude-sonnet-4-5".
func stripProviderPrefix(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return name
}
