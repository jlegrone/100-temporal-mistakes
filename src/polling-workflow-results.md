# Polling for Workflow Results

> [!TIP]
> Don't poll `DescribeWorkflowExecution` in a loop to check if a workflow completed. Use the SDK's blocking `GetWorkflow`/`result()` method, which uses long-polling under the hood.

Repeatedly calling `DescribeWorkflowExecution` or `GetWorkflowExecutionHistory` to detect completion wastes server resources, forces a latency-vs-load tradeoff, and introduces race conditions between detecting completion and fetching the result.

Every Temporal SDK provides a blocking method that waits efficiently for completion:

```go
// Go SDK
run := client.GetWorkflow(ctx, workflowID, runID)
var result MyResult
err := run.Get(ctx, &result)
// Blocks until completion, failure, or context cancellation
```

```typescript
// TypeScript SDK
const result = await client.workflow.getHandle(workflowId).result();
```

From within a workflow, use [child workflows](terms/child-workflow.md) -- never use activities to poll for workflow results, as this wastes activity slots and bloats [history](terms/event-history.md). For async notification, have the target workflow call an activity at the end that notifies your service, or send a [signal](terms/signals.md) to a known workflow on completion.
