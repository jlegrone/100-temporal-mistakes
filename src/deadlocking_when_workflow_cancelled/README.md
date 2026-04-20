# Deadlocking When a Workflow Is Canceled

> [!TIP]
> When a workflow is [canceled](../terms/cancelation.md), all derived contexts are canceled too. Cleanup code that uses the original context will never execute -- use a [disconnected context](../not_using_disconnected_context_for_cleanup/) instead.

When Temporal delivers a cancellation request, the SDK cancels the workflow's context and every context derived from it. A common mistake is running cleanup in a `defer` using the original context:

<!--SNIPSTART deadlocking-cancelled-bad-->
[deadlocking_when_workflow_cancelled/examples.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/examples.go)
```go
// BUG: ctx is already canceled in the defer
func badCleanup(ctx workflow.Context, input any) {
	defer func() {
		err := workflow.ExecuteActivity(ctx, CleanupActivity, input).Get(ctx, nil)
		// Always returns CanceledError -- cleanup never runs
		_ = err
	}()
}

```
<!--SNIPEND-->

The fix: create a disconnected context that remains valid after cancellation:

<!--SNIPSTART deadlocking-cancelled-good-->
[deadlocking_when_workflow_cancelled/examples.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/deadlocking_when_workflow_cancelled/examples.go)
```go
func goodCleanup(ctx workflow.Context, input any) {
	defer func() {
		disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		_ = workflow.ExecuteActivity(disconnectedCtx, CleanupActivity, input).Get(disconnectedCtx, nil)
	}()
}

```
<!--SNIPEND-->

Deadlocked workflows don't complete or fail -- they sit consuming resources and typically require manual [termination](../terms/terminate.md), which means the cleanup you intended never runs at all.

See also: [Not Using a Disconnected Context for Cleanup](../not_using_disconnected_context_for_cleanup/).
