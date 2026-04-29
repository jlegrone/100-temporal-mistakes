package activitypolicy

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// ----------------------------------------------------------------------
// Unit tests: evaluatePolicies (no SDK runtime)
// ----------------------------------------------------------------------

func TestEvaluatePolicies(t *testing.T) {
	cases := map[string]struct {
		schedule, startTo, heartbeat time.Duration
		retry                        *temporal.RetryPolicy
		isLocal                      bool
		want                         []string
	}{
		"happy regular": {
			schedule: time.Hour, heartbeat: 30 * time.Second,
			want: nil,
		},
		"missing schedule_to_close": {
			startTo: 30 * time.Second,
			want:    []string{ScheduleToCloseRequired},
		},
		"max_attempts=2 violates": {
			schedule: time.Hour, startTo: 30 * time.Second,
			retry: &temporal.RetryPolicy{MaximumAttempts: 2},
			want:  []string{MaxAttemptsTooLow},
		},
		"max_attempts=3 allowed": {
			schedule: time.Hour, startTo: 30 * time.Second,
			retry: &temporal.RetryPolicy{MaximumAttempts: 3},
			want:  nil,
		},
		"max_attempts=0 allowed": {
			schedule: time.Hour, startTo: 30 * time.Second,
			retry: &temporal.RetryPolicy{MaximumAttempts: 0},
			want:  nil,
		},
		"timeouts_permit_retries violation no heartbeat": {
			schedule: 60 * time.Second, startTo: 30 * time.Second,
			want: []string{TimeoutsPermitRetries},
		},
		"timeouts_permit_retries skipped with heartbeat": {
			schedule: 60 * time.Second, startTo: 30 * time.Second, heartbeat: 10 * time.Second,
			want: nil,
		},
		"timeouts_permit_retries adequate ratio": {
			schedule: 70 * time.Second, startTo: 30 * time.Second,
			want: nil,
		},
		"local activity stc too long": {
			schedule: 60 * time.Second, startTo: 30 * time.Second, isLocal: true,
			want: []string{LocalActivityStartToCloseTooLong},
		},
		"local activity stc unset": {
			schedule: 60 * time.Second, isLocal: true,
			want: []string{LocalActivityStartToCloseTooLong},
		},
		"local activity happy": {
			schedule: 30 * time.Second, startTo: 5 * time.Second, isLocal: true,
			want: nil,
		},
		"multi-violation canonical order": {
			startTo: 30 * time.Second,
			retry:   &temporal.RetryPolicy{MaximumAttempts: 2},
			want: []string{
				ScheduleToCloseRequired,
				MaxAttemptsTooLow,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vs := evaluatePolicies(tc.schedule, tc.startTo, tc.heartbeat, tc.retry, tc.isLocal)
			var got []string
			for _, v := range vs {
				got = append(got, v.policy)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

// ----------------------------------------------------------------------
// Integration test fixtures
// ----------------------------------------------------------------------

func noopActivity() (string, error) {
	return "ok", nil
}

func happyRegularWorkflow(ctx workflow.Context) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Hour,
		HeartbeatTimeout:       30 * time.Second,
	})
	var out string
	err := workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, &out)
	return out, err
}

func happyLocalWorkflow(ctx workflow.Context) (string, error) {
	ctx = workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Second,
		StartToCloseTimeout:    5 * time.Second,
	})
	var out string
	err := workflow.ExecuteLocalActivity(ctx, noopActivity).Get(ctx, &out)
	return out, err
}

func missingScheduleToCloseWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})
	return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
}

func maxAttemptsTwoWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Hour,
		StartToCloseTimeout:    30 * time.Second,
		HeartbeatTimeout:       10 * time.Second,
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
}

func timeoutsPermitRetriesWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: 60 * time.Second,
		StartToCloseTimeout:    30 * time.Second,
	})
	return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
}

func localActivityTooLongWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 60 * time.Second,
		StartToCloseTimeout:    30 * time.Second,
	})
	return workflow.ExecuteLocalActivity(ctx, noopActivity).Get(ctx, nil)
}

func multiViolationWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		HeartbeatTimeout:    10 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
}

// ----------------------------------------------------------------------
// Integration test helpers
// ----------------------------------------------------------------------

func newTestEnv(t *testing.T, opts Options) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{New(opts)},
	})
	env.RegisterActivity(noopActivity)
	return env
}

func assertPolicyViolation(t *testing.T, err error, want ...string) *ViolationDetails {
	t.Helper()
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	assert.Equal(t, ViolationErrorType, appErr.Type())
	assert.True(t, appErr.NonRetryable(), "expected non-retryable error")

	var details ViolationDetails
	require.NoError(t, appErr.Details(&details))
	assert.Equal(t, want, details.Policies)
	return &details
}

// ----------------------------------------------------------------------
// Integration tests
// ----------------------------------------------------------------------

func TestIntegration_HappyRegular(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(happyRegularWorkflow)
	env.ExecuteWorkflow(happyRegularWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var out string
	require.NoError(t, env.GetWorkflowResult(&out))
	assert.Equal(t, "ok", out)
}

func TestIntegration_HappyLocal(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(happyLocalWorkflow)
	env.ExecuteWorkflow(happyLocalWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var out string
	require.NoError(t, env.GetWorkflowResult(&out))
	assert.Equal(t, "ok", out)
}

func TestIntegration_MissingScheduleToClose_Error(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(missingScheduleToCloseWorkflow)
	env.ExecuteWorkflow(missingScheduleToCloseWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	assertPolicyViolation(t, env.GetWorkflowError(), ScheduleToCloseRequired)
}

func TestIntegration_MissingScheduleToClose_Ignore(t *testing.T) {
	env := newTestEnv(t, Options{
		Severities: map[string]Severity{
			ScheduleToCloseRequired: SeverityIgnore,
		},
	})
	env.RegisterWorkflow(missingScheduleToCloseWorkflow)
	env.ExecuteWorkflow(missingScheduleToCloseWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestIntegration_MaxAttemptsTwo_Violation(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(maxAttemptsTwoWorkflow)
	env.ExecuteWorkflow(maxAttemptsTwoWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	assertPolicyViolation(t, env.GetWorkflowError(), MaxAttemptsTooLow)
}

func TestIntegration_TimeoutsPermitRetries_Violation(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(timeoutsPermitRetriesWorkflow)
	env.ExecuteWorkflow(timeoutsPermitRetriesWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	assertPolicyViolation(t, env.GetWorkflowError(), TimeoutsPermitRetries)
}

func TestIntegration_LocalActivityTooLong(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(localActivityTooLongWorkflow)
	env.ExecuteWorkflow(localActivityTooLongWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	assertPolicyViolation(t, env.GetWorkflowError(), LocalActivityStartToCloseTooLong)
}

func TestIntegration_MultiViolation_CanonicalOrder(t *testing.T) {
	env := newTestEnv(t, Options{})
	env.RegisterWorkflow(multiViolationWorkflow)
	env.ExecuteWorkflow(multiViolationWorkflow)
	require.True(t, env.IsWorkflowCompleted())
	assertPolicyViolation(t, env.GetWorkflowError(),
		ScheduleToCloseRequired,
		MaxAttemptsTooLow,
	)
}
