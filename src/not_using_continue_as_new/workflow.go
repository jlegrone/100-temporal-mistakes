package not_using_continue_as_new

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// SubscriptionState holds the state carried across ContinueAsNew boundaries.
type SubscriptionState struct{}

// @@@SNIPSTART not-using-continue-as-new-workflow

func SubscriptionWorkflow(ctx workflow.Context, state SubscriptionState) error {
	var workflowAgedOut bool

	sel := workflow.NewSelector(ctx)
	sel.AddFuture(workflow.NewTimer(ctx, 24*time.Hour), func(f workflow.Future) {
		workflowAgedOut = true
	})
	// Add additional branches to selector ...

	for sel.HasPending() {
		sel.Select(ctx)

		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() || workflowAgedOut {
			return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
		}

		// ...
	}
	return nil
}

// @@@SNIPEND
