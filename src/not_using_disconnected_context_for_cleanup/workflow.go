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
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	err := workflow.ExecuteActivity(ctx, ProcessOrder).Get(ctx, nil)
	if err != nil && ctx.Err() == workflow.ErrCanceled {
		// BUG: ctx is already canceled -- CancelOrder is never dispatched
		_ = workflow.ExecuteActivity(ctx, CancelOrder).Get(ctx, nil)
	}
	return err
}

// @@@SNIPEND

// @@@SNIPSTART not-using-disconnected-context-good

// MyWorkflowV2 uses a disconnected context for cleanup, so the
// activity runs even after the workflow is canceled.
func MyWorkflowV2(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	err := workflow.ExecuteActivity(ctx, ProcessOrder).Get(ctx, nil)
	if err != nil && ctx.Err() == workflow.ErrCanceled {
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		_ = workflow.ExecuteActivity(newCtx, CancelOrder).Get(newCtx, nil)
	}
	return err
}

// @@@SNIPEND
