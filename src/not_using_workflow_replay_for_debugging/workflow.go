package not_using_workflow_replay_for_debugging

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// GreetActivity returns a greeting.
func GreetActivity(_ context.Context, name string) (string, error) {
	return "Hello, " + name + "!", nil
}

// MyWorkflow is a simple workflow that calls an activity.
func MyWorkflow(ctx workflow.Context, name string) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})
	var result string
	err := workflow.ExecuteActivity(ctx, GreetActivity, name).Get(ctx, &result)
	return result, err
}
