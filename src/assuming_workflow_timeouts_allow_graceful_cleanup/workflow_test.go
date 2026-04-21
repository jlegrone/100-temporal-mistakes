package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// @@@SNIPSTART assuming-workflow-timeouts-test

func TestV2_CompensatesOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Track whether CompensateActivity was called.
	compensated := false
	env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Run(
		func(args mock.Arguments) { compensated = true },
	)

	// Set a run timeout so getSoftTimeout can derive the internal deadline.
	// The child sleeps for 1h; with a 24h10m run timeout and 24h padding,
	// the soft timeout is 10m -- shorter than the child, so the deadline
	// fires first.
	env.SetWorkflowRunTimeout(24*time.Hour + 10*time.Minute)

	env.ExecuteWorkflow(MyWorkflowV2)
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.True(t, compensated, "CompensateActivity should have been called")
}

// @@@SNIPEND
