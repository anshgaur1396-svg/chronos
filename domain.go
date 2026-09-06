package chronos

type Plan struct {
	Name       string
	PricePaisa int64
}

type ProrationRequest struct {
	SubscriptionID string
	OldPlan        Plan
	NewPlan        Plan
	TotalDays      int64
	UsedDays       int64
}

type ProrationResponse struct {
	SubscriptionID    string
	OldRemainingPaisa int64
	NewRemainingPaisa int64
	AdjustmentPaisa   int64
	AdjustmentStatus  string
}

// roundHalfUP assumes totalDays > 0; callers must validate this beforehand.
func roundHalfUp(pricePaisa, remainingDays, totalDays int64) int64 {
	numerator := pricePaisa * remainingDays
	quotient := numerator / totalDays
	remainder := numerator % totalDays
	if remainder*2 >= totalDays {
		quotient++
	}

	return quotient
}
