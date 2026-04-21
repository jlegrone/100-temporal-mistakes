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

	err := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil)
	if err != nil {
		// This code is unreachable on timeout: the workflow is terminated,
		// not canceled, so none of this executes.
		log.Warn("compensating")
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
	if timeout := workflow.GetInfo(ctx).WorkflowRunTimeout; timeout > padding {
		return timeout - padding, nil
	}
	if timeout := workflow.GetInfo(ctx).WorkflowExecutionTimeout; timeout > padding {
		return timeout - padding, nil
	}
	return 0, temporal.NewNonRetryableApplicationError(
		"Workflow timeout is too small",
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

	softTimeout, err := getSoftTimeout(ctx, 24*time.Hour)
	if err != nil {
		return err
	}

	ctx, cancel := workflow.WithCancel(ctx)
	workflow.Go(ctx, func(ctx workflow.Context) {
		workflow.Sleep(ctx, softTimeout)
		cancel()
	})

	childFuture := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow)

	selector := workflow.NewSelector(ctx)

	selector.AddFuture(childFuture, func(f workflow.Future) {
		log.Info("child workflow completed")
	})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("workflow canceled")
	})
	var timedOut bool
	selector.AddFuture(workflow.NewTimer(ctx, softTimeout), func(f workflow.Future) {
		log.Warn("deadline exceeded")
		timedOut = true
	})
	selector.Select(ctx)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if timedOut {
		log.Warn("compensating")
		_ = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		}), CompensateActivity).Get(ctx, nil)
		return err
	}
	return childFuture.Get(ctx, nil)
}

// @@@SNIPEND
