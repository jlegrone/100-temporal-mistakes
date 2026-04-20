package not_using_disconnected_context_for_cleanup

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// Input represents the workflow input.
type Input struct{}

// @@@SNIPSTART not-using-disconnected-context-for-cleanup-workflow

func MyWorkflow(ctx workflow.Context, input Input) error {
	err := workflow.ExecuteActivity(ctx, ProcessOrder, input).Get(ctx, nil)
	if err != nil && ctx.Err() == workflow.ErrCanceled {
		cleanupCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		_ = workflow.ExecuteActivity(cleanupCtx, CancelOrder, input).Get(cleanupCtx, nil)
	}
	return err
}

// @@@SNIPEND

// ProcessOrder is a stub activity.
func ProcessOrder(_ Input) error { return nil }

// CancelOrder is a stub activity.
func CancelOrder(_ Input) error { return nil }
