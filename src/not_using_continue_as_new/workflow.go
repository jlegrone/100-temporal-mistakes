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

		// Check if it's time to continue as new
		if workflow.GetInfo(ctx).GetCurrentHistoryLength() > 10000 {
			return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
		}
	}
}

// @@@SNIPEND
