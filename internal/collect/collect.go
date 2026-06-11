package collect

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/rshatskiy/tokenburning/internal/adapter"
	"github.com/rshatskiy/tokenburning/internal/adapter/claudecode"
	"github.com/rshatskiy/tokenburning/internal/adapter/cline"
	"github.com/rshatskiy/tokenburning/internal/adapter/codex"
	"github.com/rshatskiy/tokenburning/internal/adapter/copilot"
	"github.com/rshatskiy/tokenburning/internal/adapter/cursor"
	"github.com/rshatskiy/tokenburning/internal/adapter/gemini"
	"github.com/rshatskiy/tokenburning/internal/adapter/opencode"
	"github.com/rshatskiy/tokenburning/internal/model"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/store"
)

// Result — итог одного прохода сбора.
type Result struct {
	Collected   int
	Quarantined int
	Skipped     int // источники, пропущенные по курсору (не изменились с прошлого прохода)
	// SampleErrors — первые несколько ошибок карантина: без них сломавшийся
	// формат логов виден только как молча растущий счётчик.
	SampleErrors []string
}

const maxSampleErrors = 5

// headerHash — sha256 первых min(n, 4096) байт файла: дешёвый детект перезаписи
// файла на том же inode. Пустая строка = посчитать не удалось (хэш не совпадёт —
// безопасный полный перечит).
func headerHash(path string, n int64) string {
	if n <= 0 {
		return ""
	}
	if n > 4096 {
		n = 4096
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, n)
	if _, err := io.ReadFull(f, buf); err != nil {
		return ""
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// Adapters возвращает реестр всех адаптеров.
func Adapters() []adapter.Adapter {
	return []adapter.Adapter{
		claudecode.New(), codex.New(), cursor.New(),
		gemini.New(), copilot.New(), opencode.New(), cline.New(),
	}
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
	quar := func(raw []byte, err error) {
		res.Quarantined++
		if err != nil && len(res.SampleErrors) < maxSampleErrors {
			res.SampleErrors = append(res.SampleErrors, err.Error())
		}
	}

	// Курсоры — оптимизация: их отсутствие/потеря не ломает сбор (вставка
	// идемпотентна по event_id), просто всё перечитается заново.
	prev, err := db.SourceCursors()
	if err != nil {
		prev = map[string]store.SourceCursor{}
	}
	var advanced []store.SourceCursor

	for _, ad := range Adapters() {
		srcs, derr := ad.Discover(paths)
		if derr != nil {
			continue
		}
		for i, s := range srcs {
			if progress != nil {
				progress(string(ad.Name()), i+1, len(srcs))
			}
			fi, statErr := os.Stat(s.Path)
			fid, _ := platform.Stat(s.Path)
			var cur adapter.Cursor
			if statErr == nil && !fid.IsZero() {
				if pc, ok := prev[s.Path]; ok && pc.FileID == fid {
					if pc.Size == fi.Size() && pc.MTime == fi.ModTime().UnixNano() {
						res.Skipped++ // тот же файл, ничего не дописано
						continue
					}
					// Append-кейс: размер вырос И начало файла не изменилось.
					// Хэш заголовка отличает дозапись от перезаписи «на том же
					// inode» (компактизация) — по FileID их не различить.
					if fi.Size() >= pc.Size && pc.HeaderHash != "" && headerHash(s.Path, pc.Size) == pc.HeaderHash {
						cur.Offset = pc.Offset
					}
					// иначе — полный перечит (Offset=0), идемпотентно по event_id
				}
			}
			nc, cerr := ad.Collect(s, cur, emit, quar)
			if cerr != nil || statErr != nil {
				continue // курсор не двигаем — следующий проход попробует снова
			}
			advanced = append(advanced, store.SourceCursor{
				Path: s.Path, FileID: fid,
				// Size/MTime — со stat ДО чтения: выросший за время чтения файл
				// просто перечитается со старого offset (идемпотентно).
				Size: fi.Size(), MTime: fi.ModTime().UnixNano(), Offset: nc.Offset,
				HeaderHash: headerHash(s.Path, fi.Size()),
			})
		}
	}
	res.Collected = len(batch)
	if err := db.Insert(batch); err != nil {
		return res, err // курсоры НЕ сохраняем: события не легли — перечитаем
	}
	_ = db.SaveSourceCursors(advanced) // best-effort: потеря курсора = лишний перечит, не потеря данных
	return res, nil
}
