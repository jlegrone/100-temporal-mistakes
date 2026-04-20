# Not Draining Signals Before Completing a Workflow

> [!TIP]
> If a workflow completes or calls [ContinueAsNew](../terms/continue-as-new.md) while [signals](../terms/signals.md) are buffered in the channel, those signals are silently lost. Drain the channel before completing.

Signals are recorded in [history](../terms/event-history.md) and the API call succeeds, so the sender believes the signal was delivered. But if the workflow returns before processing buffered signals, they're dropped. This creates silent data loss with no errors or alerts.

Before completing or calling ContinueAsNew, use non-blocking receive to drain pending signals:

<!--SNIPSTART not-draining-signals-drain-example-->
[not_draining_signals_before_completing_workflow/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_draining_signals_before_completing_workflow/workflow.go)
```go

// DrainSignalsBeforeContinueAsNew drains pending signals before calling ContinueAsNew.
func DrainSignalsBeforeContinueAsNew(ctx workflow.Context, signalCh workflow.ReceiveChannel, state State) error {
	// Check if it's time to continue as new
	if ShouldContinueAsNew(ctx) {
		// Drain any remaining signals before continuing
		for {
			var signal MySignal
			ok := signalCh.ReceiveAsync(&signal)
			if !ok {
				break
			}
			state.Apply(signal)
		}
		return workflow.NewContinueAsNewError(ctx, MyWorkflow, state)
	}
	return nil
}

```
<!--SNIPEND-->

Apply the signals to your state -- don't just read and discard them. Drain as the last step before returning, and make sure all completion paths (success, error, ContinueAsNew) include draining where applicable.
