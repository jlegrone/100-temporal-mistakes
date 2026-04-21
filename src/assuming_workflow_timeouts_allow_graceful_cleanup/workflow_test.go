package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"fmt"
	"testing"
	"time"

	internaltestsuite "github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestV1_CompensationNeverRuns(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(LongRunningWorkflow)
	env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Maybe()

	// The child sleeps for 1h. With a 10m run timeout, the workflow
	// is terminated when the timeout fires -- not canceled.
	// V1's compensation code is unreachable.
	env.SetWorkflowRunTimeout(10 * time.Minute)

	env.ExecuteWorkflow(MyWorkflowV1)
	require.True(t, env.IsWorkflowCompleted())
	require.ErrorContains(t, env.GetWorkflowError(), "deadline exceeded")

	// This assertion demonstrates the incorrect behavior; we WANT to run
	// the compensating activity before the workflow times out.
	env.AssertActivityNotCalled(t, "CompensateActivity", mock.Anything)
}

func TestMyWorkflowV2(t *testing.T) {
	type testCase struct {
		runTimeout     time.Duration
		setup          func(env *testsuite.TestWorkflowEnvironment)
		expectedError  string
		wantCompensate bool
	}

	tests := map[string]testCase{
		"completes when child finishes": {
			runTimeout: 2 * time.Hour,
		},
		"compensates on deadline": {
			runTimeout:     10 * time.Minute,
			expectedError:  "deadline exceeded",
			wantCompensate: true,
		},
		"compensates on cancelation": {
			runTimeout: 10 * time.Minute,
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterDelayedCallback(func() {
					env.CancelWorkflow()
				}, time.Second)
			},
			expectedError:  "canceled",
			wantCompensate: true,
		},
		"compensates on child error": {
			runTimeout: 2 * time.Hour,
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(LongRunningWorkflow, mock.Anything).Return(
					fmt.Errorf("child failed"),
				)
			},
			expectedError:  "child failed",
			wantCompensate: true,
		},
		"fails on too-short timeout": {
			runTimeout:     30 * time.Second,
			expectedError:  "workflow timeout is too small",
			wantCompensate: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := internaltestsuite.NewTestWorkflowEnvironment(t)
			env.RegisterWorkflow(LongRunningWorkflow)
			env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Maybe()

			env.SetWorkflowRunTimeout(tc.runTimeout)
			if tc.setup != nil {
				tc.setup(env)
			}

			env.ExecuteWorkflow(MyWorkflowV2)
			require.True(t, env.IsWorkflowCompleted())

			if tc.expectedError != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.expectedError)
			} else {
				require.NoError(t, env.GetWorkflowError())
			}
			if tc.wantCompensate {
				env.AssertActivityCalled(t, "CompensateActivity", mock.Anything)
			} else {
				env.AssertActivityNotCalled(t, "CompensateActivity", mock.Anything)
			}
		})
	}
}
