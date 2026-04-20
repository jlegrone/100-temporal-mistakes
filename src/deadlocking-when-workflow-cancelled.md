# Deadlocking When a Workflow Is Canceled

> [!TIP]
> When a workflow is [canceled](terms/cancelation.md), all derived contexts are canceled too. Cleanup code that uses the original context will never execute -- use a [disconnected context](not-using-disconnected-context-for-cleanup.md) instead.

When Temporal delivers a cancellation request, the SDK cancels the workflow's context and every context derived from it. A common mistake is running cleanup in a `defer` using the original context:

```go
// BUG: ctx is already canceled in the defer
defer func() {
    err := workflow.ExecuteActivity(ctx, CleanupActivity, input).Get(ctx, nil)
    // Always returns CanceledError -- cleanup never runs
}()
```

The fix: create a disconnected context that remains valid after cancellation:

```go
defer func() {
    disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
    defer cancel()
    _ = workflow.ExecuteActivity(disconnectedCtx, CleanupActivity, input).Get(disconnectedCtx, nil)
}()
```

Deadlocked workflows don't complete or fail -- they sit consuming resources and typically require manual [termination](terms/terminate.md), which means the cleanup you intended never runs at all.

See also: [Not Using a Disconnected Context for Cleanup](not-using-disconnected-context-for-cleanup.md).
