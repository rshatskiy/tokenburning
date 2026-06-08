package aggregate

import (
	"testing"
	"time"
)

func TestSinceForDays(t *testing.T) {
	loc := time.FixedZone("UTC+3", 3*3600)
	now := time.Date(2026, 6, 8, 14, 33, 0, 0, loc) // середина дня — не должна влиять

	// «7d» = сегодня + 6 предыдущих суток → локальная полночь 2 июня.
	got := SinceForDays(7, now)
	want := time.Date(2026, 6, 2, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("SinceForDays(7) = %v, want %v (локальная полночь, сегодня вкл.)", got, want)
	}

	// «1d» = только сегодня → полночь сегодня.
	if g := SinceForDays(1, now); !g.Equal(time.Date(2026, 6, 8, 0, 0, 0, 0, loc)) {
		t.Fatalf("SinceForDays(1) = %v, want полночь сегодня", g)
	}

	// Привязка к полуночи (время суток now не протекает в результат).
	if g := SinceForDays(30, now); g.Hour() != 0 || g.Minute() != 0 || g.Second() != 0 {
		t.Fatalf("since должно быть в локальную полночь, got %v", g)
	}

	// 0/all → нулевое время («всё»).
	if !SinceForDays(0, now).IsZero() {
		t.Fatal("SinceForDays(0) должно быть нулевым (всё)")
	}
}
