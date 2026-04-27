package payments

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ChargePaymentWorkflow invokes the ChargePayment activity with the
// timeout/retry configuration developed in "Activities: Weathering System
// Outages":
//
//   - StartToClose 30s caps a single attempt.
//   - ScheduleToClose 1h caps total retry duration through a worst-case outage.
//   - MaximumAttempts is left unset so retries continue until ScheduleToClose
//     is reached; the policy only tunes backoff.
func ChargePaymentWorkflow(ctx workflow.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			// MaximumAttempts intentionally unset: allow unlimited attempts
			// until the ScheduleToClose timeout is reached.
		},
	})

	var w *Worker
	var resp ChargePaymentResponse
	if err := workflow.ExecuteActivity(ctx, w.ChargePayment, req).Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
