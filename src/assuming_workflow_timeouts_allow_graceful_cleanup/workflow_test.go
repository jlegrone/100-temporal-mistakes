package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"testing"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-workflow-timeouts-test

func TestV2_ContinuesAsNewOnDeadline(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)

	// The child workflow sleeps for 24h, but the internal deadline
	// is 30 minutes, so the deadline fires first and the workflow
	// gracefully continues as new instead of being terminated.
	env.ExecuteWorkflow(MyWorkflowV2)
	require.True(t, env.IsWorkflowCompleted())
	var continueAsNewErr *workflow.ContinueAsNewError
	require.ErrorAs(t, env.GetWorkflowError(), &continueAsNewErr)
}

// @@@SNIPEND
