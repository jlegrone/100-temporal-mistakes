# Deadlocking When a Workflow Is Cancelled

> [!TIP]
> * When a workflow is cancelled, all pending activity and child workflow contexts are immediately cancelled too.
> * If cleanup code tries to execute activities using the already-cancelled context, it blocks forever -- the workflow deadlocks.
> * Use a [disconnected context](<not-using-disconnected-context-for-cleanup.md>) for any work that must run after cancellation.

## What?

When Temporal delivers a cancellation request to a workflow, the SDK cancels the workflow's context. This in turn cancels every derived context, including those associated with pending activities and child workflows. Any `Future.Get()` call on a pending activity will immediately return a `CanceledError`.

The deadlock happens when workflow code is structured to perform cleanup after catching the cancellation but uses the original (now cancelled) context to do it. A common pattern that triggers this:

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    // Schedule cleanup to run when the workflow exits
    defer func() {
        // BUG: ctx is already cancelled here
        err := workflow.ExecuteActivity(ctx, CleanupActivity, input).Get(ctx, nil)
        if err != nil {
            // This error is always CanceledError -- cleanup never actually runs
        }
    }()

    // Main workflow logic
    err := workflow.ExecuteActivity(ctx, MainActivity, input).Get(ctx, nil)
    if err != nil {
        return err
    }
    return nil
}
```

When the workflow is cancelled, `MainActivity` returns a `CanceledError`, the function returns, the `defer` fires, and `CleanupActivity` is scheduled with the cancelled context. The activity is never dispatched to a worker. The `Get()` call returns immediately with a `CanceledError`. In many cases the workflow ends up in a state where no forward progress is possible, causing it to deadlock from Temporal's perspective.

## Why?

This deadlock is particularly frustrating because the code looks correct at first glance. The `defer` pattern is idiomatic Go and cleanup-after-error is a natural instinct. The problem is that Temporal's cancellation model is cooperative and context-based: once a context is cancelled, nothing scheduled on it will execute.

Deadlocked workflows are stuck. They don't complete, they don't fail, they just sit there consuming resources and cluttering your workflow list. They typically require manual [termination](terms/terminate.md) to clear, which means your cleanup logic never runs at all -- the exact opposite of what you intended.

## How?

The fix is to create a disconnected context for cleanup work. A disconnected context is not tied to the parent's cancellation state and remains valid even after the workflow is cancelled.

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    defer func() {
        // Create a context that won't be cancelled when the workflow is
        disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
        defer cancel()

        // Now cleanup will actually execute
        err := workflow.ExecuteActivity(disconnectedCtx, CleanupActivity, input).Get(disconnectedCtx, nil)
        if err != nil {
            workflow.GetLogger(ctx).Error("cleanup failed", "error", err)
        }
    }()

    err := workflow.ExecuteActivity(ctx, MainActivity, input).Get(ctx, nil)
    if err != nil {
        return err
    }
    return nil
}
```

For a deeper look at disconnected contexts and when to use them, see [Not Using a Disconnected Context for Cleanup](<not-using-disconnected-context-for-cleanup.md>).

Key points to remember:

- **Never use the original context for post-cancellation work.** If the workflow is cancelled, any context derived from the original is also cancelled.
- **Always defer the `cancel()` of your disconnected context.** This prevents resource leaks in the normal (non-cancelled) code path.
- **Keep cleanup bounded.** Set a timeout on your disconnected context so that cleanup doesn't run forever if something goes wrong.
