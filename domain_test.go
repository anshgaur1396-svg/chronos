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

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		req        ProrationRequest
		wantErrMsg string
	}{
		{
			name: "valid request with normal values",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "",
		},
		{
			name: "valid request with zero prices (free plans)",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Free", PricePaisa: 0},
				NewPlan:        Plan{Name: "Pro", PricePaisa: 0},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "",
		},
		{
			name: "empty subscription ID",
			req: ProrationRequest{
				SubscriptionID: "",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "subscription_id is required",
		},
		{
			name: "empty old plan name",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "old_plan.name is required",
		},
		{
			name: "empty new plan name",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "new_plan.name is required",
		},
		{
			name: "total days is zero",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      0,
				UsedDays:       0,
			},
			wantErrMsg: "total_days must be greater than 0",
		},
		{
			name: "total days is negative",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      -5,
				UsedDays:       0,
			},
			wantErrMsg: "total_days must be greater than 0",
		},
		{
			name: "used days is negative",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       -1,
			},
			wantErrMsg: "used_days cannot be negative",
		},
		{
			name: "used days exceeds total days",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       31,
			},
			wantErrMsg: "used_days cannot exceed total_days",
		},
		{
			name: "negative total days and positive used days proves validation order",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      -5,
				UsedDays:       3,
			},
			wantErrMsg: "total_days must be greater than 0",
		},
		{
			name: "negative old plan price",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: -100},
				NewPlan:        Plan{Name: "New", PricePaisa: 20000},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "old_plan.price_paisa cannot be negative",
		},
		{
			name: "negative new plan price",
			req: ProrationRequest{
				SubscriptionID: "sub_123",
				OldPlan:        Plan{Name: "Old", PricePaisa: 10000},
				NewPlan:        Plan{Name: "New", PricePaisa: -100},
				TotalDays:      30,
				UsedDays:       10,
			},
			wantErrMsg: "new_plan.price_paisa cannot be negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.req)

			if tt.wantErrMsg == "" {
				if err != nil {
					t.Errorf("validate() expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("validate() expected error %q, got nil", tt.wantErrMsg)
				} else if err.Error() != tt.wantErrMsg {
					t.Errorf("validate() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}
