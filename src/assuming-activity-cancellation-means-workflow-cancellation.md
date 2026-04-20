# Assuming Activity Cancellation Means Workflow Cancellation

> [!TIP]
> Activities can be [cancelled](terms/cancellation.md) for many reasons -- [heartbeat timeout](terms/heartbeat-timeout.md), [start-to-close timeout](terms/start-to-close-timeout.md), [worker](terms/worker.md) shutdown, or explicit cancellation from the workflow -- not just workflow cancellation.

When an activity's context is cancelled, it's tempting to assume the parent workflow was cancelled. But activities can be interrupted for reasons unrelated to the workflow's lifecycle. If your cancellation handler assumes the workflow is done and acts accordingly, you'll introduce subtle bugs. The workflow may still be running and waiting for the activity to report back.

A common mistake is performing cleanup directly in the activity's cancellation handler when that cleanup should be orchestrated by the workflow:

```go
// BAD: activity assumes cancellation means the workflow is done
func ProcessOrderActivity(ctx context.Context, orderID string) error {
    err := startProcessing(ctx, orderID)
    if err != nil {
        return err
    }

    // Poll for completion, checking for cancellation
    for !isComplete(ctx, orderID) {
        activity.RecordHeartbeat(ctx, orderID)
        if err := ctx.Err(); err != nil {
            if errors.Is(ctx.Err(), context.Canceled) {
                // Wrong: assumes the whole workflow is cancelled,
                // so tries to clean up directly. But the workflow
                // may have just timed out this activity and wants
                // to retry or take a different path.
                db.UpdateStatus(ctx, orderID, "cancelled")
            }
            return err
        }
        time.Sleep(time.Second)
    }
    return nil
}
```

The activity should not try to be smarter than the workflow. Let the workflow decide what cancellation means and orchestrate any cleanup. The activity should simply stop work and return the cancellation error.
