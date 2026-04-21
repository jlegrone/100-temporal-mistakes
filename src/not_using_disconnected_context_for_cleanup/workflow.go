package not_using_disconnected_context_for_cleanup

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ProcessOrder is a long-running activity.
func ProcessOrder(_ context.Context) error { return nil }

// CancelOrder is a cleanup activity.
func CancelOrder(_ context.Context) error { return nil }

// @@@SNIPSTART not-using-disconnected-context-bad

// MyWorkflowV1 tries to run a cleanup activity after cancelation,
// but uses the original (already-canceled) context. The cleanup
// activity is never dispatched.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	log.Info("Processing order")
	activityFuture := workflow.ExecuteActivity(ctx, ProcessOrder)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(activityFuture, func(f workflow.Future) {})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {})
	selector.Select(ctx)

	if ctx.Err() != nil {
		log.Warn("Canceling order", "error", ctx.Err())
		// BUG: ctx is already canceled -- CancelOrder returns CanceledError
		// immediately without ever being executed.
		if err := workflow.ExecuteActivity(ctx, CancelOrder).Get(ctx, nil); err != nil {
			log.Error("Failed to cancel order", "error", err)
		} else {
			log.Info("Order canceled")
		}
	}
	return activityFuture.Get(ctx, nil)
}

// @@@SNIPEND

// @@@SNIPSTART not-using-disconnected-context-good

// MyWorkflowV2 uses a disconnected context for cleanup, so the
// activity runs even after the workflow is canceled.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	log.Info("Processing order")
	activityFuture := workflow.ExecuteActivity(ctx, ProcessOrder)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(activityFuture, func(f workflow.Future) {})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {})
	selector.Select(ctx)

	if ctx.Err() != nil {
		log.Warn("Canceling order", "error", ctx.Err())
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		if err := workflow.ExecuteActivity(newCtx, CancelOrder).Get(newCtx, nil); err != nil {
			log.Error("Failed to cancel order", "error", err)
		} else {
			log.Info("Order canceled")
		}
	}
	return activityFuture.Get(ctx, nil)
}

// @@@SNIPEND
