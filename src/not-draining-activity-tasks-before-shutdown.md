# Not Draining Activity Tasks Before Shutdown

> [!TIP]
> * When a [worker](terms/worker.md) shuts down, in-flight activities need time to complete. Killing the worker immediately causes activities to time out and retry on another worker, wasting all progress.
> * Configure a graceful shutdown drain period that gives activities enough time to finish or checkpoint via [heartbeats](terms/heartbeat.md).
> * Activities should heartbeat regularly and check for [cancellation](terms/cancellation.md) so they can save progress when the worker is shutting down.

## What?

During deployments or scaling events, workers are stopped and replaced. If a worker is killed immediately, any activities currently executing on that worker are abandoned. The Temporal server won't know the activity failed until its heartbeat or start-to-close timeout expires. At that point, the activity is retried from scratch on another worker, throwing away all the work that was already done.

This is particularly painful for long-running activities (e.g., data migrations, file processing, ML training jobs) where losing 10-15 minutes of progress per deployment adds up fast.

## Why?

Temporal workers handle two types of tasks: [workflow tasks](terms/workflow-task.md) and activity tasks. Workflow tasks are inherently safe to interrupt because workflows are deterministic and will simply [replay](terms/replay.md) from history on the next worker. Activity tasks, however, represent actual side-effecting work. When an activity is interrupted mid-execution, there is no automatic recovery mechanism other than retrying from the beginning.

Without a drain period:
- Activities time out, adding unnecessary latency equal to the timeout duration before they are retried.
- Work already performed is lost and must be redone.
- If many workers are restarted simultaneously (e.g., a rolling deployment), you can end up with a cascade of timed-out activities overwhelming remaining workers with retries.

## How?

**Configure graceful shutdown on your worker.** Most Temporal SDKs support a graceful shutdown period. When the worker receives a stop signal (SIGTERM), it stops polling for new tasks but continues executing in-flight activities until the drain period expires.

In Go, for example:

```go
w := worker.New(c, "my-task-queue", worker.Options{
    // Give activities up to 5 minutes to finish after shutdown signal
    GracefulStopTimeout: 5 * time.Minute,
})
```

Make sure your deployment orchestrator (Kubernetes, ECS, etc.) gives the worker at least as much time as the drain period before force-killing it. In Kubernetes, set `terminationGracePeriodSeconds` to match or exceed your drain timeout.

**Use heartbeats for long-running activities.** Activities that heartbeat can detect cancellation (triggered by the worker shutting down) and save their progress. On retry, the activity can resume from the last heartbeat rather than starting over.

```go
func MyLongRunningActivity(ctx context.Context, input Input) error {
    for i, item := range input.Items {
        // Check if we should stop
        if ctx.Err() != nil {
            return ctx.Err()
        }
        // Record progress so we can resume later
        activity.RecordHeartbeat(ctx, i)
        // Do the actual work
        process(item)
    }
    return nil
}
```

**Match your drain timeout to your longest reasonable activity duration.** If your longest activity typically takes 3 minutes, a 5-minute drain period gives a comfortable margin. If you have activities that can run for hours, those activities should already be heartbeating and handling cancellation gracefully, so a shorter drain period is fine as long as the activity can checkpoint and exit promptly.
