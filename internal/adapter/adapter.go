package adapter

import (
	"github.com/lens/lens/internal/model"
	"github.com/lens/lens/internal/platform"
)

// Source — конкретный источник данных (файл лога или БД) на этой машине.
type Source struct {
	Path string
}

// Cursor — состояние инкрементального чтения append-источника.
// Для не-append адаптеров может игнорироваться.
type Cursor struct {
	FileID     platform.FileID
	Size       int64
	HeaderHash string
	Offset     int64
}

// EmitFunc принимает нормализованное событие.
type EmitFunc func(model.Event)

// QuarantineFunc принимает непарсимую запись (сырьё + ошибка) — не роняя сбор.
type QuarantineFunc func(raw []byte, err error)

// Adapter изолирует один инструмент за общим интерфейсом.
type Adapter interface {
	Name() model.Tool
	Capabilities() model.Capabilities
	Discover(p platform.Paths) ([]Source, error)
	// Collect выдаёт новые события начиная с cursor. Обязан быть идемпотентным
	// (event_id стабилен) и толерантным: битая запись идёт в quarantine, не в panic.
	Collect(src Source, cursor Cursor, emit EmitFunc, quarantine QuarantineFunc) (Cursor, error)
}
