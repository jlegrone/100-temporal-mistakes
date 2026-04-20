package performing_expensive_computation_in_workflow_code

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// Record represents a data record to be transformed.
type Record struct{}

// Result represents the output of a transformation.
type Result struct{}

func expensiveTransformation(_ []Record) Result {
	return Result{}
}

// StoreResult is an activity that persists the result.
func StoreResult(_ context.Context, _ Result) error {
	return nil
}

// @@@SNIPSTART expensive-computation-bad
// BAD: expensive computation in workflow code
func MyWorkflowBad(ctx workflow.Context, data []Record) error {
	result := expensiveTransformation(data) // Takes 30 seconds
	return workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
}

// @@@SNIPEND

// @@@SNIPSTART expensive-computation-good
// TransformActivity moves the expensive computation into an activity.
func TransformActivity(_ context.Context, data []Record) (Result, error) {
	return expensiveTransformation(data), nil
}

// GOOD: move it to an activity
func MyWorkflowGood(ctx workflow.Context, data []Record) error {
	var result Result
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	})
	if err := workflow.ExecuteActivity(ctx, TransformActivity, data).Get(ctx, &result); err != nil {
		return err
	}
	return workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
}

// @@@SNIPEND
