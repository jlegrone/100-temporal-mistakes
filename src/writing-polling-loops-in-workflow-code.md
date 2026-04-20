# Writing Polling Loops in Workflow Code

> [!TIP]
> Polling loops with `workflow.Sleep()` add timer events to [history](terms/event-history.md) on every iteration, bloating it over time. Use [signals](terms/signals.md) to push state changes, or offload polling to an activity.

A loop that periodically checks a condition via `workflow.Sleep()` generates roughly 2,880 timer event pairs per day, plus activity events for each check. A workflow polling for a week can easily exceed the 50,000 event [history limit](overflowing-workflow-history-size.md).

If the condition change comes from an external system, have it send a signal -- this adds zero events while waiting:

```go
// Good: wait for a signal instead of polling
signalChan := workflow.GetSignalChannel(ctx, "status-update")
signalChan.Receive(ctx, &status)
```

If you need a timeout alongside a signal, use `workflow.NewSelector` to combine a signal channel with a timer. If you genuinely need to poll an external system, poll inside an activity with [heartbeats](terms/heartbeat.md) -- the activity can poll as frequently as needed without adding events to workflow history:

```go
func PollUntilReady(ctx context.Context) (Result, error) {
    for {
        result, err := checkExternalSystem()
        if err != nil {
            return Result{}, err
        }
        if result.Ready {
            return result, nil
        }
        activity.RecordHeartbeat(ctx, result.Status)
        time.Sleep(30 * time.Second) // Regular time.Sleep, not workflow.Sleep
    }
}
```

For very long polling durations, combine the activity approach with [ContinueAsNew](terms/continue-as-new.md).
