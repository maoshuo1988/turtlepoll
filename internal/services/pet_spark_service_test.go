package services

import "testing"

func TestPetSparkServiceComputeSparkRaw(t *testing.T) {
	tests := []struct {
		name   string
		streak int
		want   int64
	}{
		{name: "zero", streak: 0, want: 0},
		{name: "three", streak: 3, want: 25},
		{name: "seven", streak: 7, want: 50},
		{name: "fourteen", streak: 14, want: 75},
		{name: "thirty", streak: 30, want: 95},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PetSparkService.ComputeSparkRaw(tt.streak)
			if got != tt.want {
				t.Fatalf("ComputeSparkRaw(%d) = %d, want %d", tt.streak, got, tt.want)
			}
		})
	}
}

func TestPetSparkServiceApplySparkMultiplier(t *testing.T) {
	params := SparkMultiplierParams{
		Enabled:  true,
		Base:     1.3,
		PerLevel: 0.03,
		Cap:      400,
	}
	extra, final := PetSparkService.ApplySparkMultiplier(50, 10, params)
	if extra != 30 {
		t.Fatalf("extra = %d, want 30", extra)
	}
	if final != 80 {
		t.Fatalf("final = %d, want 80", final)
	}
}

func TestPetSparkServiceApplySparkMultiplierCap(t *testing.T) {
	params := SparkMultiplierParams{
		Enabled:        true,
		BaseMultiplier: 3,
		LevelScale:     0,
		ExtraCapPerDay: 20,
	}
	extra, final := PetSparkService.ApplySparkMultiplier(50, 1, params)
	if extra != 20 {
		t.Fatalf("extra = %d, want 20", extra)
	}
	if final != 70 {
		t.Fatalf("final = %d, want 70", final)
	}
}
