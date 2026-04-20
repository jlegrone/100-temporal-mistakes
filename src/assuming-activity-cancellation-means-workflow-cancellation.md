# Assuming Activity Cancellation Means Workflow Cancellation

> [!TIP]
> Activities can be [canceled](terms/cancelation.md) for many reasons -- [heartbeat timeout](terms/heartbeat-timeout.md), [start-to-close timeout](terms/start-to-close-timeout.md), [worker](terms/worker.md) shutdown, or explicit cancelation from the workflow -- not just workflow cancelation.

When an activity's context is canceled, it's tempting to assume the parent workflow was canceled. But activities can be interrupted for reasons unrelated to the workflow's lifecycle (eg. during deployment or scaledown of a worker pool). If your cancelation handler assumes the workflow is done and acts accordingly, you'll introduce subtle bugs. The workflow may still be running and waiting for the activity to report back.

A common mistake is performing cleanup directly in the activity's cancelation handler when that cleanup should be orchestrated by the workflow:

```go
// BAD: activity assumes cancelation means the workflow is done
func ProcessOrderActivity(ctx context.Context, orderID string) error {
    if err := startProcessing(ctx, orderID); err != nil {
        return err
    }

    // Poll for completion, checking for cancelation
    for !isComplete(ctx, orderID) {
        activity.RecordHeartbeat(ctx, orderID)

        // Wrong: assumes the whole workflow is canceled,
        // so tries to clean up directly. But the workflow
        // may have just timed out this activity and wants
        // to retry or take a different path.
        if err := ctx.Err(); err != nil {
            if errors.Is(ctx.Err(), context.Canceled) {
                db.UpdateStatus(ctx, orderID, "canceled")
            }
            return err
        }

        time.Sleep(time.Second)
    }

    return nil
}
```

The activity should not try to be smarter than the workflow. Let the workflow decide what cancelation means and orchestrate any cleanup. The activity should simply stop work and return the cancelation error.
