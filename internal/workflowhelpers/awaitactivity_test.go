package workflowhelpers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/jlegrone/100-temporal-mistakes/internal/testhelpers"
	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
)

type echoRequest struct {
	Message string
}

type echoResponse struct {
	Message string
}

type echoWorker struct {
	failOnce bool
	calls    int
}

func (w *echoWorker) Echo(_ context.Context, req echoRequest) (*echoResponse, error) {
	w.calls++
	if w.failOnce && w.calls == 1 {
		return nil, errors.New("transient")
	}
	return &echoResponse{Message: req.Message}, nil
}

func TestAwaitActivity_ReturnsTypedResponse(t *testing.T) {
	worker := &echoWorker{}

	wf := func(ctx workflow.Context, req echoRequest) (*echoResponse, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    time.Second,
			HeartbeatTimeout:       time.Second,
		})
		return workflowhelpers.AwaitActivity(ctx, worker.Echo, req)
	}

	env := testhelpers.NewTestWorkflowEnvironment(t)
	env.RegisterActivity(worker.Echo)
	env.ExecuteWorkflow(wf, echoRequest{Message: "hello"})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var resp echoResponse
	require.NoError(t, env.GetWorkflowResult(&resp))
	require.Equal(t, "hello", resp.Message)
}

func TestAwaitActivity_PropagatesActivityError(t *testing.T) {
	worker := &echoWorker{}

	wf := func(ctx workflow.Context, req echoRequest) (*echoResponse, error) {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    time.Second,
			HeartbeatTimeout:       time.Second,
		})
		return workflowhelpers.AwaitActivity(ctx, worker.failingEcho, req)
	}

	env := testhelpers.NewTestWorkflowEnvironment(t)
	env.RegisterActivity(worker.failingEcho)
	env.ExecuteWorkflow(wf, echoRequest{Message: "hello"})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func (w *echoWorker) failingEcho(_ context.Context, _ echoRequest) (*echoResponse, error) {
	// Use a non-retryable application error so the activity fails immediately
	// without relying on a constrained MaximumAttempts retry policy.
	return nil, temporal.NewNonRetryableApplicationError("boom", "test", nil)
}
