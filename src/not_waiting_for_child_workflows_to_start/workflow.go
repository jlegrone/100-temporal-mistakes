package not_waiting_for_child_workflows_to_start

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// Input represents the workflow input.
type Input struct{}

// @@@SNIPSTART not-waiting-for-child-workflows-to-start-workflow

func MyWorkflow(disconnectedCtx workflow.Context, input Input) error {
	// Start the cleanup child workflow
	childFuture := workflow.ExecuteChildWorkflow(disconnectedCtx, CleanupWorkflow, input)

	// Wait for the child to be scheduled -- this is the critical step
	if err := childFuture.GetChildWorkflowExecution().Get(disconnectedCtx, nil); err != nil {
		return fmt.Errorf("failed to start cleanup workflow: %w", err)
	}
	// Now safe to return -- the child runs independently
	return nil
}

// @@@SNIPEND

// CleanupWorkflow is a stub child workflow.
func CleanupWorkflow(_ workflow.Context, _ Input) error { return nil }
