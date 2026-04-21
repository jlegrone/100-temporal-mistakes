# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> When a [workflow execution timeout](../terms/workflow-execution-timeout.md) fires, Temporal [terminates](../terms/terminate.md) the workflow -- it does not [cancel](../terms/cancelation.md) it. No cleanup code runs.

A common assumption is that a timed-out workflow receives a cancelation signal and gets a chance to run compensation logic, release resources, or send notifications. This is wrong. For workflows, there is no difference in behavior between an execution timeout and a termination: no deferred functions execute, no cancelation handlers fire.

A workflow that relies on the execution timeout as its business deadline will be terminated without any opportunity to react:

<!--SNIPSTART assuming-workflow-timeouts-bad-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

// MyWorkflowV1 relies on the workflow execution timeout as its
// business deadline. It tries to compensate when the workflow is
// canceled, but when the timeout fires Temporal terminates the
// workflow -- the compensation never runs.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	if err := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil); err != nil {
		// This code is unreachable if LongRunningWorkflow takes longer to return a result than
		// the execution timeout of MyWorkflowV1. The workflow is effectively terminated, not
		// canceled, so no compensation logic executes.
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

```
<!--SNIPEND-->

If you need graceful behavior on timeout, implement the deadline yourself with a timer. If the timer fires before the workflow completes, the workflow can still take action. Keep the workflow-level execution timeout as a safety net set to something longer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

<!--SNIPSTART assuming-workflow-timeouts-good-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

// getSoftTimeout returns a timer that fires before the workflow's hard timeout,
// leaving at least padding duration for the workflow to perform cleanup (e.g.
// compensation activities) before it is terminated. It uses the run timeout if
// set, otherwise the execution timeout. Returns an error if neither timeout is
// large enough to accommodate the padding.
func getSoftTimeout(ctx workflow.Context, padding time.Duration) (workflow.Future, error) {
	info := workflow.GetInfo(ctx)
	// Prefer the run timeout if set; only fall back to execution timeout
	// if no run timeout is configured.
	timeout := info.WorkflowRunTimeout
	if timeout == 0 {
		timeout = info.WorkflowExecutionTimeout
	}
	if timeout > padding {
		return workflow.NewTimerWithOptions(ctx, timeout-padding, workflow.TimerOptions{
			Summary: fmt.Sprintf("soft_timeout_%s", padding),
		}), nil
	}
	return nil, temporal.NewNonRetryableApplicationError(
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
	selector.AddFuture(softTimeout, func(f workflow.Future) {
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

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_canceled/) for more on handling cancelation in blocking workflows.
