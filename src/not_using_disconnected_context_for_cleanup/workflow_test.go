package not_using_disconnected_context_for_cleanup

import (
	"testing"
	"time"

	internaltestsuite "github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// @@@SNIPSTART not-using-disconnected-context-test

func TestV1_CleanupNeverRuns(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	// ProcessOrder blocks for 1h, so the cancel at 1s interrupts it.
	env.OnActivity(ProcessOrder, mock.Anything).After(time.Hour).Return(nil)
	env.OnActivity(CancelOrder, mock.Anything).Return(nil).Maybe()

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV1)
	require.True(t, env.IsWorkflowCompleted())
	require.True(t, temporal.IsCanceledError(env.GetWorkflowError()))

	// V1 uses the canceled context for cleanup, so CancelOrder never runs.
	env.AssertActivityNotCalled(t, "CancelOrder", mock.Anything)
}

func TestV2_CleanupRuns(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	env.OnActivity(ProcessOrder, mock.Anything).After(time.Hour).Return(nil)
	env.OnActivity(CancelOrder, mock.Anything).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2)
	require.True(t, env.IsWorkflowCompleted())
	require.True(t, temporal.IsCanceledError(env.GetWorkflowError()))

	// V2 uses a disconnected context, so CancelOrder runs.
	env.AssertActivityCalled(t, "CancelOrder", mock.Anything)
}

// @@@SNIPEND
