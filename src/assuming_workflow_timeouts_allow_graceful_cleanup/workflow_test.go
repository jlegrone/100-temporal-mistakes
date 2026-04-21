package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"errors"
	"testing"
	"time"

	internaltestsuite "github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-workflow-timeouts-test

func TestMyWorkflowV2(t *testing.T) {
	type testCase struct {
		runTimeout    time.Duration
		setup         func(env *testsuite.TestWorkflowEnvironment)
		expectedError error
	}

	tests := map[string]testCase{
		"completes when child finishes": {
			// Timeout longer than the child's 1h sleep, so the child completes first.
			runTimeout: 2 * time.Hour,
		},
		"compensates on deadline": {
			// Timeout shorter than the child's 1h sleep, so the soft deadline fires first.
			runTimeout:    10 * time.Minute,
			expectedError: workflow.ErrDeadlineExceeded,
		},
		"compensates on cancelation": {
			runTimeout: 10 * time.Minute,
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterDelayedCallback(func() {
					env.CancelWorkflow()
				}, time.Second)
			},
			expectedError: &temporal.CanceledError{},
		},
		"fails on too-short timeout": {
			// Run timeout shorter than the padding (1m), so getSoftTimeout
			// returns an error before any work starts.
			runTimeout:    30 * time.Second,
			expectedError: &temporal.ApplicationError{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := internaltestsuite.NewTestWorkflowEnvironment(t)
			env.RegisterWorkflow(LongRunningWorkflow)
			env.SetWorkflowRunTimeout(tc.runTimeout)

			compensated := false
			env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Maybe().Run(
				func(args mock.Arguments) { compensated = true },
			)

			if tc.setup != nil {
				tc.setup(env)
			}

			env.ExecuteWorkflow(MyWorkflowV2)
			require.True(t, env.IsWorkflowCompleted())

			if tc.expectedError != nil {
				require.ErrorAs(t, env.GetWorkflowError(), &tc.expectedError)
			} else {
				require.NoError(t, env.GetWorkflowError())
			}
			// Compensation runs for operational errors (deadline, cancelation)
			// but not for configuration errors (too-short timeout).
			var appErr *temporal.ApplicationError
			wantCompensate := tc.expectedError != nil && !errors.As(tc.expectedError, &appErr)
			require.Equal(t, wantCompensate, compensated, "CompensateActivity called")
		})
	}
}

// @@@SNIPEND
