# Not Draining Signals Before Completing a Workflow

> [!TIP]
> * If a workflow completes (returns or calls [ContinueAsNew](terms/continue-as-new.md)) while there are unprocessed [signals](terms/signals.md) in its queue, those signals are lost.
> * This is especially problematic in patterns where external systems send signals expecting them to be processed.
> * Before completing or continuing as new, drain the signal channel to process any pending signals.

## What?

[Signals](terms/signals.md) are asynchronous messages sent to a running workflow. They are appended to the workflow's history and delivered to the workflow code via signal channels. However, if the workflow completes -- either by returning a result, returning an error, or calling [ContinueAsNew](terms/continue-as-new.md) -- any signals that are sitting in the channel buffer but have not been received by the workflow code are silently dropped.

This is a subtle issue because it creates a race condition: a signal sender believes the signal was delivered (the API call succeeded and the signal was recorded in the history), but the workflow never processes it.

## Why?

The consequences depend on what the signals represent:

**Data loss**: If signals carry commands or data (e.g. "add item to cart", "update subscription", "cancel order"), unprocessed signals mean lost user actions. The sender has no indication that the signal was not handled.

**Inconsistent state**: When using [ContinueAsNew](terms/continue-as-new.md), the new workflow execution starts without knowledge of the signals that were dropped in the previous execution. This creates a gap in the workflow's understanding of the world.

**Silent failures**: Unlike activity failures or workflow errors, dropped signals don't produce errors or alerts. The signal API call returns success because the signal was recorded in the history. The problem only manifests as missing side effects, which can be very hard to debug.

## How?

Before completing a workflow or calling ContinueAsNew, drain the signal channel to process any buffered signals. Here is a Go example:

```go
func MyWorkflow(ctx workflow.Context, state MyState) error {
    signalCh := workflow.GetSignalChannel(ctx, "my-signal")

    for {
        // Main workflow logic: wait for signals or other events
        selector := workflow.NewSelector(ctx)
        selector.AddReceive(signalCh, func(ch workflow.ReceiveChannel, more bool) {
            var signal MySignal
            ch.Receive(ctx, &signal)
            state.Apply(signal)
        })
        // ... other selector cases ...
        selector.Select(ctx)

        // Check if it's time to continue as new
        if shouldContinueAsNew(ctx) {
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
    }
}
```

Key points:

**Use non-blocking receive for draining**: The `ReceiveAsync` method (or equivalent in your SDK) checks the channel without blocking. If there are buffered signals, it returns them. If the channel is empty, it returns immediately. This ensures the drain loop terminates.

**Apply the signals to your state**: Don't just read and discard the signals. Process them the same way you would during normal execution. The state you pass to ContinueAsNew (or use before returning) should reflect all received signals.

**Drain right before completing**: The drain should happen as the last step before the workflow returns or calls ContinueAsNew. Any signals that arrive after the drain but before the completion will be handled by Temporal's built-in mechanism: if a signal is delivered to a completed workflow and the workflow used ContinueAsNew, the signal is delivered to the new execution.

**Consider this for all completion paths**: If your workflow can complete in multiple ways (success, error, ContinueAsNew), make sure all paths include signal draining where applicable.
