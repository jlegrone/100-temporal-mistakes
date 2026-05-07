package modifying_shared_state_in_workflow_code

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// AlertActivity is a stub activity for demonstration purposes.
func AlertActivity(_ context.Context) error {
	return nil
}

// @@@SNIPSTART modifying-shared-state-bad

// BAD: shared mutable state
var processedCount int

func MyWorkflowV1(ctx workflow.Context) error {
	processedCount++ // Data race! Non-deterministic on replay!
	if processedCount > 100 {
		if err := workflow.ExecuteActivity(ctx, AlertActivity).Get(ctx, nil); err != nil {
			return err
		}
	}
	// ...
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART modifying-shared-state-good

// GOOD: local state
func MyWorkflowV2(ctx workflow.Context) error {
	processedCount := 0 // Local to this workflow execution
	_ = processedCount
	// ...
	return nil
}

// @@@SNIPEND
