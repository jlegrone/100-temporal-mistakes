package not_validating_replay_safety_before_deployments

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type MyActivityRequest struct {
	RequestID string
}

type MyActivityResponse struct{}

// MyActivity is a stub activity used in replay testing examples.
func MyActivity(ctx context.Context, req MyActivityRequest) (*MyActivityResponse, error) {
	activity.GetLogger(ctx).Info("Hello from MyActivity", "RequestID", req.RequestID)
	return &MyActivityResponse{}, nil
}

// MyWorkflow is a stub workflow used in replay testing examples. It exercises
// a side effect and an activity so the recorded history contains both event
// types.
func MyWorkflow(ctx workflow.Context) error {
	workflow.GetLogger(ctx).Info("Hello Replay")

	requestID, err := workflowhelpers.SideEffect(ctx, func(workflow.Context) string {
		return uuid.NewString()
	})
	if err != nil {
		return err
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Minute,
	})

	if _, err := workflowhelpers.AwaitActivity(ctx, MyActivity, MyActivityRequest{RequestID: requestID}); err != nil {
		return err
	}

	return nil
}
