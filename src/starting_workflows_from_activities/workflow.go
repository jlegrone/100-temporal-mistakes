package starting_workflows_from_activities

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// MyInput represents the input to the workflow.
type MyInput struct{}

// MyResult represents the output of the child workflow.
type MyResult struct{}

// MyChildWorkflow is a child workflow.
func MyChildWorkflow(_ workflow.Context, _ MyInput) (MyResult, error) {
	return MyResult{}, nil
}

// SomeWorkflow is a workflow started from an activity (bad practice).
func SomeWorkflow(_ workflow.Context, _ MyInput) error {
	return nil
}

// @@@SNIPSTART starting-workflows-from-activities-good
// GOOD: child workflow from workflow code
func MyWorkflowV2(ctx workflow.Context, input MyInput) (MyResult, error) {
	childFuture := workflow.ExecuteChildWorkflow(ctx, MyChildWorkflow, input)
	var result MyResult
	err := childFuture.Get(ctx, &result)
	return result, err
}

// @@@SNIPEND

// @@@SNIPSTART starting-workflows-from-activities-bad
// BAD: starting a workflow from an activity
func MyActivity(ctx context.Context, input MyInput) error {
	c, _ := client.Dial(client.Options{})
	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, SomeWorkflow, input)
	return err
}

// @@@SNIPEND
