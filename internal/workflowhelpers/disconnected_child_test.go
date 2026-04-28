package workflowhelpers_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
)

type cleanupRequest struct {
	ID string
}

type cleanupResponse struct{}

func cleanupWorkflow(ctx workflow.Context, req cleanupRequest) (*cleanupResponse, error) {
	workflow.GetLogger(ctx).Info("cleanup ran", "ID", req.ID)
	return &cleanupResponse{}, nil
}

func TestStartDisconnectedChildWorkflow_StartsChildAndReturns(t *testing.T) {
	parent := func(ctx workflow.Context) error {
		// Cancel the parent immediately to prove the disconnected child still runs.
		return workflowhelpers.StartDisconnectedChildWorkflow(
			ctx,
			cleanupWorkflow,
			cleanupRequest{ID: "1"},
			workflow.ChildWorkflowOptions{WorkflowID: "cleanup-1"},
		)
	}

	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(cleanupWorkflow)

	env.ExecuteWorkflow(parent)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestStartDisconnectedChildWorkflow_RunsAfterParentCancelation(t *testing.T) {
	parent := func(ctx workflow.Context) error {
		// Wait until canceled, then start the cleanup child.
		_ = workflow.Await(ctx, func() bool { return false })
		return workflowhelpers.StartDisconnectedChildWorkflow(
			ctx,
			cleanupWorkflow,
			cleanupRequest{ID: "2"},
			workflow.ChildWorkflowOptions{WorkflowID: "cleanup-2"},
		)
	}

	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(cleanupWorkflow)
	env.RegisterDelayedCallback(env.CancelWorkflow, 0)

	env.ExecuteWorkflow(parent)
	require.True(t, env.IsWorkflowCompleted())
	// The parent waited on Await(false) until canceled, then called
	// StartDisconnectedChildWorkflow. A nil error proves the disconnected
	// child started despite the parent's cancelation.
	require.NoError(t, env.GetWorkflowError())
}
