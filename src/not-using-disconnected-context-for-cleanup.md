# Not Using a Disconnected Context for Cleanup

> [!TIP]
> * When a workflow is [cancelled](terms/cancellation.md), the main workflow context and all its descendants are cancelled too.
> * Activities or [child workflows](terms/child-workflow.md) started with a cancelled context will fail immediately -- they are never dispatched.
> * Create a [disconnected context](terms/disconnected-context.md) with `workflow.NewDisconnectedContext()` for any cleanup that must run after cancellation.

## What?

Temporal's cancellation mechanism is context-based. When a workflow receives a cancellation request, the SDK cancels the root workflow context. Every context derived from it -- including those used by pending activities, child workflows, timers, and selectors -- is cancelled as well.

If your workflow needs to perform cleanup operations after cancellation (sending a notification, releasing a lock, running compensation logic), those operations need a context that is still valid. Using the original context or any of its children will not work: the activity or child workflow will never be scheduled, and the `Get()` call will return a `CanceledError` immediately.

## Why?

This is one of the most common cancellation-related mistakes in Temporal workflows. The code to start a cleanup activity looks identical to the code that starts any other activity. There's no compile-time signal that the context is cancelled. The failure mode is silent: the cleanup activity simply doesn't run, and the workflow either completes without cleanup or [deadlocks](<deadlocking-when-workflow-cancelled.md>) depending on how the error is handled.

The consequences vary depending on what the cleanup was supposed to do:

- **Resource leaks**: locks never released, temporary resources never deleted.
- **Inconsistent state**: partial work left behind without compensation.
- **Missing notifications**: downstream systems not informed that work was abandoned.
- **Deadlocked workflows**: if the workflow waits on the cleanup result without checking for cancellation, it hangs forever.

## How?

Use `workflow.NewDisconnectedContext()` to create a context that is independent of the workflow's cancellation state:

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    var result Result

    err := workflow.ExecuteActivity(ctx, ProcessOrder, input).Get(ctx, &result)
    if err != nil {
        // Check if the workflow was cancelled
        if ctx.Err() == workflow.ErrCanceled {
            // Create a disconnected context for cleanup
            cleanupCtx, cancel := workflow.NewDisconnectedContext(ctx)
            defer cancel()

            // Give cleanup a bounded amount of time
            cleanupCtx = workflow.WithActivityOptions(cleanupCtx, workflow.ActivityOptions{
                StartToCloseTimeout: 30 * time.Second,
            })

            // This activity will actually execute, even though the workflow is cancelled
            _ = workflow.ExecuteActivity(cleanupCtx, CancelOrder, input).Get(cleanupCtx, nil)
        }
        return err
    }

    return nil
}
```

### Guidelines

1. **Always set a timeout on cleanup activities.** The disconnected context is not cancelled by the workflow, so without a timeout, a stuck cleanup activity could run until its [start-to-close timeout](terms/start-to-close-timeout.md) expires. Keep cleanup bounded and predictable.

2. **Defer the cancel function.** Even though the context is disconnected from the workflow's cancellation, calling `cancel()` when you're done prevents resource leaks in normal (non-cancelled) execution paths.

3. **Handle cleanup errors explicitly.** Since the workflow is already in a cancellation path, you need to decide what happens if cleanup itself fails. At minimum, log the error. Depending on your use case, you might retry or accept the failure.

4. **Use `defer` for cleanup that should always run.** If cleanup must happen regardless of how the workflow exits (success, failure, or cancellation), use `defer` with a disconnected context:

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    defer func() {
        disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
        defer cancel()

        _ = workflow.ExecuteActivity(disconnectedCtx, ReleaseResources, input).Get(disconnectedCtx, nil)
    }()

    // ... main workflow logic
}
```

For more on how cancelled contexts cause deadlocks, see [Deadlocking When a Workflow Is Cancelled](<deadlocking-when-workflow-cancelled.md>).
