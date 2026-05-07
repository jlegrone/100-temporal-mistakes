package fallible_local_activities

import (
	"context"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ValidateInput is a fast, reliable local activity.
func ValidateInput(_ context.Context, _ any) (any, error) {
	return nil, nil
}

// CallExternalAPI is an unreliable external call that should not be a local activity.
func CallExternalAPI(_ context.Context, _ any) error {
	return nil
}

// @@@SNIPSTART fallible-local-activities-good
func MyWorkflowV2(ctx workflow.Context, input any) (any, error) {
	// Good: use a local activity for a fast, reliable operation
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 2 * time.Second,
	})
	var result any
	err := workflow.ExecuteLocalActivity(localCtx, ValidateInput, input).Get(ctx, &result)
	return result, err
}

// @@@SNIPEND

// @@@SNIPSTART fallible-local-activities-bad
func MyWorkflowV1(ctx workflow.Context, request any) error {
	// Bad: use a local activity for an unreliable external call
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
			InitialInterval: time.Second,
		},
	})
	return workflow.ExecuteLocalActivity(localCtx, CallExternalAPI, request).Get(ctx, nil)
}

// @@@SNIPEND
