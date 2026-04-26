# Using Local Activities

<!-- TODO: Consider merging this with the fallible local activities entry, or reframing as "long duration local activities" -->

> [!TIP]
> [Local activities](../terms/local-activity.md) skip the server round-trip but run within the [workflow task](../terms/workflow-task.md) timeout (default 10s). If they take too long or fail, the entire workflow task is retried from scratch.

Local activities execute directly within the current workflow task on the same [worker](../terms/worker.md), eliminating the round-trip to the server and reducing latency. This sounds like a pure win, but they come with important constraints:

- **Bound by the workflow task timeout**: If a local activity exceeds the remaining time in the workflow task (default 10s), the task times out, gets rescheduled, and the local activity runs again from scratch -- potentially creating an infinite retry loop.
- **No independent visibility**: Local activities don't generate their own history events until the workflow task completes. Unlike regular activities, you can't see that they are running in the Temporal UI mid-execution.
<!-- TODO: Fact check this. Do local activities consume activity task slots? What about workflow task slots? -->
<!-- TODO: Add an interceptor that checks for local activities with complex retry policies or timeout set longer than 10s? Double check what happens if a local activity does run for longer than 10s in an integration test using the dev server. Do we hit the workflow cancel bug from the Go SDK? -->
- **No load balancing**: They run on the current worker only, unlike normal activities dispatched through the [task queue](../terms/task-queue.md).

<!-- TODO: Switch to v1 activity naming convention -->

<!--SNIPSTART using-local-activities-example-->
[using_local_activities/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_local_activities/activity.go)
```go
func LocalActivityExample(ctx workflow.Context, input any) (any, error) {
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
	})
	var result any
	err := workflow.ExecuteLocalActivity(localCtx, QuickLookup, input).Get(ctx, &result)
	return result, err
}

```
<!--SNIPEND-->

Use local activities for fast, reliable operations that are extremely unlikely to fail: data validation, lightweight transformations, in-memory lookups. If you need to think about whether the operation will finish within the workflow task timeout, be impacted by an outage in a downstream service, use a normal activity instead.

See also: [Fallible Local Activities](../fallible_local_activities/).
