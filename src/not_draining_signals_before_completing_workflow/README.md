# Not Draining Signals Before Completing a Workflow

<!-- TODO: If your workflow accepts multiple signals, it is possible not to observe some of them if your workflow completes or continues as new without reading all signals from the channel first. -->

> [!TIP]
> If a workflow completes or calls [ContinueAsNew](../terms/continue-as-new.md) while [signals](../terms/signals.md) are buffered in the channel, those signals are silently lost. Drain the channel before completing.

Signals are recorded in [history](../terms/event-history.md) and the API call succeeds, so the sender believes the signal was delivered. But if the workflow returns before processing buffered signals, they're dropped. This creates silent data loss with no errors or alerts.

Before completing or calling ContinueAsNew, use non-blocking receive to drain pending signals:

<!-- TODO: the code example here should have a before and after example. Also be sure to mention the https://pkg.go.dev/go.temporal.io/sdk@v1.42.0/workflow#GetUnhandledSignalNames function (could be used in an interceptor to automatically monitor workflows for this problem). It also ought to be possible to look at old workflow histories to check if there were unhandled signals (by replaying the workflow with the interceptor turned on and inspecting the logs) -->
<!--SNIPSTART not-draining-signals-drain-example-->
[not_draining_signals_before_completing_workflow/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_draining_signals_before_completing_workflow/workflow.go)
```go

// DrainSignalsBeforeContinueAsNew drains pending signals before calling ContinueAsNew.
func DrainSignalsBeforeContinueAsNew(ctx workflow.Context, signalCh workflow.ReceiveChannel, state State) error {
	if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
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
