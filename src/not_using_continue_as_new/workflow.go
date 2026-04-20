package not_using_continue_as_new

import (
	"go.temporal.io/sdk/workflow"
)

// SubscriptionState holds the state carried across ContinueAsNew boundaries.
type SubscriptionState struct{}

// @@@SNIPSTART not-using-continue-as-new-workflow

func SubscriptionWorkflow(ctx workflow.Context, state SubscriptionState) error {
	for {
		// ... do work ...

		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
		}
	}
}

// @@@SNIPEND
