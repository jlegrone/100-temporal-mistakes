package deadlocking_when_workflow_canceled

import (
	"fmt"
	"testing"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
)

func TestV1_CompletesWhenSignaled(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV1)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestV1_DeadlocksOnCancelation(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV1)

	require.True(t, env.IsWorkflowCompleted())
	// V1 cannot handle cancelation -- the Receive blocks forever.
	// The test environment eventually times it out with the default
	// 10-year workflow execution timeout (ScheduleToClose), proving
	// the workflow didn't handle cancelation gracefully.
	var timeoutErr *temporal.TimeoutError
	require.ErrorAs(t, env.GetWorkflowError(), &timeoutErr)
	require.Equal(t, enumspb.TIMEOUT_TYPE_SCHEDULE_TO_CLOSE, timeoutErr.TimeoutType())
}

func TestV2_CompletesWhenSignaled(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("done", nil)
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2)

	require.True(t, env.IsWorkflowCompleted())
	// ctx.Err() is nil when not canceled, so no error.
	require.NoError(t, env.GetWorkflowError())
}

// @@@SNIPSTART deadlocking-canceled-test

func TestV2_HandlesGracefulCancelation(t *testing.T) {
	env := testsuite.NewTestWorkflowEnvironment(t)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(MyWorkflowV2)
	err := env.GetWorkflowError()

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err), fmt.Sprintf("Expected canceled error, got: %v", err))
}

// @@@SNIPEND
