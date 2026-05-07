package not_validating_replay_safety_before_deployments

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"
	"github.com/jlegrone/100-temporal-mistakes/internal/testhelpers"
)

// @@@SNIPSTART not-validating-replay-safety-before-deployments-test

func TestReplayWorkflowHistory(t *testing.T) {
	// MyWorkflow's activity options intentionally violate timeouts_permit_retries
	// (schedule_to_close = 1m, start_to_close = 30s, no heartbeat — ratio < 2.1x)
	// to keep the example minimal. Lower the policy severity to Warn so the
	// replay still sees the warn log but doesn't fail.
	require.NoError(t, testhelpers.ReplayWorkflowHistoryFromJSONFile(t, MyWorkflow,
		"testdata/my_workflow_history.json",
		testhelpers.WithActivityPolicySeverity(activitypolicy.TimeoutsPermitRetries, activitypolicy.SeverityWarn),
	))
}

// @@@SNIPEND
