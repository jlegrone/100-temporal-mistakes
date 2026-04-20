# Not Using a Disconnected Context for Cleanup

> [!TIP]
> Activities or [child workflows](terms/child-workflow.md) started with a canceled context are never dispatched. Use `workflow.NewDisconnectedContext()` for any cleanup that must run after [cancellation](terms/cancelation.md).

When a workflow is canceled, the root context and all descendants are canceled. If cleanup code (compensation, resource release, notifications) uses the original context, it silently fails -- the activity is never scheduled and `Get()` returns `CanceledError` immediately.

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    err := workflow.ExecuteActivity(ctx, ProcessOrder, input).Get(ctx, nil)
    if err != nil && ctx.Err() == workflow.ErrCanceled {
        cleanupCtx, cancel := workflow.NewDisconnectedContext(ctx)
        defer cancel()
        cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
            StartToCloseTimeout: 30 * time.Second,
        })
        _ = workflow.ExecuteActivity(cleanupCtx, CancelOrder, input).Get(cleanupCtx, nil)
    }
    return err
}
```

Always set a timeout on cleanup activities -- without one, a stuck cleanup runs until its [start-to-close timeout](terms/start-to-close-timeout.md) expires. Defer the `cancel()` to prevent resource leaks. Use `defer` with a disconnected context for cleanup that should always run regardless of how the workflow exits.

See also: [Deadlocking When a Workflow Is Canceled](deadlocking-when-workflow-canceled.md).
