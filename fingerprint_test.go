package chronos

import "testing"

func TestRequestFingerprintChangesPerField(t *testing.T) {
	base := ProrationRequest{
		SubscriptionID: "sub_123",
		OldPlan: Plan{
			Name:       "Basic",
			PricePaisa: 10000,
		},
		NewPlan: Plan{
			Name:       "Pro",
			PricePaisa: 20000,
		},
		TotalDays: 30,
		UsedDays:  10,
	}

	baseFingerprint, err := requestFingerprint(base)
	if err != nil {
		t.Fatalf("fingerprint base request: %v", err)
	}

	// Fingerprinting must be deterministic.
	repeatedFingerprint, err := requestFingerprint(base)
	if err != nil {
		t.Fatalf("fingerprint base request again: %v", err)
	}

	if baseFingerprint != repeatedFingerprint {
		t.Fatal("same request produced different fingerprints")
	}

	tests := []struct {
		name   string
		modify func(*ProrationRequest)
	}{
		{
			name: "subscription_id",
			modify: func(req *ProrationRequest) {
				req.SubscriptionID = "sub_456"
			},
		},
		{
			name: "old_plan.name",
			modify: func(req *ProrationRequest) {
				req.OldPlan.Name = "Starter"
			},
		},
		{
			name: "old_plan.price_paisa",
			modify: func(req *ProrationRequest) {
				req.OldPlan.PricePaisa++
			},
		},
		{
			name: "new_plan.name",
			modify: func(req *ProrationRequest) {
				req.NewPlan.Name = "Enterprise"
			},
		},
		{
			name: "new_plan.price_paisa",
			modify: func(req *ProrationRequest) {
				req.NewPlan.PricePaisa++
			},
		},
		{
			name: "total_days",
			modify: func(req *ProrationRequest) {
				req.TotalDays++
			},
		},
		{
			name: "used_days",
			modify: func(req *ProrationRequest) {
				req.UsedDays++
			},
		},
		{
			name: "swap_old_and_new_plans",
			modify: func(req *ProrationRequest) {
				req.OldPlan, req.NewPlan = req.NewPlan, req.OldPlan
			},
		},
	}

	// Track fingerprints to detect collisions between distinct test requests.
	seen := map[string]string{
		baseFingerprint: "base",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := base
			tt.modify(&modified)

			fingerprint, err := requestFingerprint(modified)
			if err != nil {
				t.Fatalf("fingerprint modified request: %v", err)
			}

			if fingerprint == baseFingerprint {
				t.Fatalf("%s change did not change the fingerprint", tt.name)
			}

			if previous, exists := seen[fingerprint]; exists {
				t.Errorf(
					"fingerprint collision: %s and %s produced the same fingerprint",
					previous,
					tt.name,
				)
			} else {
				seen[fingerprint] = tt.name
			}
		})
	}
}
