# Not Waiting for Child Workflows to Start

> [!TIP]
> * When using a disconnected context for cleanup, you must wait for the child workflow to actually start before the parent returns.
> * `GetChildWorkflowExecution()` resolves when the child is scheduled on the server -- use it as your synchronization point.
> * If the parent completes before the child is scheduled, the child may never be created.

## What?

When a workflow is cancelled and you use a [disconnected context for cleanup](not-using-disconnected-context-for-cleanup.md), a common pattern is to start a child workflow to perform compensating actions. The mistake is returning from the parent workflow immediately after calling `ExecuteChildWorkflow()` without waiting for the child to actually start.

`ExecuteChildWorkflow()` returns a future, but the child workflow isn't scheduled on the server the moment you call it. The scheduling happens asynchronously. If the parent workflow completes (returns) before the server processes the child workflow creation, the child may never be started because the parent is already closed.

## Why?

When a parent workflow completes, Temporal stops processing further commands from that workflow execution. If the child workflow creation command hasn't been sent to the server yet -- or hasn't been processed -- it's effectively lost. This is a race condition: sometimes the child starts, sometimes it doesn't, making it particularly tricky to debug.

This is especially problematic in cancellation cleanup scenarios where reliability matters most. The whole point of the cleanup child workflow is to run compensating logic, and silently failing to start it defeats the purpose.

## How?

Use `GetChildWorkflowExecution()` on the child workflow future to wait until the child has been successfully scheduled:

```go
func MyWorkflow(ctx workflow.Context) error {
    err := workflow.ExecuteActivity(ctx, MyActivity).Get(ctx, nil)
    if err != nil && temporal.IsCanceledError(ctx.Err()) {
        // Create a disconnected context for cleanup
        disconnectedCtx, cancel := workflow.NewDisconnectedContext(ctx)
        defer cancel()

        // Start the cleanup child workflow
        childFuture := workflow.ExecuteChildWorkflow(disconnectedCtx, CleanupWorkflow, cleanupInput)

        // Wait for the child to be scheduled on the server
        // This is the critical step -- don't skip it!
        if err := childFuture.GetChildWorkflowExecution().Get(disconnectedCtx, nil); err != nil {
            return fmt.Errorf("failed to start cleanup workflow: %w", err)
        }

        // Now it's safe to return -- the child will continue running
        // independently even after the parent completes.
        return nil
    }
    return err
}
```

The key distinction:
- `childFuture.Get()` waits for the child workflow to **complete** (blocks until the child finishes).
- `childFuture.GetChildWorkflowExecution().Get()` waits for the child workflow to **start** (blocks only until the server confirms scheduling).

If you want fire-and-forget semantics (parent doesn't need the child's result), waiting for the child to start is the minimum you must do. If you need the child's result, wait for `childFuture.Get()` instead, which implicitly also waits for the child to start.
