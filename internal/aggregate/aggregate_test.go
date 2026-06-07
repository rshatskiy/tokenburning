package aggregate

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/store"
)

func seed(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := time.Now().UTC()
	evs := []model.Event{
		{EventID: "a", Tool: model.ToolClaudeCode, TS: now, Model: "claude-opus-4-8",
			BillingMode: model.BillingFlatEquivalent,
			Cost:        model.Cost{Amount: 10, Currency: "USD", Basis: model.BasisActual},
			Tokens:      model.Tokens{Input: 100}, SessionID: "s1", ProjectKey: "/Users/secret/topsecret-proj"},
		{EventID: "b", Tool: model.ToolCodex, TS: now, Model: "unknown",
			BillingMode: model.BillingFlatEquivalent,
			Cost:        model.Cost{Amount: 0, Currency: "USD", Basis: model.BasisEstimated},
			Tokens:      model.Tokens{Total: 500}, SessionID: "s2", ProjectKey: "/Users/secret/other"},
	}
	if err := db.Insert(evs); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBuildBreadthExcludesProjectAndBucketsUnknown(t *testing.T) {
	db := seed(t)
	p, err := Build(db, []string{"breadth"}, "all")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.Breadth == nil {
		t.Fatal("breadth nil")
	}
	raw, _ := json.Marshal(p)
	js := string(raw)
	// КРИТИЧНО: project_key никогда не попадает в payload
	if strings.Contains(js, "secret") || strings.Contains(js, "topsecret") || strings.Contains(js, "project_key") || strings.Contains(js, "/Users/") {
		t.Fatalf("payload содержит идентифицирующий project_key:\n%s", js)
	}
	// неизвестная модель забакечена в "other"
	foundOther := false
	for _, m := range p.Breadth.ByModel {
		if m.Model == "unknown" || m.Model == "" {
			t.Fatalf("сырое имя модели в payload: %q", m.Model)
		}
		if m.Model == "other" {
			foundOther = true
		}
	}
	if !foundOther {
		t.Fatal("неизвестная модель должна быть в 'other'")
	}
}

func TestBuildDepthOnlyMedians(t *testing.T) {
	db := seed(t)
	p, err := Build(db, []string{"depth"}, "all")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.Breadth != nil {
		t.Fatal("breadth не запрашивался — должен быть nil")
	}
	if p.Depth == nil {
		t.Fatal("depth nil")
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "/Users/") {
		t.Fatalf("depth payload содержит проект:\n%s", string(raw))
	}
}

func TestBuildNoCategoriesEmpty(t *testing.T) {
	db := seed(t)
	p, err := Build(db, nil, "all")
	if err != nil {
		t.Fatal(err)
	}
	if p.Breadth != nil || p.Depth != nil {
		t.Fatal("без категорий payload пуст")
	}
}
