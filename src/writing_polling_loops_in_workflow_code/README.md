# Writing Polling Loops in Workflow Code

> [!TIP]
> Polling loops with `workflow.Sleep()` add timer events to [history](terms/event-history.md) on every iteration, bloating it over time. Use [signals](terms/signals.md) to push state changes, or offload polling to an activity.

If the condition change comes from an external system or workflow that you control, have it send a signal -- this consumes no additional resources while waiting and allows your workflow to be woken up in real time.

<!-- TODO: Just delete the signal code example. This is pretty self-evident. -->

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

If it is an external system that you do not control, then you can poll from an activity:

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
		activity.RecordHeartbeat(ctx, result)
		time.Sleep(30 * time.Second) // Regular time.Sleep, not workflow.Sleep
	}
}

```
<!--SNIPEND-->

For a very long polling back off durations, you can use activity retries as you're pulling mechanism instead in order to free of resources on your worker between polling attempts.
<!-- TODO: Add activity retry based polling example. The temporal application error returned by the activity should include structured result instead of putting that in the heartbeat details. -->
