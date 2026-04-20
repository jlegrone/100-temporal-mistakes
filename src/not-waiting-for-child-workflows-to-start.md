# Not Waiting for Child Workflows to Start

> [!TIP]
> `ExecuteChildWorkflow()` doesn't immediately schedule the [child workflow](terms/child-workflow.md). If the parent completes before the server processes the creation, the child may never start.

When using a [disconnected context](terms/disconnected-context.md) for cleanup after [cancellation](terms/cancellation.md), a common mistake is returning from the parent immediately after calling `ExecuteChildWorkflow()`. Scheduling happens asynchronously -- if the parent returns first, the child creation command is lost.

Use `GetChildWorkflowExecution()` to wait until the child is actually scheduled:

```go
// Start the cleanup child workflow
childFuture := workflow.ExecuteChildWorkflow(disconnectedCtx, CleanupWorkflow, input)

// Wait for the child to be scheduled -- this is the critical step
if err := childFuture.GetChildWorkflowExecution().Get(disconnectedCtx, nil); err != nil {
    return fmt.Errorf("failed to start cleanup workflow: %w", err)
}
// Now safe to return -- the child runs independently
```

The key distinction: `childFuture.Get()` waits for the child to **complete**; `childFuture.GetChildWorkflowExecution().Get()` waits only for the child to **start**. For fire-and-forget semantics, waiting for the start is the minimum.
