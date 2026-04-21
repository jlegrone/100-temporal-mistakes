# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> When a [workflow execution timeout](../terms/workflow-execution-timeout.md) fires, Temporal [terminates](../terms/terminate.md) the workflow -- it does not [cancel](../terms/cancelation.md) it. No cleanup code runs.

A common assumption is that a timed-out workflow receives a cancelation signal and gets a chance to run compensation logic, release resources, or send notifications. This is wrong. Timeout-triggered termination is the equivalent of `kill -9`: no deferred functions execute, no cancelation handlers fire. If your workflow holds external state (a distributed lock, a lease), it will be left dangling.

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

```
<!--SNIPEND-->

If you need graceful behavior on timeout, implement the deadline yourself with a timer. If the timer fires before the workflow completes, the workflow can take action -- log, run compensation, or continue as new. Keep the workflow-level execution timeout as a safety net set to something longer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

<!--SNIPSTART assuming-workflow-timeouts-good-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

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

	if timedOut || ctx.Err() != nil {
		log.Warn("compensating")
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return workflow.ErrDeadlineExceeded
	}

	return childFuture.Get(ctx, nil)
}

```
<!--SNIPEND-->

<!--SNIPSTART assuming-workflow-timeouts-test-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go)
```go

func TestV2_CompensatesOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Track whether CompensateActivity was called.
	compensated := false
	env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Run(
		func(args mock.Arguments) { compensated = true },
	)

	// Set a run timeout so getSoftTimeout can derive the internal deadline.
	// The child sleeps for 1h; with a 24h10m run timeout and 24h padding,
	// the soft timeout is 10m -- shorter than the child, so the deadline
	// fires first.
	env.SetWorkflowRunTimeout(24*time.Hour + 10*time.Minute)

	env.ExecuteWorkflow(MyWorkflowV2)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.True(t, compensated, "CompensateActivity should have been called")
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_cancelled/) for more on handling cancelation in blocking workflows.
