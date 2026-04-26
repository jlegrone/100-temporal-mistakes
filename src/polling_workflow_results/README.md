# Polling for Workflow Results

> [!TIP]
> Don't poll `DescribeWorkflowExecution` in a loop to check if a workflow completed. Use the SDK's blocking `GetWorkflow`/`result()` method, which uses more efficient long-polling under the hood.

Repeatedly calling `DescribeWorkflowExecution` or `GetWorkflowExecutionHistory` to detect completion wastes server resources and forces a latency-vs-load tradeoff.

Every Temporal SDK provides a blocking method that waits efficiently for completion:

<!--SNIPSTART polling-workflow-results-good-->
[polling_workflow_results/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/polling_workflow_results/workflow.go)
```go

// Good: use the SDK's blocking GetWorkflow method.
func GetWorkflowResult(ctx context.Context, c client.Client, workflowID, runID string) (MyResult, error) {
	run := c.GetWorkflow(ctx, workflowID, runID)
	var result MyResult
	err := run.Get(ctx, &result)
	// Blocks until completion, failure, or context cancellation
	return result, err
}

```
<!--SNIPEND-->

```typescript
// TypeScript SDK
const result = await client.workflow.getHandle(workflowId).result();
```

From within workflows, prefer using a child workflow and awaiting its return value.

<!-- TODO: Link to invoking workflows from activity mistake. -->
