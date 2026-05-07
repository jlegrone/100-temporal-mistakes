package using_local_activities

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// QuickLookup is a fast, reliable operation suitable for a local activity.
func QuickLookup(_ context.Context, _ any) (any, error) {
	return nil, nil
}

// @@@SNIPSTART using-local-activities-example
func LocalActivityExample(ctx workflow.Context, input any) (any, error) {
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
	})
	var result any
	err := workflow.ExecuteLocalActivity(localCtx, QuickLookup, input).Get(ctx, &result)
	return result, err
}

// @@@SNIPEND
