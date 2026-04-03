package services

import "testing"

func TestBattleTimeToSeconds(t *testing.T) {
	cases := []struct {
		name string
		in   int64
		out  int64
	}{
		{name: "zero", in: 0, out: 0},
		{name: "seconds", in: 1775194980, out: 1775194980},
		{name: "millis", in: 1775191526403, out: 1775191526},
	}

	for _, c := range cases {
		if got := battleTimeToSeconds(c.in); got != c.out {
			t.Fatalf("%s: battleTimeToSeconds(%d)=%d want=%d", c.name, c.in, got, c.out)
		}
	}
}
