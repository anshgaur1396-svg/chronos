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

func TestCalculateProration(t *testing.T) {
	tests := []struct {
		subscriptionID string
		oldPlan        Plan
		newPlan        Plan
		totalDays      int64
		usedDays       int64
		want           ProrationResponse
	}{
		{"sub_111", Plan{"Old", 10000}, Plan{"New", 20000}, 30, 10, ProrationResponse{"sub_111", 6667, 13333, 6666, "owes"}},
		{"sub_112", Plan{"New", 20000}, Plan{"Old", 10000}, 30, 10, ProrationResponse{"sub_112", 13333, 6667, -6666, "credit"}},
		{"sub_113", Plan{"Old", 10000}, Plan{"Old", 10000}, 30, 10, ProrationResponse{"sub_113", 6667, 6667, 0, "none"}},
		{"sub_114", Plan{"Free", 0}, Plan{"Pro", 15000}, 30, 10, ProrationResponse{"sub_114", 0, 10000, 10000, "owes"}},
		{"sub_115", Plan{"Pro", 15000}, Plan{"Free", 0}, 30, 10, ProrationResponse{"sub_115", 10000, 0, -10000, "credit"}},
		{"sub_116", Plan{"Old", 10000}, Plan{"New", 20000}, 30, 30, ProrationResponse{"sub_116", 0, 0, 0, "none"}},
		{"sub_117", Plan{"Old", 10000}, Plan{"New", 20000}, 30, 0, ProrationResponse{"sub_117", 10000, 20000, 10000, "owes"}},
	}
	for _, tt := range tests {
		t.Run(tt.subscriptionID, func(t *testing.T) {
			input := ProrationRequest{
				SubscriptionID: tt.subscriptionID,
				OldPlan:        tt.oldPlan,
				NewPlan:        tt.newPlan,
				TotalDays:      tt.totalDays,
				UsedDays:       tt.usedDays,
			}

			got := calculateProration(input)

			if got != tt.want {
				t.Errorf("calculateProration(...) = %+v, want %+v", got, tt.want)
			}
		})
	}
}
