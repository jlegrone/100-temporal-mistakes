# Fallible Local Activities

> [!TIP]
> * Local activities that fail are retried within the same workflow task, counting against the workflow task timeout.
> * If retries take too long, the workflow task times out and the entire workflow task -- including the local activity -- is retried from scratch.
> * Use local activities only for operations expected to succeed quickly and reliably.

## What?

Local activities are a performance optimization that bypasses the normal activity task queue. Instead of scheduling an activity on the server and having a worker pick it up, a local activity executes directly within the current workflow task on the same worker. This eliminates the round trip to the server, making local activities ideal for short, fast operations.

The mistake is using local activities for operations that can fail or take a long time to retry. The retry behavior of local activities is fundamentally different from regular activities, and misunderstanding this leads to surprising failures.

## Why?

Regular activities and local activities handle retries very differently:

**Regular activities** are scheduled as independent tasks. When a regular activity fails, the retry happens as a new task picked up by a worker. The workflow task that scheduled the activity has already completed, so there's no timeout pressure.

**Local activities** execute within the bounds of a workflow task. A workflow task has a default timeout of 10 seconds (see [exceeding the workflow task timeout](exceeding-10s-task-timeout.md)). When a local activity fails and retries, each retry attempt happens within the same workflow task, eating into that timeout budget.

Here's what happens when retries exceed the workflow task timeout:

1. The local activity fails and begins retrying within the workflow task.
2. The retries (including backoff delays) consume more time than the workflow task timeout allows.
3. The workflow task times out.
4. The Temporal server reschedules the workflow task on a worker.
5. The workflow replays, hits the local activity again, and starts the retry cycle over from scratch.

This creates a loop where the local activity never makes progress. The retries always start over because the workflow task keeps timing out.

## How?

Follow these guidelines for [using local activities](using-local-activities.md) safely:

1. **Only use local activities for operations that are expected to succeed quickly.** Good candidates: lightweight computations, in-memory cache lookups, reading local configuration, fast network calls to highly available services.

2. **Keep retry policies minimal or absent.** If you set a retry policy on a local activity, ensure the total time across all retry attempts (including backoff) stays well under the workflow task timeout.

3. **Use regular activities for anything that might fail.** If the operation calls an external service that could be down, involves network I/O with unpredictable latency, or needs a robust retry policy -- use a regular activity instead.

4. **Monitor workflow task timeouts.** A spike in `workflow_task_schedule_to_start_latency` or `workflow_task_execution_failed` metrics can indicate local activities are causing workflow task timeouts.

```go
// Good: local activity for a fast, reliable operation
localActivityOpts := workflow.LocalActivityOptions{
    ScheduleToCloseTimeout: 2 * time.Second,
}
localCtx := workflow.WithLocalActivityOptions(ctx, localActivityOpts)
err := workflow.ExecuteLocalActivity(localCtx, ValidateInput, input).Get(ctx, &result)

// Bad: local activity for an unreliable external call
localActivityOpts := workflow.LocalActivityOptions{
    ScheduleToCloseTimeout: 30 * time.Second,
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 10,
        InitialInterval: time.Second,
    },
}
localCtx := workflow.WithLocalActivityOptions(ctx, localActivityOpts)
err := workflow.ExecuteLocalActivity(localCtx, CallExternalAPI, request).Get(ctx, &result)
```

In the "bad" example, if the external API is down, the local activity retries with backoff within the workflow task. After 10 seconds, the workflow task times out, and the whole cycle restarts -- the local activity never gets enough time to exhaust its retries and the workflow is stuck in a retry loop.
