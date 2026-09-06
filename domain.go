package chronos

const (
	statusOwes   = "owes"
	statusCredit = "credit"
	statusNone   = "none"
)

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

func calculateProration(inputRequest ProrationRequest) ProrationResponse {
	var outputResponse ProrationResponse

	remainingDays := inputRequest.TotalDays - inputRequest.UsedDays

	oldRemainingPaisa := roundHalfUp(inputRequest.OldPlan.PricePaisa, remainingDays, inputRequest.TotalDays)
	newRemainingPaisa := roundHalfUp(inputRequest.NewPlan.PricePaisa, remainingDays, inputRequest.TotalDays)

	adjustmentPaisa := newRemainingPaisa - oldRemainingPaisa

	var adjustmentStatus string

	if adjustmentPaisa < 0 {
		adjustmentStatus = statusCredit
	} else if adjustmentPaisa == 0 {
		adjustmentStatus = statusNone
	} else {
		adjustmentStatus = statusOwes
	}

	outputResponse.SubscriptionID = inputRequest.SubscriptionID
	outputResponse.OldRemainingPaisa = oldRemainingPaisa
	outputResponse.NewRemainingPaisa = newRemainingPaisa
	outputResponse.AdjustmentPaisa = adjustmentPaisa
	outputResponse.AdjustmentStatus = adjustmentStatus

	return outputResponse
}
