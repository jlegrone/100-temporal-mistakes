package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-workflow-timeouts-test

func TestV2_CompletesWhenSignaled(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestV2_ContinuesAsNewOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	// Don't send the signal -- let the deadline fire.
	env.ExecuteWorkflow(MyWorkflowV2, 30*time.Minute)
	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	// The workflow calls ContinueAsNew when the deadline fires.
	var continueAsNewErr *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &continueAsNewErr)
}

// @@@SNIPEND
