# Using Local Activities

> [!TIP]
> [Local activities](../terms/local-activity.md) skip the server round-trip but run within the [workflow task](../terms/workflow-task.md) timeout (default 10s). If they take too long or fail, the entire workflow task is retried from scratch.

Local activities execute directly within the current workflow task on the same [worker](../terms/worker.md), eliminating the round-trip to the server and reducing [history](../terms/event-history.md) size. This sounds like a pure win, but they come with important constraints:

- **Bound by the workflow task timeout**: If a local activity exceeds the remaining time in the workflow task (default 10s), the task times out, gets rescheduled, [replays](../terms/replay.md), and the local activity runs again from scratch -- potentially creating an infinite loop.
- **No independent visibility**: Local activities don't generate their own history events until the workflow task completes. You can't see them in the Temporal UI mid-execution.
- **No load balancing**: They run on the current worker only, unlike normal activities dispatched through the [task queue](../terms/task-queue.md).

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

Use local activities for fast, reliable operations: data transformations, in-memory lookups, lightweight validations. If you need to think about whether the operation will finish within the workflow task timeout, use a normal activity instead.

See also: [Fallible Local Activities](../fallible_local_activities/).
