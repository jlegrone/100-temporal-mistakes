package fallible_local_activities

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"
	"github.com/jlegrone/100-temporal-mistakes/internal/testhelpers"
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

// allowLongLocalActivity downgrades the local-activity start-to-close policy
// to Warn so this mistake's examples (which intentionally omit
// start_to_close_timeout to keep the focus on retry/timeout failure modes)
// can still run while emitting a warn-mode log entry.
var allowLongLocalActivity = testhelpers.WithActivityPolicySeverity(
	activitypolicy.LocalActivityStartToCloseTooLong,
	activitypolicy.SeverityWarn,
)

// @@@SNIPSTART fallible-local-activities-test

func TestLocalActivityRetriesExhausted(t *testing.T) {
	c, taskQueue := testhelpers.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(failingLocalActivityWorkflow)
		r.RegisterActivity(alwaysFails)
	}, allowLongLocalActivity)

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue:          taskQueue,
		WorkflowRunTimeout: time.Minute,
	}, failingLocalActivityWorkflow, "request")
	require.NoError(t, err)
	require.ErrorContains(t, run.Get(t.Context(), nil), "service unavailable")
}

func TestLocalActivityTooSlow(t *testing.T) {
	c, taskQueue := testhelpers.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(slowLocalActivityWorkflow)
		r.RegisterActivity(slowActivity)
	}, allowLongLocalActivity)

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue:          taskQueue,
		WorkflowRunTimeout: time.Minute,
	}, slowLocalActivityWorkflow, "request")
	require.NoError(t, err)
	require.ErrorContains(t, run.Get(t.Context(), nil), "deadline exceeded")
}

func TestLocalActivityGrowsHistory(t *testing.T) {
	c, taskQueue := testhelpers.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(failingLocalActivityWorkflow)
		r.RegisterActivity(alwaysFails)
	}, allowLongLocalActivity)

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue:           taskQueue,
		WorkflowRunTimeout:  time.Minute,
		WorkflowTaskTimeout: 200 * time.Millisecond,
	}, failingLocalActivityWorkflow, "request")
	require.NoError(t, err)

	// Wait for the workflow to complete (retries exhaust after 5 attempts).
	require.Error(t, run.Get(t.Context(), nil))

	// Count history events to prove local activity retries added events.
	iter := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false, 0)
	var eventCount int
	for iter.HasNext() {
		_, err := iter.Next()
		require.NoError(t, err)
		eventCount++
	}
	t.Logf("history event count: %d", eventCount)
	// A simple workflow with no retries would have ~5 events.
	// With server-deferred local activity retries, we expect significantly more.
	require.Greater(t, eventCount, 20, "local activity retries should grow the history")
}

// @@@SNIPEND
