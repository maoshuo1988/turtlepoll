package biztime

import (
	"testing"
	"time"
)

func TestDayNameCST(t *testing.T) {
	// 2026-04-08 00:00:01 +08:00
	tm := time.Date(2026, 4, 8, 0, 0, 1, 0, time.FixedZone("UTC+8", 8*60*60))
	if got := DayNameCST(tm); got != 20260408 {
		t.Fatalf("DayNameCST got=%d", got)
	}
	if got := DateStringCST(tm); got != "2026-04-08" {
		t.Fatalf("DateStringCST got=%s", got)
	}
}

func TestNextMidnightCST(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)

	// 同一天任意时刻 => 次日 0 点
	in1 := time.Date(2026, 4, 8, 11, 22, 33, 0, loc)
	got1 := NextMidnightCST(in1)
	want1 := time.Date(2026, 4, 9, 0, 0, 0, 0, loc)
	if !got1.Equal(want1) {
		t.Fatalf("NextMidnightCST got=%v want=%v", got1, want1)
	}

	// 边界：当天 0 点也应返回次日 0 点
	in2 := time.Date(2026, 4, 8, 0, 0, 0, 0, loc)
	got2 := NextMidnightCST(in2)
	want2 := time.Date(2026, 4, 9, 0, 0, 0, 0, loc)
	if !got2.Equal(want2) {
		t.Fatalf("NextMidnightCST boundary got=%v want=%v", got2, want2)
	}
}
