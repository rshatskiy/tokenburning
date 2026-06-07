package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
)

var _ adapter.Adapter = (*Adapter)(nil)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() model.Tool { return model.ToolCodex }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityPartial, // разбивка есть в live-сессиях, не в импортированных
		HasCache:     true,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent, model.BillingAPIUsage},
	}
}

func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source
	root := paths.CodexSessions
	if root == "" {
		return out, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && filepath.Ext(path) == ".jsonl" {
			out = append(out, adapter.Source{Path: path})
		}
		return nil
	})
	return out, err
}

type rolloutLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type sessionMeta struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
}

type tokenUsage struct {
	Input     int64 `json:"input_tokens"`
	Cached    int64 `json:"cached_input_tokens"`
	Output    int64 `json:"output_tokens"`
	Reasoning int64 `json:"reasoning_output_tokens"`
	Total     int64 `json:"total_tokens"`
}

type tokenCountPayload struct {
	Info struct {
		Last  tokenUsage `json:"last_token_usage"`
		Total tokenUsage `json:"total_token_usage"`
	} `json:"info"`
}

// Collect агрегирует одну сессию (один rollout-файл) в одно событие.
// last_token_usage — дельты (суммируем), total_token_usage.total_tokens — кумулятив (берём последний). §8.1
func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	f, err := os.Open(src.Path)
	if err != nil {
		return adapter.Cursor{}, err
	}
	defer f.Close()

	var meta sessionMeta
	var haveMeta bool
	var tk model.Tokens
	var lastTotal int64
	var sawTokens bool

	r := bufio.NewReader(f)
	for {
		raw, readErr := r.ReadBytes('\n')
		line := bytes.TrimRight(raw, "\r\n")
		if len(line) > 0 {
			var rl rolloutLine
			if err := json.Unmarshal(line, &rl); err != nil {
				quarantine(append([]byte(nil), line...), err)
			} else {
				if rl.Type == "session_meta" {
					if json.Unmarshal(rl.Payload, &meta) == nil {
						haveMeta = true
					}
				} else {
					// token_count — это тип ВНУТРИ payload (event_msg / response_item и т.п.)
					var pt struct {
						Type string `json:"type"`
					}
					if json.Unmarshal(rl.Payload, &pt) == nil && pt.Type == "token_count" {
						var tc tokenCountPayload
						if json.Unmarshal(rl.Payload, &tc) == nil {
							tk.Input += tc.Info.Last.Input
							tk.Output += tc.Info.Last.Output
							tk.CacheRead += tc.Info.Last.Cached
							tk.Reasoning += tc.Info.Last.Reasoning
							if tc.Info.Total.Total > 0 {
								lastTotal = tc.Info.Total.Total
							}
							sawTokens = true
						}
					}
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return adapter.Cursor{}, readErr
		}
	}

	if !haveMeta && !sawTokens {
		return adapter.Cursor{}, nil // нечего эмитить
	}
	tk.Total = lastTotal
	eventID := meta.ID
	if eventID == "" {
		// fallback: имя файла без расширения как стабильный ключ
		eventID = strings.TrimSuffix(filepath.Base(src.Path), ".jsonl")
	}
	ts, _ := time.Parse(time.RFC3339, meta.Timestamp)
	emit(model.Event{
		EventID:     eventID,
		Tool:        model.ToolCodex,
		TS:          ts,
		Model:       "unknown", // локально модель отсутствует; обогащение из threads — позже
		BillingMode: model.BillingFlatEquivalent,
		Tokens:      tk,
		SessionID:   meta.ID,
		ProjectKey:  meta.CWD,
	})
	return adapter.Cursor{}, nil
}
