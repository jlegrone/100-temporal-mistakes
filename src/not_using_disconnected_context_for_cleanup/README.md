# Not Using a Disconnected Context for Cleanup

> [!TIP]
> Activities or [child workflows](../terms/child-workflow.md) started with a canceled context are never dispatched. Use `workflow.NewDisconnectedContext()` for any cleanup that must run after [cancelation](../terms/cancelation.md).

When a workflow is canceled, the root context and all descendants are canceled. If cleanup code (compensation, resource release, notifications) uses the original context, it silently fails -- the activity is never scheduled and `Get()` returns `CanceledError` immediately.

<!--SNIPSTART not-using-disconnected-context-bad-->
[not_using_disconnected_context_for_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_disconnected_context_for_cleanup/workflow.go)
```go

// MyWorkflowV1 tries to run a cleanup activity after cancelation,
// but uses the original (already-canceled) context. The cleanup
// activity is never dispatched.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	log.Info("Processing order")
	activityFuture := workflow.ExecuteActivity(ctx, ProcessOrder)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(activityFuture, func(f workflow.Future) {})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {})
	selector.Select(ctx)

	if ctx.Err() != nil {
		log.Warn("Canceling order", "error", ctx.Err())
		// BUG: ctx is already canceled -- CancelOrder returns CanceledError
		// immediately without ever being executed.
		if err := workflow.ExecuteActivity(ctx, CancelOrder).Get(ctx, nil); err != nil {
			log.Error("Failed to cancel order", "error", err)
		} else {
			log.Info("Order canceled")
		}
	}
	return activityFuture.Get(ctx, nil)
}

```
<!--SNIPEND-->

Use a disconnected context for cleanup, so the activity runs even after the workflow is canceled:

<!--SNIPSTART not-using-disconnected-context-good-->
[not_using_disconnected_context_for_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_disconnected_context_for_cleanup/workflow.go)
```go

// MyWorkflowV2 uses a disconnected context for cleanup, so the
// activity runs even after the workflow is canceled.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	})

	log.Info("Processing order")
	activityFuture := workflow.ExecuteActivity(ctx, ProcessOrder)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(activityFuture, func(f workflow.Future) {})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {})
	selector.Select(ctx)

	if ctx.Err() != nil {
		log.Warn("Canceling order", "error", ctx.Err())
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		if err := workflow.ExecuteActivity(newCtx, CancelOrder).Get(newCtx, nil); err != nil {
			log.Error("Failed to cancel order", "error", err)
		} else {
			log.Info("Order canceled")
		}
	}
	return activityFuture.Get(ctx, nil)
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_canceled/).
