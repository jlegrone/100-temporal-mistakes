package not_validating_replay_safety_before_deployments

import (
	"testing"

	"github.com/jlegrone/100-temporal-mistakes/internal/testhelpers"
	"github.com/stretchr/testify/require"
)

// @@@SNIPSTART not-validating-replay-safety-before-deployments-test

func TestReplayWorkflowHistory(t *testing.T) {
	require.NoError(t, testhelpers.ReplayWorkflowHistoryFromJSONFile(t, MyWorkflow, "testdata/my_workflow_history.json"))
}

// @@@SNIPEND
