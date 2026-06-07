package cursor

import (
	"database/sql"
	"encoding/json"
	"os"
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

func (a *Adapter) Name() model.Tool { return model.ToolCursor }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		HasTokens:    model.FidelityPartial, // локально часто 0; реальные данные за API (§6.2)
		HasCache:     false,
		HasSessions:  true,
		BillingModes: []model.BillingMode{model.BillingFlatEquivalent},
	}
}

// Discover находит state.vscdb: глобальный и по одному на workspace.
func (a *Adapter) Discover(paths platform.Paths) ([]adapter.Source, error) {
	var out []adapter.Source
	root := paths.CursorStorage
	if root == "" {
		return out, nil
	}
	global := filepath.Join(root, "User", "globalStorage", "state.vscdb")
	if _, err := os.Stat(global); err == nil {
		out = append(out, adapter.Source{Path: global})
	}
	wsRoot := filepath.Join(root, "User", "workspaceStorage")
	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return out, nil // нет workspaceStorage — ок
	}
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(wsRoot, e.Name(), "state.vscdb")
			if _, err := os.Stat(p); err == nil {
				out = append(out, adapter.Source{Path: p})
			}
		}
	}
	return out, nil
}

type bubble struct {
	Type      int    `json:"type"` // 1=user, 2=assistant
	CreatedAt string `json:"createdAt"`
	Tokens    struct {
		Input  int64 `json:"inputTokens"`
		Output int64 `json:"outputTokens"`
	} `json:"tokenCount"`
	ModelInfo struct {
		ModelName string `json:"modelName"`
	} `json:"modelInfo"`
}

func (a *Adapter) Collect(src adapter.Source, _ adapter.Cursor, emit adapter.EmitFunc, quarantine adapter.QuarantineFunc) (adapter.Cursor, error) {
	// read-only; immutable, чтобы не трогать WAL чужого приложения.
	db, err := sql.Open("sqlite", "file:"+src.Path+"?mode=ro&immutable=1")
	if err != nil {
		return adapter.Cursor{}, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'bubbleId:%'`)
	if err != nil {
		// нет таблицы cursorDiskKV (часть workspace-БД) — это не ошибка, просто пусто.
		if strings.Contains(err.Error(), "no such table") {
			return adapter.Cursor{}, nil
		}
		return adapter.Cursor{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var val []byte
		if err := rows.Scan(&key, &val); err != nil {
			return adapter.Cursor{}, err
		}
		var b bubble
		if err := json.Unmarshal(val, &b); err != nil {
			quarantine(append([]byte(nil), val...), err)
			continue
		}
		if b.Type != 2 { // только assistant
			continue
		}
		ts, _ := time.Parse(time.RFC3339, b.CreatedAt)
		modelName := b.ModelInfo.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		emit(model.Event{
			EventID:     key, // bubbleId:<composerId>:<messageId> — уникален и стабилен
			Tool:        model.ToolCursor,
			TS:          ts,
			Model:       modelName,
			BillingMode: model.BillingFlatEquivalent,
			Tokens:      model.Tokens{Input: b.Tokens.Input, Output: b.Tokens.Output},
			SessionID:   composerID(key),
		})
	}
	return adapter.Cursor{}, rows.Err()
}

// composerID извлекает <composerId> из ключа bubbleId:<composerId>:<messageId>.
func composerID(key string) string {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
