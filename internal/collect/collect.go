package collect

import (
	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/adapter/claudecode"
	"github.com/rshatskiy/tokenburning/internal/adapter/codex"
	"github.com/rshatskiy/tokenburning/internal/adapter/cursor"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// Result — итог одного прохода сбора.
type Result struct {
	Collected   int
	Quarantined int
}

// Adapters возвращает реестр всех адаптеров.
func Adapters() []adapter.Adapter {
	return []adapter.Adapter{claudecode.New(), codex.New(), cursor.New()}
}

// Progress опционально вызывается на каждый источник.
type Progress func(tool string, i, n int)

// Run discover→collect→price→store по всем адаптерам, идемпотентно вставляет батч.
func Run(db *store.DB, cat *pricing.Catalog, paths platform.Paths, progress Progress) (Result, error) {
	var res Result
	var batch []model.Event
	emit := func(e model.Event) {
		e.Cost = cat.Cost(e.Model, e.Tokens)
		batch = append(batch, e)
	}
	quar := func(raw []byte, err error) { res.Quarantined++ }

	for _, ad := range Adapters() {
		srcs, derr := ad.Discover(paths)
		if derr != nil {
			continue
		}
		for i, s := range srcs {
			if progress != nil {
				progress(string(ad.Name()), i+1, len(srcs))
			}
			_, _ = ad.Collect(s, adapter.Cursor{}, emit, quar)
		}
	}
	res.Collected = len(batch)
	if err := db.Insert(batch); err != nil {
		return res, err
	}
	return res, nil
}
