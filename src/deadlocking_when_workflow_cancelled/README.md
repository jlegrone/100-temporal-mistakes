# Deadlocking When a Workflow Is Canceled

> [!TIP]
> When a workflow is [canceled](../terms/cancelation.md), all derived contexts are canceled too. Cleanup code that uses the original context will never execute -- use a [disconnected context](../not_using_disconnected_context_for_cleanup/) instead.

When Temporal delivers a cancellation request, the SDK cancels the workflow's context and every context derived from it. A common mistake is running cleanup in a `defer` using the original context:

<!--SNIPSTART deadlocking-cancelled-bad-->
[deadlocking_when_workflow_cancelled/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/workflow.go)
```go

// MyWorkflowV1 blocks forever if canceled. The Receive call blocks
// until a signal arrives, but once the workflow is canceled, no signal
// will ever be delivered.
func MyWorkflowV1(ctx workflow.Context) error {
	var done bool
	workflow.GetSignalChannel(ctx, "done").Receive(ctx, &done)
	return nil
}

```
<!--SNIPEND-->

The fix: create a disconnected context that remains valid after cancellation:

<!--SNIPSTART deadlocking-cancelled-good-->
[deadlocking_when_workflow_cancelled/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/workflow.go)
```go

// MyWorkflowV2 uses a selector to unblock on either the signal
// or cancellation.
func MyWorkflowV2(ctx workflow.Context) error {
	ch := workflow.GetSignalChannel(ctx, "done")
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {})
	selector.Select(ctx)
	return ctx.Err()
}

```
<!--SNIPEND-->

Deadlocked workflows don't complete or fail -- they sit consuming resources and typically require manual [termination](../terms/terminate.md), which means the cleanup you intended never runs at all.

See also: [Not Using a Disconnected Context for Cleanup](../not_using_disconnected_context_for_cleanup/).
