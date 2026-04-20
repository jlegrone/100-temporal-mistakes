package reading_environment_variables_in_workflow_code

import (
	"os"

	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART reading-env-vars-bad

// BAD: reading env var in workflow code
func MyWorkflowV1(ctx workflow.Context) error {
	region := os.Getenv("AWS_REGION")
	if region == "us-east-1" {
		// Route to US activities
	}
	// ...
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART reading-env-vars-good

// GOOD: pass configuration as workflow input
type WorkflowInput struct {
	Region string
}

func MyWorkflowV2(ctx workflow.Context, input WorkflowInput) error {
	if input.Region == "us-east-1" {
		// Deterministic -- value is recorded in the start event
	}
	// ...
	return nil
}

// @@@SNIPEND
