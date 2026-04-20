package not_validating_replay_safety_before_deployments

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
)

// @@@SNIPSTART not-validating-replay-safety-before-deployments-test

func TestReplayWorkflowHistory(t *testing.T) {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(MyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/my_workflow_history.json")
	require.NoError(t, err)
}

// @@@SNIPEND
