// Package quality считает метрики качества работы модели из tool_use-блоков,
// уже сохранённых в extra_raw: one-shot rate и retry по файлам.
// Retry (определение как у file-aware конкурентов): тот же файл редактируется
// повторно после shell-команды между правками (Edit foo → Bash → Edit foo).
package quality

import (
	"encoding/json"
	"sort"

	"github.com/rshatskiy/tokenburning/internal/store"
)

// ModelQuality — агрегат по модели за период.
type ModelQuality struct {
	Model      string   `json:"model"`
	EditTurns  int      `json:"editTurns"`  // правок файлов всего
	Retries    int      `json:"retries"`    // правок, оказавшихся повтором после Bash
	OneShotPct float64  `json:"oneShotPct"` // (EditTurns-Retries)/EditTurns*100
	Sessions   int      `json:"sessions"`
	DeltaPct   *float64 `json:"deltaPct,omitempty"` // изменение one-shot к прошлому окну, п.п.
}

type rawMsg struct {
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				FilePath string `json:"file_path"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

// Compute агрегирует события (отсортированные по session, ts) в метрики по моделям.
func Compute(rows []store.RawToolEvent) []ModelQuality {
	type acc struct {
		edits, retries int
		sessions       map[string]bool
	}
	byModel := map[string]*acc{}
	// состояние внутри сессии: для каждого файла — была ли Bash-команда после
	// последней правки этого файла
	var curSession string
	lastEdit := map[string]int{} // file → индекс последней правки (= номер шага)
	bashAfter := map[string]bool{}
	step := 0

	for _, r := range rows {
		if r.SessionID != curSession {
			curSession = r.SessionID
			lastEdit = map[string]int{}
			bashAfter = map[string]bool{}
			step = 0
		}
		var m rawMsg
		if json.Unmarshal(r.Raw, &m) != nil {
			continue
		}
		a := byModel[r.Model]
		if a == nil {
			a = &acc{sessions: map[string]bool{}}
			byModel[r.Model] = a
		}
		a.sessions[r.SessionID] = true
		for _, c := range m.Message.Content {
			if c.Type != "tool_use" {
				continue
			}
			step++
			switch c.Name {
			case "Bash":
				for f := range lastEdit {
					bashAfter[f] = true
				}
			case "Edit", "Write", "MultiEdit", "NotebookEdit":
				f := c.Input.FilePath
				if f == "" {
					continue
				}
				a.edits++
				if _, seen := lastEdit[f]; seen && bashAfter[f] {
					a.retries++ // тот же файл после shell-команды — повторная попытка
				}
				lastEdit[f] = step
				bashAfter[f] = false
			}
		}
	}

	var out []ModelQuality
	for model, a := range byModel {
		if a.edits == 0 {
			continue
		}
		out = append(out, ModelQuality{
			Model: model, EditTurns: a.edits, Retries: a.retries,
			OneShotPct: float64(a.edits-a.retries) / float64(a.edits) * 100,
			Sessions:   len(a.sessions),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EditTurns > out[j].EditTurns })
	return out
}
