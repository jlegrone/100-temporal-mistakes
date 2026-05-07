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
func MyWorkflowV1(ctx workflow.Context, data []Record) error {
	// TODO: make this look less contrived -- maybe use bcrypt or some other expensive operation as an example? Or even sha2 (this might be more realistic, eg. to compute a checksum for the identity of a resouce being created in a subsequent activity).
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
func MyWorkflowV2(ctx workflow.Context, data []Record) error {
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
