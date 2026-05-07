# Not Sending Heartbeats for Cancellation

<!-- TODO: Activities that don't heartbeat will never observe cancelation when their parent workflow cancels them or the workflow itself is closed (canceled, terminated, or completed). The only way to detect cancelation is by sending heartbeats. Note that cancelation CAN still be observed when a worker is shutting down, even without heartbeats. -->

<!-- TODO: Add a test demonstrating that an activity never runs its cleanup logic if it doesn't heartbeat. Hopefully we can do that with the regular unit test environment instead of a full dev server. Use the normal MyActivityV1 and MyActivityV2 style to show and verify the two different behaviors. -->

<!-- TODO: write a test (using dev server) to confirm whether an auto-heartbeating interceptor that doesn't set heartbeat details overrides heartbeat details added by the activity implementation itself. It would be nice if an interceptor could automatically implement heartbeating (eg. if the activity timeout is > 30s). The interceptor would need to be paired with a workflow side interceptor that sets a heartbeat timeout. -->

<!-- TODO: is there an entry for not setting heartbeat timeouts? There should be. -->

> [!TIP]
> Activity [cancellation](terms/cancelation.md) is cooperative -- the server notifies the [worker](terms/worker.md) during the next [heartbeat](terms/heartbeat.md) response. If an activity doesn't heartbeat, it won't learn about cancellation until it completes naturally.

Temporal does not forcefully interrupt running activities. When cancellation is requested, the server records it and delivers the signal in the next heartbeat response. If your activity never heartbeats, cancellation sits on the server with no way to reach the worker. The activity continues running, consuming resources and producing side effects that should have been avoided.

<!--SNIPSTART not-sending-heartbeats-for-cancellation-->
[not_sending_heartbeats_for_cancellation/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_sending_heartbeats_for_cancellation/activity.go)
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
<!--SNIPEND-->

Set a `HeartbeatTimeout` on the activity options -- without one, the server has no expectation of heartbeats and won't detect a stuck activity. Heartbeat at least every few seconds for activities running longer than 10 seconds. The SDK throttles heartbeats on the client side (typically to 80% of `HeartbeatTimeout`), so don't worry about over-heartbeating.

See also: [Not Using Activity Heartbeat Details](../not_using_activity_heartbeat_details/README.md).
