package not_using_workflow_versioning

import (
	"go.temporal.io/sdk/workflow"
)

// Input represents the workflow input.
type Input struct{}

// Result represents the activity result.
type Result struct{}

// @@@SNIPSTART not-using-workflow-versioning-workflow

func MyWorkflow(ctx workflow.Context, input Input) error {
	var result Result
	var err error

	v := workflow.GetVersion(ctx, "change-id", workflow.DefaultVersion, 1)
	if v == workflow.DefaultVersion {
		// Old code path: existing workflows execute this
		err = workflow.ExecuteActivity(ctx, OldActivity, input).Get(ctx, &result)
	} else {
		// New code path: new workflows execute this
		err = workflow.ExecuteActivity(ctx, NewActivity, input).Get(ctx, &result)
	}

	return err
}

// @@@SNIPEND

// OldActivity is a stub activity.
func OldActivity(_ Input) (Result, error) { return Result{}, nil }

// NewActivity is a stub activity.
func NewActivity(_ Input) (Result, error) { return Result{}, nil }
