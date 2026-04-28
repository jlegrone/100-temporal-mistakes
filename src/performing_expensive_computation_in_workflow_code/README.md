# Performing Expensive Computation in Workflow Code

<!-- TODO: add a go benchmark test for replaying a v1 workflow history 100 times that does a computation in the workflow code, and a v2 workflow that does the same computation but via an activity. Put the benchmark outputs in the readme. -->
<!-- TODO: mention the importance of optimizing replay for when workers are redeployed -->

> [!TIP]
> [Workflow tasks](../terms/workflow-task.md) have a default 10-second timeout. Expensive computation blocks the task, causing timeouts and livelocks where the workflow perpetually retries but never makes progress.

<!-- TODO: minimize the emphasis on task timeout. It's just one (worst case) symptom. -->
Performing heavy computation directly in workflow code -- large data transformations, complex calculations, parsing large files -- can cause the workflow task to exceed its timeout. The server reschedules the task, the [worker](../terms/worker.md) [replays](../terms/replay.md) the full [history](../terms/event-history.md), hits the same expensive computation, times out again, and the cycle repeats. Replay makes this worse: the computation runs on every replay, compounding the cost.

<!--SNIPSTART expensive-computation-bad-->
[performing_expensive_computation_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/performing_expensive_computation_in_workflow_code/workflow.go)
```go
// BAD: expensive computation in workflow code
func MyWorkflowV1(ctx workflow.Context, data []Record) error {
	// TODO: make this look less contrived -- maybe use bcrypt or some other expensive operation as an example? Or even sha2 (this might be more realistic, eg. to compute a checksum for the identity of a resouce being created in a subsequent activity).
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

Activity results are recorded in workflow history so the computation doesn't repeat on replay. For moderate computation (tens of milliseconds), [local activities](../terms/local-activity.md) are a lighter-weight option. Truly trivial operations (comparisons, arithmetic, string formatting) are fine in workflow code as long as it is deterministic.

See also: [Exceeding the 10-Second Workflow Task Timeout](../exceeding-10s-task-timeout.md).
