package chronos

import "testing"

// func TestSomething(t *testing.T) { ... }

func TestRoundHalfUp(t *testing.T) {
	tests := []struct {
		name          string
		pricePaisa    int64
		remainingDays int64
		totalDays     int64
		want          int64
	}{
		{"above Half", 6588, 25, 31, 5313},
		{"below Half", 10203, 45, 60, 7652},
		{"exactly Half", 20406, 45, 60, 15305},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundHalfUp(tt.pricePaisa, tt.remainingDays, tt.totalDays)

			if got != tt.want {
				t.Errorf("roundHalfUp(...) = %d, want %d", got, tt.want)
			}
		})
	}
}
