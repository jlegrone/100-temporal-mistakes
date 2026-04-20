# Writing Polling Loops in Workflow Code

> [!TIP]
> Polling loops with `workflow.Sleep()` add timer events to [history](terms/event-history.md) on every iteration, bloating it over time. Use [signals](terms/signals.md) to push state changes, or offload polling to an activity.

A loop that periodically checks a condition via `workflow.Sleep()` generates roughly 2,880 timer event pairs per day, plus activity events for each check. A workflow polling for a week can easily exceed the 50,000 event [history limit](overflowing-workflow-history-length.md).

If the condition change comes from an external system, have it send a signal -- this adds zero events while waiting:

<!--SNIPSTART writing-polling-loops-in-workflow-code-signal-->
[writing_polling_loops_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/writing_polling_loops_in_workflow_code/workflow.go)
```go

// WaitForStatusUpdate waits for a signal instead of polling,
// adding zero events to the history while waiting.
func WaitForStatusUpdate(ctx workflow.Context) (Status, error) {
	var status Status
	signalChan := workflow.GetSignalChannel(ctx, "status-update")
	signalChan.Receive(ctx, &status)
	return status, nil
}

```
<!--SNIPEND-->

If you need a timeout alongside a signal, use `workflow.NewSelector` to combine a signal channel with a timer. If you genuinely need to poll an external system, poll inside an activity with [heartbeats](terms/heartbeat.md) -- the activity can poll as frequently as needed without adding events to workflow history:

<!--SNIPSTART writing-polling-loops-in-workflow-code-activity-->
[writing_polling_loops_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/writing_polling_loops_in_workflow_code/workflow.go)
```go

// PollUntilReady polls an external system inside an activity with
// heartbeats. The activity can poll as frequently as needed without
// adding events to workflow history.
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
<!--SNIPEND-->

For very long polling durations, combine the activity approach with [ContinueAsNew](terms/continue-as-new.md).
