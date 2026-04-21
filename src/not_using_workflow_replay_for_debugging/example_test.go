package not_using_workflow_replay_for_debugging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
)

// @@@SNIPSTART not-using-workflow-replay-for-debugging-good

func TestReplayWorkflow(t *testing.T) {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(MyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/history.json")
	require.NoError(t, err)
}

// @@@SNIPEND
