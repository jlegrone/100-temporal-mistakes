# Writing Polling Loops in Workflow Code

> [!TIP]
> * Polling loops using `workflow.Sleep()` add timer events to the [history](terms/event-history.md) on every iteration, bloating it over time.
> * Use [signals](terms/signals.md) or [updates](terms/updates.md) to push state changes to the workflow instead of polling for them.
> * For long-running polling needs, use an activity or child workflow with [ContinueAsNew](terms/continue-as-new.md).

## What?

A common anti-pattern is writing a loop in workflow code that periodically checks a condition using `workflow.Sleep()`:

```go
// Bad: polling loop in workflow code
for {
    result, err := workflow.ExecuteActivity(ctx, CheckCondition).Get(ctx, nil)
    if result.Done {
        break
    }
    workflow.Sleep(ctx, 30 * time.Second)
}
```

Each iteration of this loop adds events to the workflow history: at minimum a `TimerStarted` and `TimerFired` event pair for each sleep, plus the activity-related events for each check. Over hours or days, this bloats the history toward the [maximum size limit](overflowing-workflow-history-size.md).

## Why?

Temporal workflows are event-sourced. Every operation -- timers, activity calls, [child workflows](terms/child-workflow.md) -- adds events to the history. This history is persisted and [replayed](terms/replay.md) when the workflow must recover.

A polling loop that runs every 30 seconds generates roughly 2,880 timer event pairs per day, plus the activity events. A workflow polling for a week could easily accumulate tens of thousands of events, approaching or exceeding the default 50,000 event limit.

Beyond the hard limit, large histories also slow down replay, increase storage costs, and put more load on the Temporal server.

## How?

The right approach depends on what you're waiting for:

### Push instead of poll with signals

If the condition change originates from an external system, have that system send a [signal](terms/signals.md) to the workflow:

```go
// Good: wait for a signal instead of polling
var status string
signalChan := workflow.GetSignalChannel(ctx, "status-update")
signalChan.Receive(ctx, &status)
```

This adds zero events to the history while waiting. The signal itself adds events only when it arrives.

### Use `workflow.Sleep()` with a reasonable timeout

If you need a timeout alongside a signal (e.g., "wait for approval, but time out after 24 hours"), combine them:

```go
selector := workflow.NewSelector(ctx)
selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
    c.Receive(ctx, &status)
})
selector.AddFuture(workflow.NewTimer(ctx, 24*time.Hour), func(f workflow.Future) {
    status = "timed-out"
})
selector.Select(ctx)
```

### Offload polling to an activity

If you genuinely need to poll an external system, poll inside an activity with [heartbeats](terms/heartbeat.md). The activity can poll as frequently as needed without adding events to the workflow history:

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

For very long polling durations, combine the activity approach with [ContinueAsNew](terms/continue-as-new.md) to keep the workflow history bounded.
