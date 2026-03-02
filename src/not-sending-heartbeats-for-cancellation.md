# Not Sending Heartbeats from Activities You Want to Cancel

> [!TIP]
> * Activity [cancellation](terms/cancellation.md) in Temporal is cooperative -- the server notifies the [worker](terms/worker.md) during the next [heartbeat](terms/heartbeat.md) response.
> * If an activity doesn't heartbeat, it won't learn about cancellation until it completes naturally.
> * For long-running activities where timely cancellation matters, regular heartbeating is essential.

## What?

Temporal does not forcefully interrupt running activities. Activity cancellation is a cooperative protocol: when a cancellation is requested (because the workflow was cancelled, a timeout fired, or the workflow explicitly cancelled the activity), the Temporal server records the cancellation. The worker learns about it the next time the activity sends a heartbeat. The server responds to that heartbeat with a cancellation indicator, and the activity's context is then cancelled.

If your activity never heartbeats, the cancellation request sits on the server with no way to reach the worker. The activity continues running to completion, blissfully unaware that it was supposed to stop. You effectively have no cancellation at all.

## Why?

This matters most for long-running activities -- anything that takes more than a few seconds. Consider these scenarios:

- **Workflow cancellation**: A user cancels a workflow, expecting all in-flight work to stop promptly. An activity without heartbeats keeps running for minutes or hours, consuming resources and producing side effects that should have been avoided.
- **Timeout-based cancellation**: You configure a `HeartbeatTimeout` expecting that a stuck activity will be detected and cancelled. But if the activity doesn't heartbeat, `HeartbeatTimeout` has no effect -- it only fires if the server expects heartbeats and stops receiving them.
- **Graceful shutdown**: When a worker is shutting down, it cancels in-flight activities. Activities that don't heartbeat won't check for cancellation and may be killed abruptly when the process exits, rather than getting a chance to clean up.

In all these cases, the root cause is the same: cancellation travels via the heartbeat channel, and if that channel is never used, the message is never delivered.

## How?

Add regular heartbeat calls to any activity where timely cancellation is important:

```go
func LongRunningActivity(ctx context.Context, input Input) error {
    for i, item := range input.Items {
        // Check for cancellation before each unit of work
        if ctx.Err() != nil {
            return ctx.Err()
        }

        // Do a unit of work
        processItem(item)

        // Heartbeat with progress -- this is where cancellation is detected
        activity.RecordHeartbeat(ctx, i)
    }
    return nil
}
```

### Guidelines

1. **Heartbeat at regular intervals.** A good rule of thumb is to heartbeat at least every few seconds for activities that run longer than 10 seconds. The exact interval depends on how quickly you need to respond to cancellation.

2. **Check `ctx.Err()` after heartbeating.** The heartbeat call cancels the context if the server responds with a cancellation signal. After `RecordHeartbeat`, check `ctx.Err()` or use the context in your next operation -- it fails with a `CanceledError` if cancellation was received.

3. **Set `HeartbeatTimeout` on the activity options.** Without a `HeartbeatTimeout`, the server has no expectation of heartbeats and won't detect a stuck activity. Set it to a value slightly larger than your expected heartbeat interval:

```go
activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
    StartToCloseTimeout: 1 * time.Hour,
    HeartbeatTimeout:    10 * time.Second,
})
```

4. **Use heartbeat details for progress tracking.** The data you pass to `RecordHeartbeat` is available on retry via `activity.GetHeartbeatDetails()`. This lets you resume from where you left off rather than starting over, which is especially valuable for long-running activities that process items in a loop.

5. **Don't heartbeat too aggressively.** Heartbeats are RPCs to the Temporal server. Heartbeating every millisecond adds unnecessary load. The SDK throttles heartbeats on the client side (typically to 80% of `HeartbeatTimeout`), but you should still be intentional about your heartbeat frequency.
