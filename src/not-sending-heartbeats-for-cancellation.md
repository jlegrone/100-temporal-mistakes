# Not Sending Heartbeats for Cancellation

> [!TIP]
> Activity [cancellation](terms/cancelation.md) is cooperative -- the server notifies the [worker](terms/worker.md) during the next [heartbeat](terms/heartbeat.md) response. If an activity doesn't heartbeat, it won't learn about cancellation until it completes naturally.

Temporal does not forcefully interrupt running activities. When cancellation is requested, the server records it and delivers the signal in the next heartbeat response. If your activity never heartbeats, cancellation sits on the server with no way to reach the worker. The activity continues running, consuming resources and producing side effects that should have been avoided.

```go
func LongRunningActivity(ctx context.Context, input Input) error {
    for i, item := range input.Items {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        processItem(item)
        activity.RecordHeartbeat(ctx, i) // Cancellation is detected here
    }
    return nil
}
```

Set a `HeartbeatTimeout` on the activity options -- without one, the server has no expectation of heartbeats and won't detect a stuck activity. Heartbeat at least every few seconds for activities running longer than 10 seconds. The SDK throttles heartbeats on the client side (typically to 80% of `HeartbeatTimeout`), so don't worry about over-heartbeating.

See also: [Not Using Activity Heartbeat Details](not-using-activity-heartbeat-details.md).
