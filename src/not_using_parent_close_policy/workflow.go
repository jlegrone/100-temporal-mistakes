package not_using_parent_close_policy

import (
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

// Input represents the workflow input.
type Input struct{}

// @@@SNIPSTART not-using-parent-close-policy-workflow

func MyWorkflow(ctx workflow.Context, input Input) error {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})
	future := workflow.ExecuteChildWorkflow(childCtx, ChildWorkflow, input)
	return future.Get(childCtx, nil)
}

// @@@SNIPEND

// ChildWorkflow is a stub child workflow.
func ChildWorkflow(_ workflow.Context, _ Input) error { return nil }
