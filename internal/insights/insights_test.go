package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func ev(id, sess string, ts time.Time, input, cacheRead int64, cost float64) model.Event {
	return model.Event{
		EventID: id, Tool: model.ToolClaudeCode, TS: ts, Model: "claude-opus-4-8",
		BillingMode: model.BillingFlatEquivalent,
		Cost:        model.Cost{Amount: cost, Currency: "USD", Basis: model.BasisActual, PricingVersion: "t"},
		Tokens:      model.Tokens{Input: input, CacheRead: cacheRead},
		SessionID:   sess, ProjectKey: "/p/alpha",
	}
}

func testDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Падение кэш-хита 80% → 30% неделя к неделе детектируется.
func TestCacheDrop(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	var batch []model.Event
	// прошлая неделя: 80% кэш-хит (2M cache / 0.5M input)
	batch = append(batch, ev("p1", "s1", now.AddDate(0, 0, -10), 500_000, 2_000_000, 1))
	// эта неделя: 30% (0.7M input / 0.3M cache)
	batch = append(batch, ev("c1", "s2", now.AddDate(0, 0, -2), 700_000, 300_000, 1))
	if err := db.Insert(batch); err != nil {
		t.Fatal(err)
	}
	got := cacheDrops(db, now)
	if len(got) != 1 || got[0].Kind != "cache_drop" {
		t.Fatalf("ожидался cache_drop, получено %+v", got)
	}
}

// Сессия в 10× медианы и дороже $10 попадает в выбросы.
func TestExpensiveSession(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	var batch []model.Event
	for i := 0; i < 9; i++ {
		batch = append(batch, ev(string(rune('a'+i)), "s"+string(rune('a'+i)), now.AddDate(0, 0, -20), 1000, 0, 2))
	}
	batch = append(batch, ev("big", "s-big", now.AddDate(0, 0, -1), 1000, 0, 25))
	if err := db.Insert(batch); err != nil {
		t.Fatal(err)
	}
	got := expensiveSessions(db, now)
	if len(got) != 1 || got[0].Data["session"] != "s-big" {
		t.Fatalf("ожидался выброс s-big, получено %+v", got)
	}
}

// Модель с токенами и нулевой стоимостью → совет про alias.
func TestUnpricedModel(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	e := ev("u1", "su", now.AddDate(0, 0, -3), 2_000_000, 0, 0)
	e.Model = "my-proxy-model"
	if err := db.Insert([]model.Event{e}); err != nil {
		t.Fatal(err)
	}
	got := unpricedModels(db, now)
	if len(got) != 1 || got[0].Data["model"] != "my-proxy-model" {
		t.Fatalf("ожидался unpriced_model, получено %+v", got)
	}
}

// Раздутый CLAUDE.md и 6+ MCP-серверов дают по сигналу.
func TestClaudeSetup(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 25*1024)
	if err := os.WriteFile(filepath.Join(home, ".claude", "CLAUDE.md"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	mcp := map[string]any{"mcpServers": map[string]any{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1, "f": 1}}
	b, _ := json.Marshal(mcp)
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	got := claudeSetup(home)
	kinds := map[string]bool{}
	for _, g := range got {
		kinds[g.Kind] = true
	}
	if !kinds["claude_md_big"] || !kinds["mcp_many"] {
		t.Fatalf("ожидались claude_md_big и mcp_many, получено %+v", got)
	}
}
