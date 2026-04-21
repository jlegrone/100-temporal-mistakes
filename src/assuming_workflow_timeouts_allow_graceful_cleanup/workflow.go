package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// LongRunningWorkflow represents a child workflow that completes
// when it receives a "done" signal.
func LongRunningWorkflow(ctx workflow.Context) error {
	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)
	return nil
}

// @@@SNIPSTART assuming-workflow-timeouts-bad

// MyWorkflowV1 relies on the workflow execution timeout for its
// business deadline. When the timeout fires, the workflow is
// terminated -- no cleanup runs.
func MyWorkflowV1(ctx workflow.Context) error {
	return workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil)
}

// @@@SNIPEND

// @@@SNIPSTART assuming-workflow-timeouts-good

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the child workflow completes, the
// workflow can react gracefully instead of being terminated.
func MyWorkflowV2(ctx workflow.Context, deadline time.Duration) error {
	log := workflow.GetLogger(ctx)

	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: "long-running",
	})
	childFuture := workflow.ExecuteChildWorkflow(childCtx, LongRunningWorkflow)
	timer := workflow.NewTimer(ctx, deadline)

	selector := workflow.NewSelector(ctx)

	var timedOut bool
	selector.AddFuture(childFuture, func(f workflow.Future) {
		log.Info("child workflow completed")
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		log.Warn("deadline exceeded", "deadline", deadline)
		timedOut = true
	})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("workflow canceled")
	})
	selector.Select(ctx)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if timedOut {
		return workflow.NewContinueAsNewError(ctx, MyWorkflowV2, deadline)
	}
	return childFuture.Get(ctx, nil)
}

// @@@SNIPEND
