package not_draining_signals_before_completing_workflow

import (
	"go.temporal.io/sdk/workflow"
)

// MySignal represents a signal payload.
type MySignal struct{}

// State holds the workflow state.
type State struct{}

// Apply processes a signal and updates the state.
func (s *State) Apply(_ MySignal) {}

// MyWorkflow is a placeholder for the ContinueAsNew target.
func MyWorkflow(_ workflow.Context, _ State) error {
	return nil
}

// @@@SNIPSTART not-draining-signals-drain-example

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

// @@@SNIPEND
