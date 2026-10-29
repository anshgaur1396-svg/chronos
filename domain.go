package chronos

import "errors"

const (
	statusOwes   = "owes"
	statusCredit = "credit"
	statusNone   = "none"
)

type Plan struct {
	Name       string `json:"name"`
	PricePaisa int64  `json:"price_paisa"`
}

type ProrationRequest struct {
	SubscriptionID string `json:"subscription_id"`
	OldPlan        Plan   `json:"old_plan"`
	NewPlan        Plan   `json:"new_plan"`
	TotalDays      int64  `json:"total_days"`
	UsedDays       int64  `json:"used_days"`
}

type ProrationResponse struct {
	SubscriptionID    string `json:"subscription_id"`
	OldRemainingPaisa int64  `json:"old_remaining_paisa"`
	NewRemainingPaisa int64  `json:"new_remaining_paisa"`
	AdjustmentPaisa   int64  `json:"adjustment_paisa"`
	AdjustmentStatus  string `json:"adjustment_status"`
}

// roundHalfUp assumes totalDays > 0; callers must validate this beforehand.
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

func validate(req ProrationRequest) error {
	if req.SubscriptionID == "" {
		return errors.New("subscription_id is required")
	}
	if req.OldPlan.Name == "" {
		return errors.New("old_plan.name is required")
	}
	if req.TotalDays <= 0 {
		return errors.New("total_days must be greater than 0")
	}
	if req.UsedDays < 0 {
		return errors.New("used_days cannot be negative")
	}
	if req.UsedDays > req.TotalDays {
		return errors.New("used_days cannot exceed total_days")
	}
	if req.OldPlan.PricePaisa < 0 {
		return errors.New("old_plan.price_paisa cannot be negative")
	}
	if req.NewPlan.Name == "" {
		return errors.New("new_plan.name is required")
	}
	if req.NewPlan.PricePaisa < 0 {
		return errors.New("new_plan.price_paisa cannot be negative")
	}

	return nil
}
