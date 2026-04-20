# Performing Expensive Computation in Workflow Code

> [!TIP]
> [Workflow tasks](../terms/workflow-task.md) have a default 10-second timeout. Expensive computation blocks the task, causing timeouts and livelocks where the workflow perpetually retries but never makes progress.

Performing heavy computation directly in workflow code -- large data transformations, complex calculations, parsing large files -- can cause the workflow task to exceed its timeout. The server reschedules the task, the [worker](../terms/worker.md) [replays](../terms/replay.md) the full [history](../terms/event-history.md), hits the same expensive computation, times out again, and the cycle repeats. Replay makes this worse: the computation runs on every replay, compounding the cost.

<!--SNIPSTART expensive-computation-bad-->
[performing_expensive_computation_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/performing_expensive_computation_in_workflow_code/workflow.go)
```go
// BAD: expensive computation in workflow code
func MyWorkflowV1(ctx workflow.Context, data []Record) error {
	result := expensiveTransformation(data) // Takes 30 seconds
	return workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
}

```
<!--SNIPEND-->

<!--SNIPSTART expensive-computation-good-->
[performing_expensive_computation_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/performing_expensive_computation_in_workflow_code/workflow.go)
```go
// TransformActivity moves the expensive computation into an activity.
func TransformActivity(_ context.Context, data []Record) (Result, error) {
	return expensiveTransformation(data), nil
}

// GOOD: move it to an activity
func MyWorkflowV2(ctx workflow.Context, data []Record) error {
	var result Result
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	})
	if err := workflow.ExecuteActivity(ctx, TransformActivity, data).Get(ctx, &result); err != nil {
		return err
	}
	return workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
}

```
<!--SNIPEND-->

Activities have independently configurable timeouts, can [heartbeat](../terms/heartbeat.md) progress, and their results are recorded in history so the computation doesn't repeat on replay. For moderate computation (a few hundred milliseconds), [local activities](../terms/local-activity.md) are a lighter-weight option. Truly trivial work (comparisons, arithmetic, string formatting) is fine in workflow code.

See also: [Exceeding the 10-Second Workflow Task Timeout](../exceeding-10s-task-timeout.md).
