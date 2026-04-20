# Assuming Activity Cancellation Means Workflow Cancellation

> [!TIP]
> Activities can be [cancelled](terms/cancellation.md) for many reasons -- [heartbeat timeout](terms/heartbeat-timeout.md), [start-to-close timeout](terms/start-to-close-timeout.md), [worker](terms/worker.md) shutdown, or explicit cancellation from the workflow -- not just workflow cancellation.

When an activity's context is cancelled, it's tempting to assume the parent workflow was cancelled. But activities can be interrupted for reasons unrelated to the workflow's lifecycle. If your cancellation handler assumes the workflow is done and acts accordingly (skipping compensation, not returning partial results), you'll introduce subtle bugs. The workflow may still be running and waiting for the activity to report back.

```go
func MyActivity(ctx context.Context, input Input) (Result, error) {
    for {
        // Do work in increments...
        activity.RecordHeartbeat(ctx, progressInfo)

        if ctx.Err() != nil {
            // Return partial results -- let the WORKFLOW decide what to do
            return Result{Partial: true, Progress: progressInfo}, ctx.Err()
        }
    }
}
```

The activity should not try to be smarter than the workflow. Report what happened, return what you have, and let the orchestration layer handle the rest.
