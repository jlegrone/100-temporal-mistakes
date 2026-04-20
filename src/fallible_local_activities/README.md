# Fallible Local Activities

> [!TIP]
> [Local activity](../terms/local-activity.md) retries count against the [workflow task](../terms/workflow-task.md) timeout. If retries take too long, the workflow task times out and the entire cycle restarts from scratch -- creating a livelock.

Local activities retry within the same workflow task, which has a default 10-second timeout. When a local activity fails and retries with backoff, each attempt eats into that budget. If retries exceed the timeout, the workflow task times out, gets rescheduled, [replays](../terms/replay.md), hits the local activity again, and the retry cycle restarts from zero. The local activity never gets enough time to exhaust its retries.

<!--SNIPSTART fallible-local-activities-good-->
[fallible_local_activities/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/fallible_local_activities/activity.go)
```go
func GoodLocalActivity(ctx workflow.Context, input any) (any, error) {
	// Good: local activity for a fast, reliable operation
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 2 * time.Second,
	})
	var result any
	err := workflow.ExecuteLocalActivity(localCtx, ValidateInput, input).Get(ctx, &result)
	return result, err
}

```
<!--SNIPEND-->

<!--SNIPSTART fallible-local-activities-bad-->
[fallible_local_activities/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/fallible_local_activities/activity.go)
```go
func BadLocalActivity(ctx workflow.Context, request any) error {
	// Bad: local activity for an unreliable external call
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
			InitialInterval: time.Second,
		},
	})
	return workflow.ExecuteLocalActivity(localCtx, CallExternalAPI, request).Get(ctx, nil)
}

```
<!--SNIPEND-->

Use local activities only for operations expected to succeed quickly and reliably. If the operation calls an external service with variable latency, or needs a robust [retry policy](../terms/retry-policy.md), use a regular activity instead.

See also: [Using Local Activities](../using_local_activities/).
