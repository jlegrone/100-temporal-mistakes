package polling_workflow_results

import (
	"context"

	"go.temporal.io/sdk/client"
)

// MyResult is the result type returned by the workflow.
type MyResult struct{}

// @@@SNIPSTART polling-workflow-results-good

// Good: use the SDK's blocking GetWorkflow method.
func GetWorkflowResult(ctx context.Context, c client.Client, workflowID, runID string) (MyResult, error) {
	run := c.GetWorkflow(ctx, workflowID, runID)
	var result MyResult
	err := run.Get(ctx, &result)
	// Blocks until completion, failure, or context cancellation
	return result, err
}

// @@@SNIPEND
