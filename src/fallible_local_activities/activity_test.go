package fallible_local_activities

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// alwaysFails simulates an unreliable external call.
func alwaysFails(_ context.Context, _ any) error {
	return fmt.Errorf("service unavailable")
}

// slowActivity simulates a slow external call.
func slowActivity(ctx context.Context, _ any) error {
	select {
	case <-time.After(30 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// failingLocalActivityWorkflow calls alwaysFails as a local activity.
func failingLocalActivityWorkflow(ctx workflow.Context, request any) error {
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 5,
			InitialInterval: time.Second,
		},
	})
	return workflow.ExecuteLocalActivity(localCtx, alwaysFails, request).Get(ctx, nil)
}

// slowLocalActivityWorkflow calls slowActivity as a local activity.
func slowLocalActivityWorkflow(ctx workflow.Context, request any) error {
	localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Second,
	})
	return workflow.ExecuteLocalActivity(localCtx, slowActivity, request).Get(ctx, nil)
}

// @@@SNIPSTART fallible-local-activities-test

func TestLocalActivityRetriesExhausted(t *testing.T) {
	c, taskQueue := testsuite.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(failingLocalActivityWorkflow)
		r.RegisterActivity(alwaysFails)
	})

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue:          taskQueue,
		WorkflowRunTimeout: time.Minute,
	}, failingLocalActivityWorkflow, "request")
	require.NoError(t, err)
	require.ErrorContains(t, run.Get(t.Context(), nil), "service unavailable")
}

func TestLocalActivityTooSlow(t *testing.T) {
	c, taskQueue := testsuite.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(slowLocalActivityWorkflow)
		r.RegisterActivity(slowActivity)
	})

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue:          taskQueue,
		WorkflowRunTimeout: time.Minute,
	}, slowLocalActivityWorkflow, "request")
	require.NoError(t, err)
	require.ErrorContains(t, run.Get(t.Context(), nil), "deadline exceeded")
}

// @@@SNIPEND
