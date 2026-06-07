package claudecode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/lens/lens/internal/adapter"
	"github.com/lens/lens/internal/model"
	"github.com/lens/lens/internal/platform"
)

// compile-time assertion: Adapter satisfies adapter.Adapter.
var _ adapter.Adapter = (*Adapter)(nil)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolClaudeCode }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityFull,
		HasCache:     true,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent, model.BillingAPIUsage},
	}
}

func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source
	root := paths.ClaudeCodeProjects
	if root == "" {
		return out, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // недоступный подкаталог не роняет discover
		}
		if !d.IsDir() && filepath.Ext(path) == ".jsonl" {
			out = append(out, adapter.Source{Path: path})
		}
		return nil
	})
	return out, err
}

// rawRecord — поля верхнего уровня, которые нам нужны.
type rawRecord struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId"`
	SessionID string          `json:"sessionId"`
	CWD       string          `json:"cwd"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type rawMessage struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Usage *struct {
		Input          int64 `json:"input_tokens"`
		Output         int64 `json:"output_tokens"`
		CacheRead      int64 `json:"cache_read_input_tokens"`
		CacheCreation  int64 `json:"cache_creation_input_tokens"`
		CacheBreakdown *struct {
			Ephemeral1h int64 `json:"ephemeral_1h_input_tokens"`
			Ephemeral5m int64 `json:"ephemeral_5m_input_tokens"`
		} `json:"cache_creation"`
	} `json:"usage"`
}

func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	f, err := os.Open(src.Path)
	if err != nil {
		return adapter.Cursor{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024) // длинные строки JSONL
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec rawRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			quarantine(append([]byte(nil), line...), err)
			continue
		}
		if rec.Type != "assistant" || len(rec.Message) == 0 {
			continue // биллинг несут только assistant-записи
		}
		var msg rawMessage
		if err := json.Unmarshal(rec.Message, &msg); err != nil {
			quarantine(append([]byte(nil), line...), err)
			continue
		}
		if msg.Usage == nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, rec.Timestamp)
		tk := model.Tokens{
			Input:     msg.Usage.Input,
			Output:    msg.Usage.Output,
			CacheRead: msg.Usage.CacheRead,
		}
		if b := msg.Usage.CacheBreakdown; b != nil {
			tk.Cache1h = b.Ephemeral1h
			tk.Cache5m = b.Ephemeral5m
		} else {
			tk.Cache5m = msg.Usage.CacheCreation // без разбивки — весь creation как 5m
		}
		modelName := msg.Model
		if modelName == "" {
			modelName = "unknown"
		}
		emit(model.Event{
			EventID:     rec.RequestID,
			Tool:        model.ToolClaudeCode,
			TS:          ts,
			Model:       modelName,
			BillingMode: model.BillingFlatEquivalent,
			Tokens:      tk,
			SessionID:   rec.SessionID,
			ProjectKey:  rec.CWD, // хеширование — в aggregate-срезе; локально это путь
			ExtraRaw:    append([]byte(nil), line...),
		})
	}
	if err := sc.Err(); err != nil {
		return adapter.Cursor{}, err
	}
	return adapter.Cursor{}, nil
}
