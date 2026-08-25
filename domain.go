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
