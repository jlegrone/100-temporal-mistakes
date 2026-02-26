# Polling for Workflow Results

> [!TIP]
> * Don't repeatedly call `DescribeWorkflowExecution` or `GetWorkflowExecutionHistory` in a loop to check if a workflow has completed.
> * Use the SDK's blocking `GetWorkflowResult` (or equivalent) which efficiently waits for completion without polling.
> * From within a workflow, use child workflows or `ExecuteActivity` futures which natively provide completion notification.

## What?

A common anti-pattern is writing a polling loop that repeatedly checks whether a workflow has finished:

```go
// Don't do this
for {
    resp, err := client.DescribeWorkflowExecution(ctx, workflowID, runID)
    if err != nil {
        return err
    }
    if resp.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
        break
    }
    time.Sleep(5 * time.Second)
}
// Now fetch the result...
```

Or the equivalent with `GetWorkflowExecutionHistory`, scanning for a `WorkflowExecutionCompleted` event.

This works, but it is wasteful, fragile, and ignores a much better mechanism that every Temporal SDK provides out of the box.

## Why?

Polling is problematic for several reasons:

1. **Wasted resources**: Every poll is an RPC to the Temporal server. A polling loop running every few seconds multiplied by many concurrent callers puts unnecessary load on the server and increases API rate limit consumption.
2. **Latency tradeoff**: You're forced to choose between frequent polls (more load, lower latency) and infrequent polls (less load, higher latency). Either way, you detect completion later than necessary.
3. **Error handling complexity**: You need to handle transient RPC errors in your polling loop, decide when to back off, and deal with timeouts. This is boilerplate that obscures your actual business logic.
4. **Race conditions**: Between detecting completion and fetching the result, the workflow execution could be archived or the run ID could change (if the workflow continued as new).

## Solution

### From external code (API handlers, scripts, services)

Use the SDK client's blocking result retrieval. Every Temporal SDK provides a method that efficiently waits for a workflow to complete using a long-poll mechanism under the hood:

```go
// Go SDK
run := client.GetWorkflow(ctx, workflowID, runID)
var result MyResult
err := run.Get(ctx, &result)
// This blocks until the workflow completes, fails, or the context is cancelled
```

```python
# Python SDK
result = await client.get_workflow_handle(workflow_id).result()
```

```typescript
// TypeScript SDK
const result = await client.workflow.getHandle(workflowId).result();
```

These methods use the Temporal server's long-poll API internally, which is far more efficient than repeated `Describe` calls. The server holds the connection open and responds immediately when the workflow completes.

### From within a workflow

If you need to wait for another workflow's result from within a workflow, use a child workflow:

```go
func ParentWorkflow(ctx workflow.Context) error {
    var result MyResult
    // The child workflow future resolves when the child completes
    err := workflow.ExecuteChildWorkflow(ctx, ChildWorkflow, input).Get(ctx, &result)
    // ...
}
```

Never use activities to poll for workflow results from within a workflow. This wastes activity slots, grows history with polling events, and can lead to workflow history overflow for long-running target workflows.

### Async notification pattern

If you need to be notified of completion without blocking a thread, consider:

- **Callbacks via activities**: Have the target workflow call an activity at the end that notifies your service (via webhook, message queue, etc.).
- **Signal on completion**: Have the target workflow send a [signal](terms/signals.md) to a known workflow when it finishes.
