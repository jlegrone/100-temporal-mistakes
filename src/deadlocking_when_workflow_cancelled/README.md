# Deadlocking When a Workflow Is Canceled

> [!TIP]
> When a workflow is [canceled](../terms/cancelation.md), blocking calls like `Receive` that don't also listen for `ctx.Done()` will block forever, preventing the workflow from making progress.

When Temporal delivers a cancelation request, the SDK cancels the workflow's context. But operations that block without checking for cancelation -- like `channel.Receive(ctx, ...)` -- will never unblock, because the event they're waiting for will never arrive. The workflow is stuck: it can't complete, can't run cleanup, and will sit there until it hits a workflow timeout or is manually terminated.

<!--SNIPSTART deadlocking-cancelled-bad-->
[deadlocking_when_workflow_cancelled/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/workflow.go)
```go

// MyWorkflowV1 blocks forever if canceled. The Receive call blocks
// until a signal arrives, but once the workflow is canceled, no signal
// will ever be delivered.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	log.Debug("waiting for done signal")

	ch := workflow.GetSignalChannel(ctx, "done")

	ch.Receive(ctx, nil)
	log.Debug("received done signal")

	return nil
}

```
<!--SNIPEND-->

The fix: use a `Selector` to listen for both the expected event and `ctx.Done()`, so the workflow unblocks on cancelation:

<!--SNIPSTART deadlocking-cancelled-good-->
[deadlocking_when_workflow_cancelled/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/workflow.go)
```go

// MyWorkflowV2 uses a selector to unblock on either the signal
// or cancellation.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	log.Debug("waiting for done signal")

	ch := workflow.GetSignalChannel(ctx, "done")
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
		log.Debug("received done signal")
	})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("received done", "error", ctx.Err())
	})
	selector.Select(ctx)
	return ctx.Err()
}

```
<!--SNIPEND-->

<!--SNIPSTART deadlocking-cancelled-test-->
[deadlocking_when_workflow_cancelled/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/workflow_test.go)
```go

func TestV2_HandlesGracefulCancelation(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2)
	err := env.GetWorkflowError()

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err), fmt.Sprintf("Expected canceled error, got: %v", err))
}

```
<!--SNIPEND-->

See also: [Not Using a Disconnected Context for Cleanup](../not_using_disconnected_context_for_cleanup/) for the related problem of running cleanup activities after cancelation.
