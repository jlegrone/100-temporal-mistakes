package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// LongRunningWorkflow simulates a child workflow that takes a long time.
func LongRunningWorkflow(ctx workflow.Context) error {
	return workflow.Sleep(ctx, 24*time.Hour)
}

// CompensateActivity runs compensation logic.
func CompensateActivity(_ context.Context) error {
	return nil
}

// @@@SNIPSTART assuming-workflow-timeouts-bad

// MyWorkflowV1 relies on the workflow execution timeout as its
// business deadline. It tries to compensate when the workflow is
// canceled, but when the timeout fires Temporal terminates the
// workflow -- the compensation never runs.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	err := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil)
	if err != nil {
		// This code is unreachable on timeout: the workflow is terminated,
		// not canceled, so none of this executes.
		log.Warn("compensating")
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		return err
	}
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART assuming-workflow-timeouts-good

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the child workflow completes, the
// workflow can react gracefully instead of being terminated.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	childFuture := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow)
	deadline := workflow.NewTimer(ctx, 30*time.Minute)

	selector := workflow.NewSelector(ctx)

	var timedOut bool
	selector.AddFuture(childFuture, func(f workflow.Future) {
		log.Info("child workflow completed")
	})
	selector.AddFuture(deadline, func(f workflow.Future) {
		log.Warn("deadline exceeded")
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
		return workflow.NewContinueAsNewError(ctx, MyWorkflowV2)
	}
	return childFuture.Get(ctx, nil)
}

// @@@SNIPEND
