# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> When a [workflow execution timeout](../terms/workflow-execution-timeout.md) fires, Temporal [terminates](../terms/terminate.md) the workflow -- it does not [cancel](../terms/cancelation.md) it. No cleanup code runs.

A common assumption is that a timed-out workflow receives a cancelation signal and gets a chance to run compensation logic, release resources, or send notifications. This is wrong. Timeout-triggered termination is the equivalent of `kill -9`: no deferred functions execute, no cancelation handlers fire. If your workflow holds external state (a distributed lock, a lease), it will be left dangling.

A workflow that relies on the execution timeout as its business deadline will be terminated without any opportunity to react:

<!--SNIPSTART assuming-workflow-timeouts-bad-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

// MyWorkflowV1 relies on the workflow execution timeout for its
// business deadline. When the timeout fires, the workflow is
// terminated -- no cleanup runs.
func MyWorkflowV1(ctx workflow.Context) error {
	return workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil)
}

```
<!--SNIPEND-->

If you need graceful behavior on timeout, implement the deadline yourself with a timer. If the timer fires before the workflow completes, the workflow can take action -- log, run compensation, or continue as new. Keep the workflow-level execution timeout as a safety net set to something longer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

<!--SNIPSTART assuming-workflow-timeouts-good-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

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

```
<!--SNIPEND-->

<!--SNIPSTART assuming-workflow-timeouts-test-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go)
```go

func TestV2_CompletesWhenChildFinishes(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Signal the child workflow to complete before the deadline.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflowByID("long-running", "done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestV2_ContinuesAsNewOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Don't signal the child -- let the deadline fire first.
	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	var continueAsNewErr *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &continueAsNewErr)
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_cancelled/) for more on handling cancelation in blocking workflows.
