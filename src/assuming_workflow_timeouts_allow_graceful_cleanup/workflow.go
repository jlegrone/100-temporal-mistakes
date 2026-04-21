package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"context"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// LongRunningWorkflow simulates a child workflow that takes a long time.
func LongRunningWorkflow(ctx workflow.Context) error {
	return workflow.Sleep(ctx, time.Hour)
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

	if err := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil); err != nil {
		// This code is unreachable on timeout: the workflow is terminated,
		// not canceled, so none of this executes.
		log.Warn("compensating", "error", err)
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		return err
	}

	return nil
}

// @@@SNIPEND

// @@@SNIPSTART assuming-workflow-timeouts-good

func getSoftTimeout(ctx workflow.Context, padding time.Duration) (time.Duration, error) {
	info := workflow.GetInfo(ctx)
	// Prefer the run timeout if set; only fall back to execution timeout
	// if no run timeout is configured.
	timeout := info.WorkflowRunTimeout
	if timeout == 0 {
		timeout = info.WorkflowExecutionTimeout
	}
	if timeout > padding {
		return timeout - padding, nil
	}
	return 0, temporal.NewNonRetryableApplicationError(
		"workflow timeout is too small",
		"wf_timeout_too_small",
		nil,
		map[string]string{
			"min_timeout": padding.String(),
		})
}

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the child workflow completes, the
// workflow can react gracefully instead of being terminated.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	// Reserve at least 1 minute for the workflow to perform compensating actions
	// before it is terminated due to workflow run timeout.
	softTimeout, err := getSoftTimeout(ctx, time.Minute)
	if err != nil {
		return err
	}

	var (
		childFuture = workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow)
		selector    = workflow.NewSelector(ctx)
		selectErr   error
	)

	// Wait for child workflow
	selector.AddFuture(childFuture, func(f workflow.Future) {
		log.Info("child workflow completed")
		selectErr = childFuture.Get(ctx, nil)
	})
	// Wait for workflow cancelation
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("workflow canceled")
		selectErr = ctx.Err()
	})
	// Wait for soft timeout
	selector.AddFuture(workflow.NewTimer(ctx, softTimeout), func(f workflow.Future) {
		log.Warn("deadline exceeded")
		selectErr = workflow.ErrDeadlineExceeded
	})

	// Select once; this sets up a race between the child workflow, context
	// cancellation, and our soft timeout.
	selector.Select(ctx)
	// Run compensating activity if any branch of the selector generated an error.
	if selectErr != nil {
		log.Warn("compensating", "error", selectErr)
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		return selectErr
	}

	return nil
}

// @@@SNIPEND
