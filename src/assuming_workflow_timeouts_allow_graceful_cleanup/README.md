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
	log := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, "done")
	ch.Receive(ctx, nil)

	log.Info("done")
	return nil
}

```
<!--SNIPEND-->

If you need graceful behavior on timeout, implement the deadline yourself with a timer. If the timer fires before the workflow completes, the workflow can take action -- log, run compensation, or continue as new. Keep the workflow-level execution timeout as a safety net set to something longer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

<!--SNIPSTART assuming-workflow-timeouts-good-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the signal arrives, the workflow
// completes gracefully with an error instead of being terminated.
func MyWorkflowV2(ctx workflow.Context, deadline time.Duration) error {
	log := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, "done")
	timer := workflow.NewTimer(ctx, deadline)

	selector := workflow.NewSelector(ctx)

	var timedOut bool
	selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
		log.Info("done")
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		log.Warn("deadline exceeded", "deadline", deadline)
		timedOut = true
	})
	selector.Select(ctx)

	if timedOut {
		return workflow.NewContinueAsNewError(ctx, MyWorkflowV2, deadline)
	}
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART assuming-workflow-timeouts-test-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go)
```go

func TestV2_CompletesWhenSignaled(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestV2_ContinuesAsNewOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	// Don't send the signal -- let the deadline fire.
	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	// The workflow calls ContinueAsNew when the deadline fires.
	var continueAsNewErr *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &continueAsNewErr)
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_cancelled/) for more on handling cancelation in blocking workflows.
