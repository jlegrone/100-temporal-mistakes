package payments

import (
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type PurchaseItemRequest struct {
	SKU string
}

type PurchaseItemResponse struct {
	Payment *ChargePaymentResponse
}

// PurchaseItem looks up the charge for the requested SKU and invokes the
// ChargePayment activity with the timeout/retry configuration developed in
// "Activities: Weathering System Outages":
//
//   - StartToClose 30s caps a single attempt.
//   - ScheduleToClose 1h caps total retry duration through a worst-case outage.
//   - MaximumAttempts is left unset so retries continue until ScheduleToClose
//     is reached; the policy only tunes backoff.
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
	// Decide the amount to charge based on SKU. Normally this might be a database lookup instead of a side effect.
	chargeRequest, err := workflowhelpers.SideEffect(ctx, func(ctx workflow.Context) *ChargePaymentRequest {
		return generateChargeRequestForSKU(req.SKU)
	})
	if err != nil {
		return nil, err
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			// MaximumAttempts intentionally unset: allow unlimited attempts
			// until the ScheduleToClose timeout is reached.
		},
	})

	resp, err := workflowhelpers.AwaitActivity(ctx, w.ChargePayment, *chargeRequest)
	if err != nil {
		return nil, err
	}

	return &PurchaseItemResponse{
		Payment: resp,
	}, nil
}

func generateChargeRequestForSKU(SKU string) *ChargePaymentRequest {
	switch SKU {
	case "1234":
		return &ChargePaymentRequest{
			AmountCents: 100,
		}
	default:
		return nil
	}
}
