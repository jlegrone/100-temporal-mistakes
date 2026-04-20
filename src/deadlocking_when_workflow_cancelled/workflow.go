package deadlocking_when_workflow_cancelled

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// CleanupActivity performs cleanup after workflow cancellation.
func CleanupActivity(_ context.Context, _ any) error {
	return nil
}

// @@@SNIPSTART deadlocking-cancelled-bad
// BUG: ctx is already canceled in the defer
func MyWorkflowV1(ctx workflow.Context, input any) {
	defer func() {
		err := workflow.ExecuteActivity(ctx, CleanupActivity, input).Get(ctx, nil)
		// Always returns CanceledError -- cleanup never runs
		_ = err
	}()
}

// @@@SNIPEND

// @@@SNIPSTART deadlocking-cancelled-good
func MyWorkflowV2(ctx workflow.Context, input any) {
	defer func() {
		disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		_ = workflow.ExecuteActivity(disconnectedCtx, CleanupActivity, input).Get(disconnectedCtx, nil)
	}()
}

// @@@SNIPEND
