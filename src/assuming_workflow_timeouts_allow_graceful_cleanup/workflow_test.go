package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-workflow-timeouts-test

func TestV2_CompletesWhenChildFinishes(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Signal the child workflow to complete before the deadline.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflowByID("long-running", "done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestV2_ContinuesAsNewOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// Don't signal the child -- let the deadline fire first.
	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	var continueAsNewErr *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &continueAsNewErr)
}

// @@@SNIPEND
