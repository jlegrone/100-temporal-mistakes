# Not Waiting for Child Workflows to Start

<!-- TODO: Add a unit test demonstrating the issue (with a V1 workflow version) and a fix (with V2). -->
<!-- TODO: Add a ExecuteDisconnectedChildWorkflow helper to the workflowhelpers package that demonstrates how to avoid this issue in Go. Maybe it could also be an interceptor? Add a "v3" version that is exactly the same as the v1 version of the workflow but with this helper/interceptor enabled and demonstrate that the behavior is fixed in a unit test. -->
<!-- TODO: Find out if this bug / surprising behavior also exists in TypeScript & Python SDKs. -->

> [!TIP]
> `ExecuteChildWorkflow()` doesn't immediately schedule the [child workflow](terms/child-workflow.md). If the parent completes before the server processes the creation, the child may never start.

When using a [disconnected context](terms/disconnected-context.md) for cleanup after [cancellation](terms/cancelation.md), a common mistake is returning from the parent immediately after calling `ExecuteChildWorkflow()`. Scheduling happens asynchronously -- if the parent returns first, the child creation command is lost.

Use `GetChildWorkflowExecution()` to wait until the child is actually scheduled:

<!--SNIPSTART not-waiting-for-child-workflows-to-start-workflow-->
[not_waiting_for_child_workflows_to_start/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_waiting_for_child_workflows_to_start/workflow.go)
```go

func MyWorkflow(disconnectedCtx workflow.Context, input Input) error {
	// Start the cleanup child workflow
	childFuture := workflow.ExecuteChildWorkflow(disconnectedCtx, CleanupWorkflow, input)

	// Wait for the child to be scheduled -- this is the critical step
	if err := childFuture.GetChildWorkflowExecution().Get(disconnectedCtx, nil); err != nil {
		return fmt.Errorf("failed to start cleanup workflow: %w", err)
	}
	// Now safe to return -- the child runs independently
	return nil
}

```
<!--SNIPEND-->
