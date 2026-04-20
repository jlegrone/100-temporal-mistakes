package deadlocking_when_workflow_cancelled

import (
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal"
	"github.com/stretchr/testify/require"
)

// @@@SNIPSTART deadlocking-cancelled-test

func TestV2_CompletesOnCancellation(t *testing.T) {
	env := internal.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV1)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	// require.ErrorAs(t, &temporal.CanceledError{}, env.GetWorkflowError())
}

// @@@SNIPEND
