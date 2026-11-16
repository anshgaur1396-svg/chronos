package chronos

import (
	"errors"
	"math/big"
)

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

// roundHalfUp assumes valid nonnegative prices, 0 <= remainingDays <= totalDays,
// and totalDays > 0. big.Int prevents intermediate multiplication overflow.
func roundHalfUp(pricePaisa, remainingDays, totalDays int64) int64 {
	numerator := new(big.Int).Mul(
		big.NewInt(pricePaisa),
		big.NewInt(remainingDays),
	)

	denominator := big.NewInt(totalDays)

	quotient := new(big.Int)
	remainder := new(big.Int)

	quotient.QuoRem(numerator, denominator, remainder)

	// Round half up when 2 * remainder >= denominator.
	twiceRemainder := new(big.Int).Lsh(remainder, 1)

	if twiceRemainder.Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}

	return quotient.Int64()
}

func calculateProration(inputRequest ProrationRequest) ProrationResponse {
	remainingDays := inputRequest.TotalDays - inputRequest.UsedDays

	oldRemainingPaisa := roundHalfUp(
		inputRequest.OldPlan.PricePaisa,
		remainingDays,
		inputRequest.TotalDays,
	)

	newRemainingPaisa := roundHalfUp(
		inputRequest.NewPlan.PricePaisa,
		remainingDays,
		inputRequest.TotalDays,
	)

	adjustmentPaisa := newRemainingPaisa - oldRemainingPaisa

	adjustmentStatus := statusOwes

	if adjustmentPaisa < 0 {
		adjustmentStatus = statusCredit
	} else if adjustmentPaisa == 0 {
		adjustmentStatus = statusNone
	}

	return ProrationResponse{
		SubscriptionID:    inputRequest.SubscriptionID,
		OldRemainingPaisa: oldRemainingPaisa,
		NewRemainingPaisa: newRemainingPaisa,
		AdjustmentPaisa:   adjustmentPaisa,
		AdjustmentStatus:  adjustmentStatus,
	}
}

// validate checks the order of the request made
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
