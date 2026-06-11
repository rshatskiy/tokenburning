package quality

import (
	"testing"

	"github.com/rshatskiy/tokenburning/internal/store"
)

func raw(blocks string) []byte {
	return []byte(`{"message":{"content":[` + blocks + `]}}`)
}

const editFoo = `{"type":"tool_use","name":"Edit","input":{"file_path":"/p/foo.go"}}`
const editBar = `{"type":"tool_use","name":"Write","input":{"file_path":"/p/bar.go"}}`
const bash = `{"type":"tool_use","name":"Bash","input":{}}`

// Edit foo → Bash → Edit foo = retry; правка другого файла между шагами — нет.
func TestRetryDetection(t *testing.T) {
	rows := []store.RawToolEvent{
		{SessionID: "s1", Model: "m", TS: 1, Raw: raw(editFoo)},
		{SessionID: "s1", Model: "m", TS: 2, Raw: raw(bash)},
		{SessionID: "s1", Model: "m", TS: 3, Raw: raw(editFoo)}, // retry
		{SessionID: "s1", Model: "m", TS: 4, Raw: raw(editBar)}, // не retry (другой файл)
		{SessionID: "s1", Model: "m", TS: 5, Raw: raw(editFoo)}, // не retry (без Bash между)
	}
	got := Compute(rows)
	if len(got) != 1 {
		t.Fatalf("ожидалась 1 модель: %+v", got)
	}
	q := got[0]
	if q.EditTurns != 4 || q.Retries != 1 {
		t.Fatalf("edits=%d retries=%d, ожидалось 4/1", q.EditTurns, q.Retries)
	}
	if q.OneShotPct != 75 {
		t.Fatalf("OneShotPct=%v, ожидалось 75", q.OneShotPct)
	}
}

// Состояние не утекает между сессиями.
func TestSessionIsolation(t *testing.T) {
	rows := []store.RawToolEvent{
		{SessionID: "s1", Model: "m", TS: 1, Raw: raw(editFoo)},
		{SessionID: "s1", Model: "m", TS: 2, Raw: raw(bash)},
		{SessionID: "s2", Model: "m", TS: 3, Raw: raw(editFoo)}, // новая сессия — не retry
	}
	q := Compute(rows)[0]
	if q.Retries != 0 || q.Sessions != 2 {
		t.Fatalf("retries=%d sessions=%d, ожидалось 0/2", q.Retries, q.Sessions)
	}
}
