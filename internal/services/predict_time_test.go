package services

import "testing"

func TestPredictTimestampToSeconds(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want int64
	}{
		{name: "zero", in: 0, want: 0},
		{name: "seconds", in: 1_783_827_000, want: 1_783_827_000},
		{name: "milliseconds", in: 1_783_827_000_123, want: 1_783_827_000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := predictTimestampToSeconds(tc.in)
			if got != tc.want {
				t.Fatalf("predictTimestampToSeconds(%d)=%d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
